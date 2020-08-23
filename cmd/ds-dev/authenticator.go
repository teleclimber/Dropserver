package main

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// DevAuthenticator authenticates all requests
type DevAuthenticator struct {
	auth domain.Authentication
}

// Set an authentication that will be returned on Authenticate calls
func (a *DevAuthenticator) Set(auth domain.Authentication) {
	a.auth = auth
}

// Authenticate returns an auth in all cases
func (a *DevAuthenticator) Authenticate(http.ResponseWriter, *http.Request) (*domain.Authentication, error) {
	return &a.auth, nil
}

// SetForAppspace is a noop, returns empty string
func (a *DevAuthenticator) SetForAppspace(http.ResponseWriter, domain.UserID, domain.AppspaceID, string) (string, error) {
	// I don't think this should ever be used
	return "", nil
}
