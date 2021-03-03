package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// Server struct sets all parameters about the server
type Server struct {
	Config        *domain.RuntimeConfig
	Authenticator interface {
		Authenticate(*http.Request) domain.Authentication
	}

	// admin routes, user routes, auth routes....
	UserRoutes     domain.RouteHandler
	AppspaceRouter domain.RouteHandler

	Metrics domain.MetricsI

	publicStaticHandler http.Handler
}

// Start starts up the server so it listens for connections
func (s *Server) Start() { //return a server type
	s.publicStaticHandler = http.FileServer(http.Dir(s.Config.Exec.StaticAssetsDir))

	cfg := s.Config.Server

	// Proxy:
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	})

	addr := ":" + strconv.FormatInt(int64(cfg.Port), 10)

	fmt.Println("Server listening on port " + addr)
	fmt.Println("Static Assets domain: " + s.Config.Exec.PublicStaticDomain)
	fmt.Println("User Routes domain: " + s.Config.Exec.UserRoutesDomain)

	var err error
	if s.Config.Server.NoSsl {
		err = http.ListenAndServe(addr, nil)
	} else {
		err = http.ListenAndServeTLS(addr, s.Config.Server.SslCert, s.Config.Server.SslKey, nil)
	}
	if err != nil {
		s.getLogger("Start(), http.ListenAndServe[TLS]").Error(err)
		os.Exit(1)
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

	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
	}
	host = strings.ToLower(host)

	fmt.Println(host, req.URL)

	switch host {
	case s.Config.Exec.PublicStaticDomain:
		s.publicStaticHandler.ServeHTTP(res, req)
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
