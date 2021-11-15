package main

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// DevAuthenticator authenticates all requests
type DevAuthenticator struct {
	noAuth bool
	auth   domain.Authentication
}

// SetNoAuth makes Authenticator return nil on auth
func (a *DevAuthenticator) SetNoAuth() {
	a.auth = domain.Authentication{}
	a.noAuth = true
}

// Set an authentication that will be returned on Authenticate calls
func (a *DevAuthenticator) Set(auth domain.Authentication) {
	a.noAuth = false
	a.auth = auth
}

// SetForAppspace is a noop, returns empty string
func (a *DevAuthenticator) SetForAppspace(http.ResponseWriter, domain.ProxyID, domain.AppspaceID, string) (string, error) {
	// I don't think this should ever be used
	return "", nil
}

// AppspaceUserProxyID middleware sets the proxy ID for
// the user authenticated for the requested appspace
func (a *DevAuthenticator) AppspaceUserProxyID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.noAuth {
			ctx := domain.CtxWithAppspaceUserProxyID(r.Context(), a.auth.ProxyID)
			ctx = domain.CtxWithSessionID(ctx, a.auth.CookieID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func (a *DevAuthenticator) Unset(w http.ResponseWriter, r *http.Request) {
	a.SetNoAuth()
	// This implies user clicked on "logout" in appspace.
	// This should be reflected in ds-dev frontend
}

func (a *DevAuthenticator) GetProxyID() (domain.ProxyID, bool) {
	if !a.auth.Authenticated {
		return domain.ProxyID(""), false
	}
	return a.auth.ProxyID, true
}
