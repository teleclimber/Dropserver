package testmocks

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=auth_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks Authenticator,V0TokenManager

// Authenticator is an interface that can set and authenticate cookies
// And in the future it will handle other forms of authentication
type Authenticator interface {
	SetForAccount(http.ResponseWriter, domain.UserID) error
	SetForAppspace(http.ResponseWriter, domain.ProxyID, domain.AppspaceID, string) (string, error)
	Unset(http.ResponseWriter, *http.Request)
	AppspaceUserProxyID(http.Handler) http.Handler
}

// V0TokenManager tracks and returns appspace login tokens
type V0TokenManager interface {
	GetForProxyID(appspaceID domain.AppspaceID, proxyID domain.ProxyID) string
	CheckToken(appspaceID domain.AppspaceID, token string) (domain.V0AppspaceLoginToken, bool)
}
