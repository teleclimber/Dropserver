package appspacerouter

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// handles /dropserver/ routes of an app-space

// V0DropserverRoutes represents struct for /dropserver appspace routes
type V0DropserverRoutes struct {
	Authenticator interface {
		Unset(w http.ResponseWriter, r *http.Request)
	} `checkinject:"required"`
}

func (d *V0DropserverRoutes) subRouter() http.Handler {
	mux := chi.NewRouter()

	mux.Get("/logout", d.logout)

	return mux
}

func (d *V0DropserverRoutes) logout(w http.ResponseWriter, r *http.Request) {
	d.Authenticator.Unset(w, r)
	if !strings.Contains(r.Header.Get("accept"), "text/html") {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>Logged out</h1>"))
}

func (d *V0DropserverRoutes) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0DropserverRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
