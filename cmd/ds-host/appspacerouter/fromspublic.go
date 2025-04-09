package appspacerouter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/getcleanhost"
)

type FromPublic struct {
	Authenticator interface {
		AppspaceUserProxyID(http.Handler) http.Handler
		SetForAppspace(http.ResponseWriter, domain.ProxyID, domain.AppspaceID, string) (string, error)
	} `checkinject:"required"`
	V0TokenManager interface {
		CheckToken(appspaceID domain.AppspaceID, token string) (domain.V0AppspaceLoginToken, bool)
	} `checkinject:"required"`
	AppspaceModel interface {
		GetFromDomain(string) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceRouter interface {
		BuildRoutes(mux *chi.Mux)
	} `checkinject:"required"`

	mux *chi.Mux
}

func (f *FromPublic) Init() {
	f.mux = chi.NewRouter()
	f.mux.Use(f.loadAppspace)
	f.mux.Use(f.Authenticator.AppspaceUserProxyID, f.processLoginToken)
	f.AppspaceRouter.BuildRoutes(f.mux)
}

func (f *FromPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mux.ServeHTTP(w, r)
}

func (f *FromPublic) loadAppspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: use of r.Host not good enough. see the requestHost function of https://github.com/go-chi/hostrouter
		// May need to determine host at server and stash it in context.
		host, err := getcleanhost.GetCleanHost(r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		appspace, err := f.AppspaceModel.GetFromDomain(host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if appspace == nil {
			w.WriteHeader(http.StatusNotFound)
			notFoundPage(w)
			return
		}

		r = r.WithContext(domain.CtxWithAppspaceData(r.Context(), *appspace))

		next.ServeHTTP(w, r)
	})
}

func (f *FromPublic) processLoginToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginTokenValues := r.URL.Query()["dropserver-login-token"]
		if len(loginTokenValues) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		if len(loginTokenValues) > 1 {
			http.Error(w, "multiple login tokens", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)

		token, ok := f.V0TokenManager.CheckToken(appspace.AppspaceID, loginTokenValues[0])
		if !ok {
			// no matching token is not an error. It can happen if user reloads the page for ex.
			next.ServeHTTP(w, r)
			return
		}

		cookieID, err := f.Authenticator.SetForAppspace(w, token.ProxyID, token.AppspaceID, appspace.DomainName)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		ctx = domain.CtxWithAppspaceUserProxyID(ctx, token.ProxyID)
		ctx = domain.CtxWithSessionID(ctx, cookieID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
