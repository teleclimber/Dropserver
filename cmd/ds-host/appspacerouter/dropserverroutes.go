package appspacerouter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type DropserverRoutes struct {
	V0DropServerRoutes interface {
		subRouter() http.Handler
	} `checkinject:"required"`
}

// this needs to be subroutes

func (d *DropserverRoutes) Router() http.Handler {
	mux := chi.NewRouter()

	mux.Get("/apiversions", d.apiVersions)
	mux.Mount("/v0", d.V0DropServerRoutes.subRouter())

	return mux
}

func (d *DropserverRoutes) apiVersions(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "api check not implemented", http.StatusNotImplemented)
}
