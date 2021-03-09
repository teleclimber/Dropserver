package userroutes

import (
	"database/sql"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// AdminRoutes handles routes for applications uploading, creating, deleting.
type AdminRoutes struct {
	UserModel interface {
		GetAll() ([]domain.User, error)
		IsAdmin(userID domain.UserID) bool
		GetAllAdmins() ([]domain.UserID, error)
	}
	SettingsModel interface {
		Get() (domain.Settings, error)
		Set(domain.Settings) error
	}
	UserInvitationModel interface {
		GetAll() ([]domain.UserInvitation, error)
		Get(email string) (domain.UserInvitation, error)
		Create(email string) error
		Delete(email string) error
	}
	Validator domain.Validator
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
		invite, hasInvite, err := a.getInvitationFromPath(routeData)
		if err != nil {
			if err == sql.ErrNoRows {
				returnError(res, errNotFound)
				return
			}
			returnError(res, err)
			return
		}

		if hasInvite {
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
	settings, err := a.SettingsModel.Get()
	if err != nil {
		returnError(res, err)
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

	err = a.SettingsModel.Set(*reqData)
	if err != nil {
		returnError(res, err)
		return
	}

	res.WriteHeader(http.StatusOK)
}

// invitations
func (a *AdminRoutes) getInvitationFromPath(routeData *domain.AppspaceRouteData) (domain.UserInvitation, bool, error) {
	email, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	var invite domain.UserInvitation

	if email == "" {
		return invite, false, nil
	}

	invite, err := a.UserInvitationModel.Get(email)
	if err != nil {
		return invite, true, err
	}

	return invite, true, nil
}
func (a *AdminRoutes) getInvitations(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	invites, err := a.UserInvitationModel.GetAll()
	if err != nil {
		returnError(res, err)
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

	err = a.UserInvitationModel.Create(reqData.Email)
	if err != nil {
		returnError(res, err)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (a *AdminRoutes) deleteInvitation(res http.ResponseWriter, invite domain.UserInvitation) {
	err := a.UserInvitationModel.Delete(invite.Email)
	if err != nil {
		returnError(res, err)
		return
	}

	res.WriteHeader(http.StatusOK)
}
