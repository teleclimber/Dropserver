package main

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	dsappdevfrontend "github.com/teleclimber/DropServer/frontend-ds-dev"
)

// in ds-dev, server assumes all paths are appspace paths
// ..unless they are /dropserver/ or /dropserver-dev/
// Where /dropserver is the normal ds appspace api
// and /dropserver-dev/ is the control panel for ds-dev.

// Server struct sets all parameters about the server
type Server struct {
	Config               *domain.RuntimeConfig `checkinject:"required"`
	DropserverDevHandler *DropserverDevServer  `checkinject:"required"`
	AppspaceRouter       http.Handler          `checkinject:"required"`
}

// Start starts up the server so it listens for connections
func (s *Server) Start() { //return a server type
	frontendFS, fserr := fs.Sub(dsappdevfrontend.FS, "dist")
	if fserr != nil {
		panic(fserr)
	}
	avatarsSubFS, fserr := fs.Sub(avatarsFS, "avatars")
	if fserr != nil {
		panic(fserr)
	}

	r := chi.NewRouter()
	r.Route("/dropserver-dev", func(r chi.Router) {
		r.Get("/base-data", s.DropserverDevHandler.GetBaseData)
		r.Get("/livedata", s.DropserverDevHandler.StartLivedata)

		r.Get("/avatar/baked-in", s.getAvatarList)
		r.Handle("/avatar/baked-in/*", http.StripPrefix("/dropserver-dev/avatar/baked-in/", http.FileServer(http.FS(avatarsSubFS))))
		r.Handle("/avatar/appspace/*", http.StripPrefix("/dropserver-dev/avatar/appspace/", http.FileServer(http.Dir(filepath.Join(s.Config.Exec.AppspacesPath, "data", "avatars")))))

		r.Get("/", s.serveAppIndex)
		r.Handle("/*", http.StripPrefix("/dropserver-dev/", http.FileServer(http.FS(frontendFS))))
	})
	r.Handle("/*", s.AppspaceRouter)

	cfg := s.Config.Server
	addr := ":" + strconv.FormatInt(int64(cfg.Port), 10)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		os.Exit(1)
	}
}

func (s *Server) serveAppIndex(w http.ResponseWriter, r *http.Request) {
	htmlBytes, err := dsappdevfrontend.FS.ReadFile("dist/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlBytes)
}

func (s *Server) getAvatarList(w http.ResponseWriter, r *http.Request) {
	dirs, err := avatarsFS.ReadDir("avatars")
	if err != nil {
		panic(err)
	}

	respData := make([]string, len(dirs))
	for i, d := range dirs {
		respData[i] = d.Name()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respData)
}
