package userroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type FromPublic struct {
	Authenticator interface {
		AccountUser(http.Handler) http.Handler
	} `checkinject:"required"`
	AuthRoutes routeGroup `checkinject:"required"`
	UserRoutes interface {
		BuildRoutes(mux *chi.Mux)
	} `checkinject:"required"`

	mux *chi.Mux
}

func (f *FromPublic) Init() {
	r := chi.NewRouter()
	r.Use(addCSPHeaders)
	r.Use(f.Authenticator.AccountUser)
	r.Group(f.AuthRoutes.routeGroup)
	f.UserRoutes.BuildRoutes(r)
	f.mux = r
}

func (f *FromPublic) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	f.mux.ServeHTTP(res, req)
}
