package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type DevAppspaceRouter struct {
	Authenticator interface {
		AppspaceUserProxyID(http.Handler) http.Handler
		SetForAppspace(http.ResponseWriter, domain.ProxyID, domain.AppspaceID, string) (string, error)
	} `checkinject:"required"`
	AppspaceModel interface {
		GetFromDomain(string) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceRouter interface {
		BuildRoutes(mux *chi.Mux)
	} `checkinject:"required"`

	mux *chi.Mux
}

func (d *DevAppspaceRouter) Init() {
	d.mux = chi.NewRouter()
	d.mux.Use(d.loadAppspace)
	d.mux.Use(d.Authenticator.AppspaceUserProxyID)
	d.AppspaceRouter.BuildRoutes(d.mux)
}

func (d *DevAppspaceRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mux.ServeHTTP(w, r)
}

func (d *DevAppspaceRouter) loadAppspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appspace, _ := d.AppspaceModel.GetFromDomain("")
		r = r.WithContext(domain.CtxWithAppspaceData(r.Context(), *appspace))
		next.ServeHTTP(w, r)
	})
}
