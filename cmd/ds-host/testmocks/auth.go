package testmocks

import (
	"context"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=auth_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks Authenticator,V0TokenManager,V0RequestToken,DS2DS

// Authenticator is an interface that can set and authenticate cookies
// And in the future it will handle other forms of authentication
type Authenticator interface {
	Authenticate(*http.Request) domain.Authentication
	SetForAccount(http.ResponseWriter, domain.UserID) error
	SetForAppspace(http.ResponseWriter, domain.ProxyID, domain.AppspaceID, string) (string, error)
	UnsetForAccount(http.ResponseWriter, *http.Request)
}

// V0TokenManager tracks and returns appspace login tokens
type V0TokenManager interface {
	GetForOwner(appspaceID domain.AppspaceID, dropID string) string
	CheckToken(token string) (domain.V0AppspaceLoginToken, bool)
	SendLoginToken(appspaceID domain.AppspaceID, dropID string, ref string) error
}

// V0RequestToken manages requests for login tokens from remote hosts
type V0RequestToken interface {
	RequestToken(ctx context.Context, userID domain.UserID, appspaceDomain string, sessionID string) (string, error)
	ReceiveToken(ref, token string)
	ReceiveError(ref string, err error)
}

// DS2DS is a helper for ds-host to ds-host communications
type DS2DS interface {
	GetRemoteAPIVersion(domainName string) (int, error)
}
