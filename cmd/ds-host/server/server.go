package server

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	dshostfrontend "github.com/teleclimber/DropServer/frontend-ds-host"
	"github.com/teleclimber/DropServer/internal/getcleanhost"
)

// Server struct sets all parameters about the server
type Server struct {
	Config        *domain.RuntimeConfig
	Authenticator interface {
		Authenticate(*http.Request) domain.Authentication
	}
	Views interface {
		GetStaticFS() fs.FS
	}

	// admin routes, user routes, auth routes....
	UserRoutes     domain.RouteHandler
	AppspaceRouter domain.RouteHandler

	mux    *http.ServeMux
	server *http.Server
}

func (s *Server) Init() {
	s.mux = http.NewServeMux()

	s.mux.Handle(
		s.Config.Exec.UserRoutesDomain+"/static/",
		http.StripPrefix("/static/", http.FileServer(http.FS(s.Views.GetStaticFS()))))

	frontendFS, fserr := fs.Sub(dshostfrontend.FS, "dist")
	if fserr != nil {
		panic(fserr)
	}
	s.mux.Handle(
		s.Config.Exec.UserRoutesDomain+"/frontend-assets/",
		http.FileServer(http.FS(frontendFS)))

	// Everything else:
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	})
}

// Start up the server so it listens for connections
func (s *Server) Start() { //return a server type
	cfg := s.Config.Server

	addr := ":" + strconv.FormatInt(int64(cfg.Port), 10)

	s.getLogger("Start()").Debug("User Routes address: " + s.Config.Exec.UserRoutesDomain + s.Config.Exec.PortString)

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.mux}

	var err error
	if s.Config.Server.NoSsl {
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
		panic(err) // for now.
	}
}

// needed server graceful shutdown
// func (s *Server) Start() {
// }

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// Can we have some application-global iddlewares at work here?
	// Like CSRF? -> Any POST, PUT, PATCH, .. gets checked for a CSRF token?
	// I guess the middleware would have same signature as others, and include reouteData
	//

	// temporary CORS header to allow frontend dev.
	// TODO: Make this a config option!
	res.Header().Set("Access-Control-Allow-Origin", "*")

	// switch on top level routes:
	// - admin
	// - user
	// - auth
	// - appspace...

	auth := s.Authenticator.Authenticate(req)

	routeData := &domain.AppspaceRouteData{ //curently using AppspaceRouteData for user routes as well
		URLTail:        req.URL.Path,
		Authentication: &auth}

	host, err := getcleanhost.GetCleanHost(req.Host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	switch host {
	case s.Config.Exec.UserRoutesDomain:
		s.UserRoutes.ServeHTTP(res, req, routeData)
	default:
		// It's an appspace subdomain
		// first filter through blacklist of domains
		s.AppspaceRouter.ServeHTTP(res, req, routeData)
	}
}

func (s *Server) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("Server")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
