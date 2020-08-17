package testmocks

import (
	"net/http"
	"net/url"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=auth_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks Authenticator,AppspaceLogin

// Authenticator is an interface that can set and authenticate cookies
// And in the future it will handle other forms of authentication
type Authenticator interface {
	Authenticate(http.ResponseWriter, *http.Request) (*domain.Authentication, error)
	SetForAccount(http.ResponseWriter, domain.UserID) error
	SetForAppspace(http.ResponseWriter, domain.UserID, domain.AppspaceID, string) (string, error)
	UnsetForAccount(http.ResponseWriter, *http.Request)
}

// AppspaceLogin tracks and returns appspace login tokens
type AppspaceLogin interface {
	Create(appspaceID domain.AppspaceID, appspaceURL url.URL) domain.AppspaceLoginToken
	LogIn(loginToken string, userID domain.UserID) (domain.AppspaceLoginToken, error)
	CheckRedirectToken(redirectToken string) (domain.AppspaceLoginToken, error)
}
