package userroutes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

// AppspaceLoginRoutes handle
type AppspaceLoginRoutes struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	AppspaceModel interface {
		GetFromDomain(dom string) (*domain.Appspace, error)
	} `checkinject:"required"`
	RemoteAppspaceModel interface {
		Get(userID domain.UserID, domainName string) (domain.RemoteAppspace, error)
	} `checkinject:"required"`
	DS2DS interface {
		GetRemoteAPIVersion(domainName string) (int, error)
	} `checkinject:"required"`
	V0RequestToken interface {
		RequestToken(ctx context.Context, userID domain.UserID, appspaceDomain string, sessionID string) (string, error)
	} `checkinject:"required"`
	V0TokenManager interface {
		GetForOwner(appspaceID domain.AppspaceID, dropID string) (string, error)
	} `checkinject:"required"`
}

func (u *AppspaceLoginRoutes) routeGroup(r chi.Router) {
	r.With(mustBeAuthenticated).Get("/appspacelogin", u.getTokenForRedirect)
}

func (u *AppspaceLoginRoutes) getTokenForRedirect(w http.ResponseWriter, r *http.Request) {
	log := u.getLogger("getTokenForRedirect").Clone

	appspaceDomain, ok := readSingleQueryParam(r, "appspace")
	if !ok {
		http.Error(w, "Missing or malformed appspace domain query parameter", http.StatusBadRequest)
		return
	}
	err := validator.DomainName(appspaceDomain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	appspaceDomain = validator.NormalizeDomainName(appspaceDomain)

	authUserID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		log().Log("no auth user ID in context")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sessionID, ok := domain.CtxSessionID(r.Context())
	if !ok {
		log().Log("no session ID in context")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	appspace, err := u.AppspaceModel.GetFromDomain(appspaceDomain)
	if err != nil {
		// some day appspace model will return an error if appspace not found.
		// handle that appropriately.
		returnError(w, err)
		return
	}
	if appspace != nil && appspace.OwnerID == authUserID {
		// Found an appspace owned by the user requesting a token.
		// We're assuming the appspace owner is a user.
		// This is handled differently from "remote" appspaces because
		// there is no entry for this appspace in owner's "remote" appspaces.
		token, err := u.V0TokenManager.GetForOwner(appspace.AppspaceID, appspace.DropID)
		if err != nil {
			returnError(w, err)
			return
		}
		http.Redirect(w, r, u.makeRedirectLink(appspaceDomain, token), http.StatusTemporaryRedirect)
		return
	}

	ver, err := u.DS2DS.GetRemoteAPIVersion(appspaceDomain)
	if err != nil {
		// we'll get more detailed with errors later...
		// it could be we couldn't reach teh remote server, or got an http error or something
		// it could be there is no common API version we can use
		http.Error(w, "error determining remote API version: "+err.Error(), http.StatusInternalServerError)
		return
	}

	switch ver {
	case 0:
		loginToken, err := u.V0RequestToken.RequestToken(r.Context(), authUserID, appspaceDomain, sessionID)
		if err != nil {
			// we need to get subtle with errors, probably have a whole set of sentinel errors?
			w.WriteHeader(http.StatusServiceUnavailable) //for now
			w.Write([]byte(err.Error()))
			return
		}
		http.Redirect(w, r, u.makeRedirectLink(appspaceDomain, loginToken), http.StatusTemporaryRedirect)

	default:
		// log this. This should not happen.
		http.Error(w, "remote API version missing", http.StatusInternalServerError)
	}
}

func (u *AppspaceLoginRoutes) makeRedirectLink(appspaceDomain, token string) string {
	query := make(url.Values)
	query.Add("dropserver-login-token", token)

	return fmt.Sprintf("%s://%s%s?%s", u.Config.ExternalAccess.Scheme, appspaceDomain, u.Config.Exec.PortString, query.Encode())
}

func (e *AppspaceLoginRoutes) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceLoginRoutes")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
