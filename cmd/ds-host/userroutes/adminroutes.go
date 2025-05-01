package userroutes

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

// AdminRoutes handles routes for applications uploading, creating, deleting.
type AdminRoutes struct {
	UserModel interface {
		GetAll() ([]domain.User, error)
		GetFromID(domain.UserID) (domain.User, error)
		IsAdmin(userID domain.UserID) bool
		GetAllAdmins() ([]domain.UserID, error)
		CreateWithTSNet(string, string) (domain.User, error)
		UpdateTSNet(domain.UserID, string, string) error
		DeleteTSNet(domain.UserID) error
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

	r.Use(a.mustBeAdmin)

	r.Get("/user/", a.getUsers)
	r.Post("/user/", a.postUser)
	r.Post("/user/{user_id}/tsnet", a.postUserTSNet)
	r.Delete("/user/{user_id}/tsnet", a.deleteUserTSNet)
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

func (a *AdminRoutes) mustBeAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := domain.CtxAuthUserID(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !a.UserModel.IsAdmin(userID) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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
		ur := UserData{u, false}
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

type postUserReq struct {
	TSNetID string `json:"tsnet_id"`
}

// postUser creates a new user with TSNet user id.
// In the future creating with email should be possible too.
// It returns the created user.
func (a *AdminRoutes) postUser(w http.ResponseWriter, r *http.Request) {
	reqData := postUserReq{}
	err := readJSON(r, &reqData)
	if err != nil {
		a.getLogger("postUser readJSON").Debug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	peerUser, err := a.getTSNetUser(reqData.TSNetID)
	if err != nil {
		a.getLogger("postUserTSNet").Debug(err.Error())
		writeBadRequest(w, "tsnet_id", "unable to find peer")
		return
	}
	user, err := a.UserModel.CreateWithTSNet(peerUser.FullID, peerUser.LoginName)
	if err == domain.ErrIdentifierExists {
		writeBadRequest(w, "tsnet_id", "peer already associated with another user")
		return
	} else if err != nil {
		writeServerError(w)
	}
	writeJSON(w, UserData{user, false})
}

type userTSNetReq struct {
	TSNetID string `json:"tsnet_id"`
}

type userTSNetResp struct {
	TSNetIdentifier string `json:"tsnet_identifier"`
	TSNetExtraName  string `json:"tsnet_extra_name"`
}

// postUserTSNet receives a tsnet identifier (just the bare tsnet id)
// and updates the user db with the full identifier and extra name
// based on the connected tsnet data.
// It sends back the user data pertaining to tsnet.
func (a *AdminRoutes) postUserTSNet(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		a.getLogger("postUserTSNet userIDFromRequest").Debug(err.Error())
		writeBadRequest(w, "user_id", err.Error())
		return
	}

	reqData := userTSNetReq{}
	err = readJSON(r, &reqData)
	if err != nil {
		a.getLogger("postUserTSNet readJSON").Debug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.TSNetIdentifier(reqData.TSNetID)
	if err != nil {
		a.getLogger("postUserTSNet validator.TSNetIdentifier").Debug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	peerUser, err := a.getTSNetUser(reqData.TSNetID)
	if err != nil {
		a.getLogger("postUserTSNet").Debug(err.Error())
		writeBadRequest(w, "tsnet_id", "unable to find peer")
		return
	}

	err = a.UserModel.UpdateTSNet(userID, peerUser.FullID, peerUser.LoginName)
	if err != nil {
		writeServerError(w)
		return
	}

	writeJSON(w, userTSNetResp{
		TSNetIdentifier: peerUser.FullID,
		TSNetExtraName:  peerUser.LoginName,
	})
}

func (a *AdminRoutes) getTSNetUser(tsnetID string) (domain.TSNetPeerUser, error) {
	if tsnetID == "" {
		return domain.TSNetPeerUser{}, errors.New("tsnet_id can not be empty")
	}
	peerUser := domain.TSNetPeerUser{}
	found := false
	for _, peerUser = range a.UserTSNet.GetPeerUsers() {
		if peerUser.ID == tsnetID {
			found = true
			break
		}
	}
	if !found {
		return domain.TSNetPeerUser{}, errors.New("tsnet id was not found among current peer users")
	}
	return peerUser, nil
}

func (a *AdminRoutes) deleteUserTSNet(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromRequest(r)
	if err != nil {
		writeBadRequest(w, "user_id", err.Error())
		return
	}

	err = a.UserModel.DeleteTSNet(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeOK(w)
}

func userIDFromRequest(r *http.Request) (domain.UserID, error) {
	userIDStr, err := url.QueryUnescape(chi.URLParam(r, "user_id"))
	if err != nil {
		return 0, fmt.Errorf("unable to read user_id from params: %w", err)
	}
	userIDInt, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, fmt.Errorf("unable to convert user_id to int: %w", err)
	}
	return domain.UserID(userIDInt), nil
}

// settingsResp represents admin settings
type settingsResp struct {
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

	respData := settingsResp{
		RegistrationOpen: settings.RegistrationOpen,
		TSNet:            tsnetConfig}

	writeJSON(w, respData)
}

type registrationPost struct {
	Open bool `json:"open"`
}

func (a *AdminRoutes) postRegistration(w http.ResponseWriter, r *http.Request) {
	reqData := &registrationPost{}
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

	err = validator.TSNetCreateConfig(reqData)
	if err != nil {
		a.getLogger("postTSNet validator").Debug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

type postConnect struct {
	Connect bool `js:"connect"`
}

func (a *AdminRoutes) postTSNetConnect(w http.ResponseWriter, r *http.Request) {
	reqData := postConnect{}
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
type postInvitation struct {
	Email string `json:"email"`
}

func (a *AdminRoutes) postInvitation(w http.ResponseWriter, r *http.Request) {
	reqData := &postInvitation{}
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

func (a *AdminRoutes) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("AdminRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
