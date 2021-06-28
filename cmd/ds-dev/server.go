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
	Config               *domain.RuntimeConfig `checkinject:"required"`
	DropserverDevHandler http.Handler          `checkinject:"required"`
	AppspaceRouter       http.Handler          `checkinject:"required"`
}

// Start starts up the server so it listens for connections
func (s *Server) Start() { //return a server type
	cfg := s.Config.Server

	http.Handle("/", s)

	addr := ":" + strconv.FormatInt(int64(cfg.Port), 10)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		os.Exit(1)
	}
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	head, tail := shiftpath.ShiftPath(req.URL.Path)
	switch head {
	case "dropserver":
		http.Error(res, "not implemented yet", http.StatusNotImplemented)
	case "dropserver-dev":
		req.URL.Path = tail // shouldnt' modify request
		s.DropserverDevHandler.ServeHTTP(res, req)
	default:
		s.AppspaceRouter.ServeHTTP(res, req)
	}
}
