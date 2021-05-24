package userroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
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
}

func (a *AdminRoutes) subRouter() http.Handler {
	r := chi.NewRouter()

	// TODO admin-only middleware

	r.Get("/user/", a.getUsers)
	r.Get("/settings", a.getSettings)
	r.Patch("/settings", a.patchSettings)
	r.Get("/invitation/", a.getInvitations)
	r.Post("/invitation", a.postInvitation)
	r.Delete("/invitation/{email}", a.deleteInvitation)

	return r
}

// TODO: do this next
// func mustBeAdmin(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		_, ok := domain.CtxAuthUserID(r.Context())
// 		if !ok {
// 			// TODO: only do this when request is for an html page.
// 			if strings.HasPrefix(r.URL.Path, "/api") {
// 				w.WriteHeader(http.StatusUnauthorized)
// 			} else {
// 				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
// 			}
// 			return
// 		}

// 		next.ServeHTTP(w, r)
// 	})
// }

// getUsers returns the list of users on the system
func (a *AdminRoutes) getUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.UserModel.GetAll()
	if err != nil {
		returnError(w, err)
		return
	}

	admins, err := a.UserModel.GetAllAdmins()
	if err != nil {
		returnError(w, err)
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

	writeJSON(w, usersResp)
}

// SettingsResp represents admin settings
type SettingsResp struct {
	RegistrationOpen bool `json:"registration_open"`
}

func (a *AdminRoutes) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := a.SettingsModel.Get()
	if err != nil {
		returnError(w, err)
		return
	}

	respData := SettingsResp{
		RegistrationOpen: settings.RegistrationOpen,
	}

	writeJSON(w, respData)
}

// TODO this is really not the right way to go about patching settings.
// We should really have a route for each setting to post against.
// Work for another day
func (a *AdminRoutes) patchSettings(w http.ResponseWriter, r *http.Request) {
	reqData := &domain.Settings{}
	err := readJSON(r, reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// gotta validate the fields that aren't bool.

	err = a.SettingsModel.Set(*reqData)
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *AdminRoutes) getInvitations(w http.ResponseWriter, r *http.Request) {
	invites, err := a.UserInvitationModel.GetAll()
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, invites)
}

// PostInvitation is for incoming post requests to create invitation
type PostInvitation struct {
	Email string `json:"email"`
}

func (a *AdminRoutes) postInvitation(w http.ResponseWriter, r *http.Request) {
	reqData := &PostInvitation{}
	err := readJSON(r, reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.Email(reqData.Email)
	if err != nil {
		http.Error(w, "email validation error", http.StatusBadRequest)
		return
	}

	err = a.UserInvitationModel.Create(reqData.Email)
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *AdminRoutes) deleteInvitation(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")

	// TODO validate and normalize email

	err := a.UserInvitationModel.Delete(email)
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
