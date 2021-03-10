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
	a.noAuth = true
}

// Set an authentication that will be returned on Authenticate calls
func (a *DevAuthenticator) Set(auth domain.Authentication) {
	a.noAuth = false
	a.auth = auth
}

// Authenticate returns an auth in all cases
func (a *DevAuthenticator) Authenticate(http.ResponseWriter, *http.Request) (*domain.Authentication, error) {
	if a.noAuth {
		return nil, nil
	}
	return &a.auth, nil
}

// SetForAppspace is a noop, returns empty string
func (a *DevAuthenticator) SetForAppspace(http.ResponseWriter, domain.ProxyID, domain.AppspaceID, string) (string, error) {
	// I don't think this should ever be used
	return "", nil
}
