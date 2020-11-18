package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// in ds-dev, server assumes all paths are appspace paths
// ..unless they are /dropserver/ or /dropserver-dev/
// Where /dropserver is the normal ds appspace api
// and /dropserver-dev/ is the control panel for ds-dev.

// Server struct sets all parameters about the server
type Server struct {
	Config        *domain.RuntimeConfig
	Authenticator interface {
		Authenticate(http.ResponseWriter, *http.Request) (*domain.Authentication, error)
	}
	DropserverDevHandler http.Handler

	AppspaceRouter domain.RouteHandler
}

// Start starts up the server so it listens for connections
func (s *Server) Start() { //return a server type
	cfg := s.Config.Server

	// Proxy:
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	})

	addr := ":" + strconv.FormatInt(int64(cfg.Port), 10)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		//s.getLogger("Start(), http.ListenAndServeTLS").Error(err)
		os.Exit(1)
	}
}

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

	auth, err := s.Authenticator.Authenticate(res, req)
	if err != nil {
		http.Error(res, "authentication error", http.StatusInternalServerError)
		return
	}

	routeData := &domain.AppspaceRouteData{ //curently using AppspaceRouteData for user routes as well
		URLTail:        req.URL.Path,
		Subdomains:     &[]string{"abc"},
		Authentication: auth}

	head, tail := shiftpath.ShiftPath(req.URL.Path)
	switch head {
	case "dropserver":
		http.Error(res, "not implemented yet", http.StatusNotImplemented)
	case "dropserver-dev":
		req.URL.Path = tail // shouldnt' modify request
		s.DropserverDevHandler.ServeHTTP(res, req)
	default:
		s.AppspaceRouter.ServeHTTP(res, req, routeData)
	}
}
