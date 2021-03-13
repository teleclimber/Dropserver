package userroutes

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/validator"
)

type PostAppspaceUser struct {
	AuthType    string   `json:"auth_type"`
	AuthID      string   `json:"auth_id"`
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
}

type PatchAppspaceUserMeta struct {
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
}

// AppspaceUserRoutes handles routes for getting and mainpulating
// appspace users
type AppspaceUserRoutes struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetForAppspace(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
		Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
		UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, permissions []string) error
		Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error
	}
}

func (a *AppspaceUserRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	//routeData.URLTail = tail

	if head == "" {
		switch req.Method {
		case http.MethodGet:
			//get all appspace users
			a.getAllUsers(res, req, routeData, appspace)
		case http.MethodPost:
			// add a user
			a.newUser(res, req, routeData, appspace)
		default:
			returnError(res, errBadRequest)
		}

	} else {

		switch req.Method {
		case http.MethodGet:
			a.getUser(res, req, routeData, appspace)
		case http.MethodPatch:
			// change something about existing user
			a.updateUserMeta(res, req, routeData, appspace)
		case http.MethodDelete:
			a.deleteUser(res, req, routeData, appspace)
		default:
			returnError(res, errBadRequest)
		}
	}

}

func (a *AppspaceUserRoutes) getAllUsers(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	users, err := a.AppspaceUserModel.GetForAppspace(appspace.AppspaceID)
	if err != nil {
		returnError(res, err)
		return
	}

	// still unsure whether we just use the struct from domain or a special struct for the JSON.

	writeJSON(res, users)
}

func (a *AppspaceUserRoutes) newUser(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	reqData := PostAppspaceUser{}
	err := readJSON(req, &reqData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	authID, err := validateAuthStrings(reqData.AuthType, reqData.AuthID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	displayName := ""
	if reqData.DisplayName != "" {
		displayName = validator.NormalizeDisplayName(reqData.DisplayName)
		if err = validator.DisplayName(displayName); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// not yet sure how to deal with permissions....
	if len(reqData.Permissions) != 0 {
		// if( len)

		// hasMeta = true
	}

	proxyID, err := a.AppspaceUserModel.Create(appspace.AppspaceID, reqData.AuthType, authID)
	if err != nil {
		returnError(res, err)
		return
	}

	if displayName != "" {
		err = a.AppspaceUserModel.UpdateMeta(appspace.AppspaceID, proxyID, displayName, []string{})
		if err != nil {
			// This is where it would be nice to roll back....
			// We can ignore the error and return the result of "get"
			// And user will notice that something didn't quite work.
			// error is captured in logger in model
		}
	}

	appspaceUser, err := a.AppspaceUserModel.Get(appspace.AppspaceID, proxyID)
	if err != nil {
		returnError(res, err)
		return
	}

	writeJSON(res, appspaceUser)
}

func (a *AppspaceUserRoutes) getUser(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	proxyID, err := a.getProxyID(routeData.URLTail)
	if err != nil {
		returnError(res, err)
		return
	}

	user, err := a.AppspaceUserModel.Get(appspace.AppspaceID, proxyID)
	if err != nil {
		if err == sql.ErrNoRows {
			returnError(res, errNotFound)
			return
		}
	}

	writeJSON(res, user)
}

func (a *AppspaceUserRoutes) updateUserMeta(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	proxyID, err := a.getProxyID(routeData.URLTail)
	if err != nil {
		returnError(res, err)
		return
	}

	reqData := PatchAppspaceUserMeta{}
	err = readJSON(req, &reqData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	displayName := validator.NormalizeDisplayName(reqData.DisplayName)
	if err = validator.DisplayName(displayName); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// not yet sure how to deal with permissions....
	if len(reqData.Permissions) != 0 {
		// if( len)

		// hasMeta = true
	}

	err = a.AppspaceUserModel.UpdateMeta(appspace.AppspaceID, proxyID, displayName, []string{})
	if err != nil {
		returnError(res, err)
		return
	}

	appspaceUser, err := a.AppspaceUserModel.Get(appspace.AppspaceID, proxyID)
	if err != nil {
		returnError(res, err)
		return
	}

	writeJSON(res, appspaceUser)
}

func (a *AppspaceUserRoutes) deleteUser(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	proxyID, err := a.getProxyID(routeData.URLTail)
	if err != nil {
		returnError(res, err)
		return
	}

	err = a.AppspaceUserModel.Delete(appspace.AppspaceID, proxyID)
	if err != nil {
		returnError(res, err)
	}
}

func validateAuthStrings(authType, authID string) (string, error) {
	err := validator.AppspaceUserAuthType(authType)
	if err != nil {
		return "", err
	}

	if authType == "email" {
		authID = validator.NormalizeEmail(authID)
		err = validator.Email(authID)
		if err != nil {
			return "", err
		}
	} else if authType == "dropid" {
		authID = validator.NormalizeDropIDFull(authID)
		err = validator.DropIDFull(authID)
		if err != nil {
			return "", err
		}
	} else {
		return "", errors.New("unimplemented auth type " + authType)
	}
	return authID, nil
}

func (a *AppspaceUserRoutes) getProxyID(urlTail string) (domain.ProxyID, error) {
	proxyStr, _ := shiftpath.ShiftPath(urlTail)

	err := validator.UserProxyID(proxyStr)
	if err != nil {
		return domain.ProxyID(""), err
	}
	return domain.ProxyID(proxyStr), nil
}
