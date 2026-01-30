package userroutes

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// AppspaceLoginRoutes handle
type AppspaceLoginRoutes struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	AppspaceModel interface {
		GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, error)
		GetFromDomain(dom string) (*domain.Appspace, error)
	} `checkinject:"required"`
	ManageAppspaceUsers interface {
		GetProxyIDForUserID(domain.AppspaceID, domain.UserID) (domain.ProxyID, error)
	} `checkinject:"required"`
	V0TokenManager interface {
		GetForProxyID(appspaceID domain.AppspaceID, proxyID domain.ProxyID) string
	} `checkinject:"required"`
}

func (u *AppspaceLoginRoutes) routeGroup(r chi.Router) {
	r.Use(mustBeAuthenticated)
	r.Get("/appspacelogin", u.getTokenForRedirect)
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

	proxyID, err := u.ManageAppspaceUsers.GetProxyIDForUserID(appspaceID, authUserID)
	if err == domain.ErrNoRowsInResultSet {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// TODO check we properly handle a user with a conflict!

	token := u.V0TokenManager.GetForProxyID(appspace.AppspaceID, proxyID)

	http.Redirect(w, r, u.makeRedirectLink(appspace.DomainName, token), http.StatusTemporaryRedirect)
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
