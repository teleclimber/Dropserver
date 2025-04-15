package userroutes

import (
	"net/http"
	"net/url"

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
	} `checkinject:"required"`
	SettingsModel interface {
		Get() (domain.Settings, error)
		SetRegistrationOpen(bool) error
		GetTSNet() (domain.TSNetCommon, error)
		SetTSNet(domain.TSNetCommon) error
		SetTSNetConnect(bool) error
		DeleteTSNet() error
	} `checkinject:"required"`
	UserInvitationModel interface {
		GetAll() ([]domain.UserInvitation, error)
		Get(email string) (domain.UserInvitation, error)
		Create(email string) error
		Delete(email string) error
	} `checkinject:"required"`
	UserTSNet interface {
		Create(domain.TSNetCreateConfig) error
		Connect(bool) error
		Disconnect()
		Delete() error
		GetStatus() domain.TSNetStatus
		GetPeerUsers() []domain.TSNetPeerUser
	} `checkinject:"required"`
}

func (a *AdminRoutes) subRouter() http.Handler {
	r := chi.NewRouter()

	// TODO admin-only middleware

	r.Get("/user/", a.getUsers)
	r.Get("/settings", a.getSettings)
	r.Post("/settings/registration", a.postRegistration)
	r.Post("/settings/tsnet/connect", a.postTSNetConnect)
	r.Post("/settings/tsnet", a.postTSNet)
	r.Delete("/settings/tsnet", a.deleteTSNet)
	r.Get("/invitation/", a.getInvitations)
	r.Post("/invitation", a.postInvitation)
	r.Delete("/invitation/{email}", a.deleteInvitation)
	r.Get("/tsnet", a.getTSNetStatus)
	r.Get("/tsnet/peerusers", a.getTSNetPeerUsers)

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
	RegistrationOpen bool               `json:"registration_open"`
	TSNet            domain.TSNetCommon `json:"tsnet"`
}

func (a *AdminRoutes) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := a.SettingsModel.Get()
	if err != nil {
		returnError(w, err)
		return
	}

	tsnetConfig, err := a.SettingsModel.GetTSNet()
	if err != nil {
		returnError(w, err)
		return
	}

	respData := SettingsResp{
		RegistrationOpen: settings.RegistrationOpen,
		TSNet:            tsnetConfig}

	writeJSON(w, respData)
}

type RegistrationPost struct {
	Open bool `json:"open"`
}

func (a *AdminRoutes) postRegistration(w http.ResponseWriter, r *http.Request) {
	reqData := &RegistrationPost{}
	err := readJSON(r, reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.SettingsModel.SetRegistrationOpen(reqData.Open)
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *AdminRoutes) getTSNetStatus(w http.ResponseWriter, r *http.Request) {
	status := a.UserTSNet.GetStatus()
	writeJSON(w, status)
}

func (a *AdminRoutes) getTSNetPeerUsers(w http.ResponseWriter, r *http.Request) {
	peerUsers := a.UserTSNet.GetPeerUsers()
	writeJSON(w, peerUsers)
}

func (a *AdminRoutes) postTSNet(w http.ResponseWriter, r *http.Request) {
	reqData := domain.TSNetCreateConfig{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO gotta validate the fields that aren't bool.

	err = a.SettingsModel.SetTSNet(domain.TSNetCommon{
		ControlURL: reqData.ControlURL,
		Hostname:   reqData.Hostname,
		Connect:    true})
	if err != nil {
		returnError(w, err)
		return
	}

	err = a.UserTSNet.Create(reqData)
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *AdminRoutes) deleteTSNet(w http.ResponseWriter, r *http.Request) {
	err := a.SettingsModel.DeleteTSNet()
	if err != nil {
		returnError(w, err)
		return
	}

	err = a.UserTSNet.Delete()
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type PostConnect struct {
	Connect bool `js:"connect"`
}

func (a *AdminRoutes) postTSNetConnect(w http.ResponseWriter, r *http.Request) {
	reqData := PostConnect{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.SettingsModel.SetTSNetConnect(reqData.Connect)
	if err != nil {
		returnError(w, err)
		return
	}

	if reqData.Connect {
		err = a.UserTSNet.Connect(true)
		if err != nil {
			returnError(w, err)
			return
		}
	} else {
		a.UserTSNet.Disconnect()
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
		w.WriteHeader(http.StatusOK) // Status OK means request was technically correct but action did not take place
		w.Write([]byte("Email not valid"))
		return
	}

	err = a.UserInvitationModel.Create(reqData.Email)
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *AdminRoutes) deleteInvitation(w http.ResponseWriter, r *http.Request) {
	email, err := url.QueryUnescape(chi.URLParam(r, "email"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = a.UserInvitationModel.Delete(email)
	if err != nil {
		if err == domain.ErrNoRowsAffected {
			w.WriteHeader(http.StatusNotFound)
		} else {
			returnError(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}
