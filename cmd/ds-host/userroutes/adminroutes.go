package userroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// AdminRoutes handles routes for applications uploading, creating, deleting.
type AdminRoutes struct {
	UserModel     domain.UserModel
	SettingsModel domain.SettingsModel
	Logger        domain.LogCLientI
}

// ServeHTTP handles http traffic to the admin routes
func (a *AdminRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Cookie == nil || !a.UserModel.IsAdmin(routeData.Cookie.UserID) {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	switch head {
	case "user":
		switch req.Method {
		case http.MethodGet:
			a.getUsers(res, req, routeData)
		default:
			http.Error(res, "method not implemented for user", http.StatusBadRequest)
		}
	case "settings":
		switch req.Method {
		case http.MethodGet:
			a.getSettings(res, req, routeData)
		case http.MethodPatch:
			a.patchSettings(res, req, routeData)
		default:
			http.Error(res, "method not implemented for user", http.StatusBadRequest)
		}
	default:
		res.WriteHeader(http.StatusNotFound)
	}

}

// getUsers returns the list of users on the system
func (a *AdminRoutes) getUsers(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	users, dsErr := a.UserModel.GetAll()
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	admins, dsErr := a.UserModel.GetAllAdmins()
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	usersResp := []userResp{}
	for _, u := range users {
		ur := userResp{
			UserID:  int(u.UserID),
			Email:   u.Email,
			IsAdmin: false}
		for _, uid := range admins {
			if uid == u.UserID {
				ur.IsAdmin = true
				break
			}
		}

		usersResp = append(usersResp, ur)
	}

	writeJSON(res, adminGetUsersResp{Users: usersResp})
}

func (a *AdminRoutes) getSettings(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	settings, dsErr := a.SettingsModel.Get()
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	respData := getSettingsResp{
		Settings: *settings}

	writeJSON(res, respData)
}

func (a *AdminRoutes) patchSettings(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	reqData := &postSettingsReq{}
	dsErr := readJSON(req, reqData)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	// gotta validate the fields that aren't bool.

	dsErr = a.SettingsModel.Set(&reqData.Settings)
	if dsErr != nil {
		dsErr.HTTPError(res)
	}

	res.WriteHeader(http.StatusOK)
}

