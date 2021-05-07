package userroutes

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/validator"
)

type RemoteAppspaceMeta struct {
	UserID      domain.UserID `json:"user_id"`
	DomainName  string        `json:"domain_name"`
	OwnerDropID string        `json:"owner_dropid"`
	UserDropID  string        `json:"dropid"`
	NoSSL       bool          `json:"no_ssl"`
	PortString  string        `json:"port_string"`
	Created     time.Time     `json:"created_dt"`
}

type RemoteAppspaceRoutes struct {
	Config              domain.RuntimeConfig
	RemoteAppspaceModel interface {
		Get(userID domain.UserID, domainName string) (domain.RemoteAppspace, error)
		GetForUser(userID domain.UserID) ([]domain.RemoteAppspace, error)
		Create(userID domain.UserID, domainName string, ownerDropID string, dropID string) error
		Delete(userID domain.UserID, domainName string) error
	}
	AppspaceModel interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, error)
	}
	DropIDModel interface {
		Get(handle string, dom string) (domain.DropID, error)
	}
}

func (a *RemoteAppspaceRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	domainStr, _ := shiftpath.ShiftPath(routeData.URLTail)

	if domainStr == "" {
		switch req.Method {
		case http.MethodGet:
			a.getForUser(res, req, routeData)
		case http.MethodPost:
			a.postNew(res, req, routeData)
		default:
			http.Error(res, "bad method for /remoteappspace", http.StatusBadRequest)
		}
	} else {
		switch req.Method {
		case http.MethodGet:
			a.getRemoteAppspace(res, req, routeData)
		case http.MethodDelete:
			a.delete(res, req, routeData)
		default:
			http.Error(res, "bad method for /remoteappspace/...", http.StatusBadRequest)
		}
	}
}

func (a *RemoteAppspaceRoutes) getForUser(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	remotes, err := a.RemoteAppspaceModel.GetForUser(routeData.Authentication.UserID)
	if err != nil {
		returnError(res, err)
		return
	}

	respData := make([]RemoteAppspaceMeta, len(remotes))
	for i, r := range remotes {
		respData[i] = a.makeRemoteAppspaceMeta(r)
	}

	writeJSON(res, respData)
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

func (a *RemoteAppspaceRoutes) postNew(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// We should have a struct just for post
	// and it should have a "checkConn" field
	ret := RemoteAppspacePostRet{}
	ret.InputsValid = true

	var remote RemoteAppspacePost
	readJSON(req, &remote)

	// validate and normlize domain name
	err := validator.DomainName(remote.DomainName)
	if err != nil {
		ret.DomainMessage = "address is invalid"
		ret.InputsValid = false
	}
	domainName := validator.NormalizeDomainName(remote.DomainName)

	// Check that the domain is not in use as a remote appspace for this user
	if ret.InputsValid {
		_, err = a.RemoteAppspaceModel.Get(routeData.Authentication.UserID, domainName)
		if err == nil {
			ret.DomainMessage = "remote appspace address already exists for user"
			ret.InputsValid = false
		}
		if err != sql.ErrNoRows {
			returnError(res, err)
			return
		}
	}

	if ret.InputsValid {
		// Check that the domain is not in use as a local appspace
		appspaces, err := a.AppspaceModel.GetForOwner(routeData.Authentication.UserID)
		if err != nil {
			returnError(res, err)
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
		returnError(res, err) // this is selected from a dropdown in frontend so should never be invalid.
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
		returnError(res, err)
		return
	}

	if ret.InputsValid {
		if remote.CheckOnly {
			// here we have to launch the comms to remote server.
			// This is almost certainly the REmoteAppspaceController

		} else {
			err = a.RemoteAppspaceModel.Create(routeData.Authentication.UserID, domainName, "", userDropID)
			if err != nil {
				returnError(res, err)
				return
			}
		}
	}

	writeJSON(res, ret)
}

func (a *RemoteAppspaceRoutes) getRemoteAppspace(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	domainRawStr, _ := shiftpath.ShiftPath(routeData.URLTail)
	domainStr, err := getCleanDomain(domainRawStr)
	if err != nil {
		returnError(res, err)
		return
	}

	remote, err := a.RemoteAppspaceModel.Get(routeData.Authentication.UserID, domainStr)
	if err != nil {
		returnError(res, err)
		return
	}

	writeJSON(res, a.makeRemoteAppspaceMeta(remote))
}

func (a *RemoteAppspaceRoutes) delete(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	domainRawStr, _ := shiftpath.ShiftPath(routeData.URLTail)
	domainStr, err := getCleanDomain(domainRawStr)
	if err != nil {
		returnError(res, err)
		return
	}

	err = a.RemoteAppspaceModel.Delete(routeData.Authentication.UserID, domainStr)
	if err != nil {
		returnError(res, err)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func getCleanDomain(dom string) (string, error) {
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
		NoSSL:       a.Config.Server.NoSsl,
		PortString:  a.Config.PortString,
		Created:     appspace.Created}
}
