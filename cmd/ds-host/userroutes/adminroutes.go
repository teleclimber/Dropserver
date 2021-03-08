package userroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// AdminRoutes handles routes for applications uploading, creating, deleting.
type AdminRoutes struct {
	UserModel interface {
		GetAll() ([]domain.User, error)
		IsAdmin(userID domain.UserID) bool
		GetAllAdmins() ([]domain.UserID, error)
	}
	SettingsModel       domain.SettingsModel
	UserInvitationModel domain.UserInvitationModel
	Validator           domain.Validator
}

// ServeHTTP handles http traffic to the admin routes
func (a *AdminRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Authentication == nil || !a.UserModel.IsAdmin(routeData.Authentication.UserID) {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail
	switch head {
	case "user":
		switch req.Method {
		case http.MethodGet:
			a.getUsers(res, req, routeData)
		default:
			http.Error(res, "method not implemented", http.StatusBadRequest)
		}
	case "settings":
		switch req.Method {
		case http.MethodGet:
			a.getSettings(res, req, routeData)
		case http.MethodPatch:
			a.patchSettings(res, req, routeData)
		default:
			http.Error(res, "method not implemented", http.StatusBadRequest)
		}
	case "invitation":
		// here gotta read email from path, as that is how delete works
		// ..and any other methods we attach to that invitation (like re-send email, ...)
		invite, dsErr := a.getInvitationFromPath(routeData)
		if dsErr != nil {
			if dsErr.Code() == dserror.NoRowsInResultSet {
				res.WriteHeader(http.StatusNotFound)
				return
			}
			dsErr.HTTPError(res)
			return
		}

		if invite != nil {
			switch req.Method {
			case http.MethodDelete:
				a.deleteInvitation(res, invite)
			default:
				http.Error(res, "method not implemented", http.StatusBadRequest)
			}
		} else {
			switch req.Method {
			case http.MethodGet:
				a.getInvitations(res, req, routeData)
			case http.MethodPost:
				a.postInvitation(res, req, routeData)
			default:
				http.Error(res, "method not implemented", http.StatusBadRequest)
			}
		}
	default:
		res.WriteHeader(http.StatusNotFound)
	}

}

// getUsers returns the list of users on the system
func (a *AdminRoutes) getUsers(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	users, err := a.UserModel.GetAll()
	if err != nil {
		returnError(res, err)
		return
	}

	admins, err := a.UserModel.GetAllAdmins()
	if err != nil {
		returnError(res, err)
		return
	}

	usersResp := []UserData{}
	for _, u := range users {
		ur := UserData{
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

	writeJSON(res, usersResp)
}

// SettingsResp represents admin settings
type SettingsResp struct {
	RegistrationOpen bool `json:"registration_open"`
}

func (a *AdminRoutes) getSettings(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	settings, dsErr := a.SettingsModel.Get()
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	respData := SettingsResp{
		RegistrationOpen: settings.RegistrationOpen,
	}

	writeJSON(res, respData)
}

// TODO this is really not the right way to go about patching settings.
// We should really have a route for each setting to post against.
// Work for another day
func (a *AdminRoutes) patchSettings(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	reqData := &domain.Settings{}
	err := readJSON(req, reqData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// gotta validate the fields that aren't bool.

	dsErr := a.SettingsModel.Set(reqData)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	res.WriteHeader(http.StatusOK)
}

// invitations
func (a *AdminRoutes) getInvitationFromPath(routeData *domain.AppspaceRouteData) (*domain.UserInvitation, domain.Error) {
	email, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if email == "" {
		return nil, nil
	}

	invite, dsErr := a.UserInvitationModel.Get(email)
	if dsErr != nil {
		return nil, dsErr
	}

	return invite, nil
}
func (a *AdminRoutes) getInvitations(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	invites, dsErr := a.UserInvitationModel.GetAll()
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	writeJSON(res, invites)
}

// PostInvitation is for incoming post requests to create invitation
type PostInvitation struct {
	Email string `json:"email"`
}

func (a *AdminRoutes) postInvitation(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	reqData := &PostInvitation{}
	err := readJSON(req, reqData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	dsErr := a.Validator.Email(reqData.Email)
	if dsErr != nil {
		http.Error(res, "email validation error: "+dsErr.PublicString(), http.StatusBadRequest)
		return
	}

	dsErr = a.UserInvitationModel.Create(reqData.Email)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (a *AdminRoutes) deleteInvitation(res http.ResponseWriter, invite *domain.UserInvitation) {
	dsErr := a.UserInvitationModel.Delete(invite.Email)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	res.WriteHeader(http.StatusOK)
}
