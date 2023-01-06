package userroutes

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type RemoteAppspaceMeta struct {
	UserID      domain.UserID `json:"user_id"`
	DomainName  string        `json:"domain_name"`
	OwnerDropID string        `json:"owner_dropid"`
	UserDropID  string        `json:"dropid"`
	NoTLS       bool          `json:"no_tls"`
	PortString  string        `json:"port_string"`
	Created     time.Time     `json:"created_dt"`
}

type RemoteAppspaceRoutes struct {
	Config              domain.RuntimeConfig `checkinject:"required"`
	RemoteAppspaceModel interface {
		Get(userID domain.UserID, domainName string) (domain.RemoteAppspace, error)
		GetForUser(userID domain.UserID) ([]domain.RemoteAppspace, error)
		Create(userID domain.UserID, domainName string, ownerDropID string, dropID string) error
		Delete(userID domain.UserID, domainName string) error
	} `checkinject:"required"`
	AppspaceModel interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, error)
	} `checkinject:"required"`
	DropIDModel interface {
		Get(handle string, dom string) (domain.DropID, error)
	} `checkinject:"required"`
}

func (a *RemoteAppspaceRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", a.getForUser)
	r.Post("/", a.postNew)

	r.Route("/{remotedomain}", func(r chi.Router) {
		r.Get("/", a.get)
		r.Delete("/", a.delete)
	})

	return r
}

func (a *RemoteAppspaceRoutes) getForUser(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	remotes, err := a.RemoteAppspaceModel.GetForUser(userID)
	if err != nil {
		returnError(w, err)
		return
	}

	respData := make([]RemoteAppspaceMeta, len(remotes))
	for i, r := range remotes {
		respData[i] = a.makeRemoteAppspaceMeta(r)
	}

	writeJSON(w, respData)
}

type RemoteAppspacePost struct {
	CheckOnly  bool   `json:"check_only"`
	DomainName string `json:"domain_name"`
	UserDropID string `json:"user_dropid"`
	// maybe also desired handle and avatar
}
type RemoteAppspacePostRet struct {
	InputsValid   bool   `json:"inputs_valid"`
	DomainMessage string `json:"domain_message"`
	RemoteMessage string `json:"remote_message"`
	// RemoteMeta? Like remote drop id and any other stuff
}

func (a *RemoteAppspaceRoutes) postNew(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	// We should have a struct just for post
	// and it should have a "checkConn" field
	ret := RemoteAppspacePostRet{}
	ret.InputsValid = true

	var remote RemoteAppspacePost
	readJSON(r, &remote)

	// validate and normlize domain name
	err := validator.DomainName(remote.DomainName)
	if err != nil {
		ret.DomainMessage = "address is invalid"
		ret.InputsValid = false
	}
	domainName := validator.NormalizeDomainName(remote.DomainName)

	// Check that the domain is not in use as a remote appspace for this user
	if ret.InputsValid {
		_, err = a.RemoteAppspaceModel.Get(userID, domainName)
		if err == nil {
			ret.DomainMessage = "remote appspace address already exists for user"
			ret.InputsValid = false
		}
		if err != sql.ErrNoRows {
			returnError(w, err)
			return
		}
	}

	if ret.InputsValid {
		// Check that the domain is not in use as a local appspace
		appspaces, err := a.AppspaceModel.GetForOwner(userID)
		if err != nil {
			returnError(w, err)
			return
		}
		for _, appspace := range appspaces {
			if validator.NormalizeDomainName(appspace.DomainName) == domainName {
				ret.DomainMessage = "domain already existst as a local appspace"
				ret.InputsValid = false
			}
		}
	}

	// validate normalize and check exists dropid
	err = validator.DropIDFull(remote.UserDropID)
	if err != nil {
		returnError(w, err) // this is selected from a dropdown in frontend so should never be invalid.
		return
	}
	userDropID := validator.NormalizeDropIDFull(remote.UserDropID)

	// check user drop id exists
	handle, dom := validator.SplitDropID(userDropID)
	_, err = a.DropIDModel.Get(handle, dom)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.New("user DropID does not exist")
		}
		returnError(w, err)
		return
	}

	if ret.InputsValid {
		if remote.CheckOnly {
			// here we have to launch the comms to remote server.
			// This is almost certainly the REmoteAppspaceController

		} else {
			err = a.RemoteAppspaceModel.Create(userID, domainName, "", userDropID)
			if err != nil {
				returnError(w, err)
				return
			}
		}
	}

	writeJSON(w, ret)
}

func (a *RemoteAppspaceRoutes) get(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	domainStr, err := getDomain(r)
	if err != nil {
		returnError(w, err)
		return
	}

	remote, err := a.RemoteAppspaceModel.Get(userID, domainStr)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, a.makeRemoteAppspaceMeta(remote))
}

func (a *RemoteAppspaceRoutes) delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	domainStr, err := getDomain(r)
	if err != nil {
		returnError(w, err)
		return
	}

	err = a.RemoteAppspaceModel.Delete(userID, domainStr)
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getDomain(r *http.Request) (string, error) {
	dom := chi.URLParam(r, "remotedomain")

	// chi is inconsistent with its pathescapes apparently:
	// https://github.com/go-chi/chi/issues/570
	domainStr, err := url.PathUnescape(dom)
	if err != nil {
		return "", err
	}

	err = validator.DomainName(domainStr)
	if err != nil {
		return "", err
	}
	return validator.NormalizeDomainName(domainStr), nil
}

func (a *RemoteAppspaceRoutes) makeRemoteAppspaceMeta(appspace domain.RemoteAppspace) RemoteAppspaceMeta {
	return RemoteAppspaceMeta{
		DomainName:  appspace.DomainName,
		OwnerDropID: appspace.OwnerDropID,
		UserDropID:  appspace.UserDropID,
		NoTLS:       a.Config.ExternalAccess.Scheme == "http", // used to create link in frontend, incorrect because this should prop of the remote
		PortString:  a.Config.Exec.PortString,                 // this is wrong for same reason as above
		Created:     appspace.Created}
}
