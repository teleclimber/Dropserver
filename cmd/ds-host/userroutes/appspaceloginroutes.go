package userroutes

import (
	"context"
	"net/http"
	"net/url"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/validator"
)

// AppspaceLoginRoutes handle
type AppspaceLoginRoutes struct {
	Config        *domain.RuntimeConfig
	AppspaceModel interface {
		GetFromDomain(dom string) (*domain.Appspace, error)
	}
	RemoteAppspaceModel interface {
		Get(userID domain.UserID, domainName string) (domain.RemoteAppspace, error)
	}
	DS2DS interface {
		GetRemoteAPIVersion(domainName string) (int, error)
	}
	V0RequestToken interface {
		RequestToken(ctx context.Context, userID domain.UserID, appspaceDomain string, sessionID string) (string, error)
	}
	V0TokenManager interface {
		GetForOwner(appspaceID domain.AppspaceID, dropID string) (string, error)
	}
}

func (u *AppspaceLoginRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// user must absolutely be logged in at this point.

	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	switch head {
	case "":
		u.getTokenForRedirect(res, req, routeData)
	default:
		returnError(res, errNotFound)
	}
}

func (u *AppspaceLoginRoutes) getTokenForRedirect(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	query := req.URL.Query()
	appspaceDomains, ok := query["appspace"]
	if !ok || len(appspaceDomains) != 1 {
		http.Error(res, "Missing or malformed appspace domain query parameter", http.StatusBadRequest)
		return
	}
	appspaceDomain, err := url.QueryUnescape(appspaceDomains[0])
	if err != nil {
		http.Error(res, "Malformed appspace domain query parameter", http.StatusBadRequest)
		return
	}
	err = validator.DomainName(appspaceDomain)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	appspaceDomain = validator.NormalizeDomainName(appspaceDomain)

	appspace, err := u.AppspaceModel.GetFromDomain(appspaceDomain)
	if err != nil {
		// some day appspace model will return an error if appspace not found.
		// handle that appropriately.
		returnError(res, err)
		return
	}
	if appspace != nil && appspace.OwnerID == routeData.Authentication.UserID {
		// Found an appspace owned by the user requesting a token.
		// We're assuming the appspace owner is a user.
		// This is handled differently from "remote" appspaces because
		// there is no entry for this appspace in owner's "remote" appspaces.
		token, err := u.V0TokenManager.GetForOwner(appspace.AppspaceID, appspace.DropID)
		if err != nil {
			returnError(res, err)
			return
		}
		http.Redirect(res, req, u.makeRedirectLink(appspaceDomain, token), http.StatusTemporaryRedirect)
		return
	}

	ver, err := u.DS2DS.GetRemoteAPIVersion(appspaceDomain)
	if err != nil {
		// we'll get more detailed with errors later...
		// it could be we couldn't reach teh remote server, or got an http error or something
		// it could be there is no common API version we can use
		http.Error(res, "error determining remote API version: "+err.Error(), http.StatusInternalServerError)
		return
	}

	switch ver {
	case 0:
		loginToken, err := u.V0RequestToken.RequestToken(req.Context(), routeData.Authentication.UserID, appspaceDomain, routeData.Authentication.CookieID)
		if err != nil {
			// we need to get subtle with errors, probably have a whole set of sentinel errors?
			res.WriteHeader(http.StatusServiceUnavailable) //for now
			res.Write([]byte(err.Error()))
			return
		}
		http.Redirect(res, req, u.makeRedirectLink(appspaceDomain, loginToken), http.StatusTemporaryRedirect)

	default:
		// log this. This should not happen.
		http.Error(res, "remote API version missing", http.StatusInternalServerError)
	}
}

func (u *AppspaceLoginRoutes) makeRedirectLink(appspaceDomain, token string) string {
	query := make(url.Values)
	query.Add("dropserver-login-token", token)

	return "https://" + appspaceDomain + u.Config.Exec.PortString + "?" + query.Encode()
}
