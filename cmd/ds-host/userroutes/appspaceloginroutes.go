package userroutes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

// AppspaceLoginRoutes handle
type AppspaceLoginRoutes struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	AppspaceModel interface {
		GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, error)
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
	r.Use(mustBeAuthenticated)
	r.Get("/appspacelogin", u.getTokenForRedirect)
	r.Get("/remoteappspacelogin", u.getTokenForRemoteRedirect)
}

func (u *AppspaceLoginRoutes) getTokenForRedirect(w http.ResponseWriter, r *http.Request) {
	log := u.getLogger("getTokenForRedirect").Clone

	appspaceIdStr, ok := readSingleQueryParam(r, "appspace_id")
	if !ok {
		http.Error(w, "Missing or malformed appspace ID query parameter", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(appspaceIdStr)
	if err != nil {
		http.Error(w, "Malformed appspace ID query parameter", http.StatusBadRequest)
		return
	}
	appspaceID := domain.AppspaceID(id)

	authUserID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		log().Log("no auth user ID in context")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	appspace, err := u.AppspaceModel.GetFromID(appspaceID)
	if err == domain.ErrNoRowsInResultSet {
		http.Error(w, "Appspace not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Here we will change things a bit:
	// - get auth user's auth ids
	// - from appspace's meta db find matching appspace users
	// - [also from appspace_instance_users get an appspace user]
	// - If exactly one appspace user procedd w/ token and redirect.
	if appspace.OwnerID == authUserID {
		// Found an appspace owned by the user requesting a token.
		// We're assuming the appspace owner is a user.
		// This is handled differently from "remote" appspaces because
		// there is no entry for this appspace in owner's "remote" appspaces.
		token, err := u.V0TokenManager.GetForOwner(appspace.AppspaceID, appspace.DropID)
		if err != nil {
			returnError(w, err)
			return
		}
		http.Redirect(w, r, u.makeRedirectLink(appspace.DomainName, token), http.StatusTemporaryRedirect)
		return
	}

	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func (u *AppspaceLoginRoutes) getTokenForRemoteRedirect(w http.ResponseWriter, r *http.Request) {
	log := u.getLogger("getTokenForRemoteRedirect").Clone

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
