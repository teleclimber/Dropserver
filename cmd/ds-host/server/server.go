package server

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/getcleanhost"
)

// Server struct sets all parameters about the server
type Server struct {
	Config *domain.RuntimeConfig `checkinject:"required"`

	// admin routes, user routes, auth routes....
	UserRoutes     http.Handler `checkinject:"required"`
	AppspaceRouter http.Handler `checkinject:"required"`

	server *http.Server
}

func (s *Server) Init() {

}

// Start up the server so it listens for connections
func (s *Server) Start() { //return a server type
	cfg := s.Config.Server

	addr := ":" + strconv.FormatInt(int64(cfg.Port), 10)

	s.getLogger("Start()").Debug("User Routes address: " + s.Config.Exec.UserRoutesDomain + s.Config.PortString)

	s.server = &http.Server{
		Addr:    addr,
		Handler: s} // for now we're passing s directly, but in future probably need to wrap s in some middlewares and pass that in

	var err error
	if s.Config.Server.SslCert == "" {
		err = s.server.ListenAndServe()
	} else {
		err = s.server.ListenAndServeTLS(s.Config.Server.SslCert, s.Config.Server.SslKey)
	}
	if err != nil {
		s.getLogger("Start(), http.ListenAndServe[TLS]").Error(err)
		os.Exit(1)
	}
}

func (s *Server) Shutdown() {
	if s.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		panic(err) // for now.// log it
	}
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	host, err := getcleanhost.GetCleanHost(req.Host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// This is going to need to account for dropid domains
	// which may double as appspace domains and user domains?

	switch host {
	case s.Config.Exec.UserRoutesDomain:
		s.UserRoutes.ServeHTTP(res, req)
	default:
		// It's an appspace subdomain
		// first filter through blacklist of domains
		s.AppspaceRouter.ServeHTTP(res, req)
	}
}

func (s *Server) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("Server")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
