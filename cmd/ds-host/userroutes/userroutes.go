package userroutes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	dshostfrontend "github.com/teleclimber/DropServer/frontend-ds-host"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/twine"
	"github.com/teleclimber/DropServer/internal/validator"
)

var errNotFound = errors.New("not found")
var errBadRequest = errors.New("bad request") // feels like this should be wrapped or something, so we can also have access to the original error?
var errForbidden = errors.New("forbidden")

func returnError(res http.ResponseWriter, err error) {
	switch err {
	case errNotFound:
		http.Error(res, "not found", http.StatusNotFound)
	case errBadRequest:
		http.Error(res, "bad request", http.StatusBadRequest)
	case errForbidden:
		http.Error(res, "forbidden", http.StatusForbidden)
	default:
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

// UserRoutes handles routes for appspaces.
type UserRoutes struct {
	AuthRoutes           domain.RouteHandler
	AppspaceLoginRoutes  domain.RouteHandler
	ApplicationRoutes    domain.RouteHandler
	AppspaceRoutes       domain.RouteHandler
	RemoteAppspaceRoutes domain.RouteHandler
	ContactRoutes        domain.RouteHandler
	DomainRoutes         domain.RouteHandler
	DropIDRoutes         domain.RouteHandler
	MigrationJobRoutes   domain.RouteHandler
	AdminRoutes          domain.RouteHandler
	AppspaceStatusTwine  domain.TwineService
	MigrationJobTwine    domain.TwineService
	UserModel            interface {
		GetFromID(userID domain.UserID) (domain.User, error)
		UpdatePassword(userID domain.UserID, password string) error
		GetFromEmailPassword(email, password string) (domain.User, error)
		IsAdmin(userID domain.UserID) bool
	}
}

// ServeHTTP handles http traffic to the user routes
func (u *UserRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	// Consider that apart from login routes, everything else requires authentication
	// Would like to make that abundantly clear in code structure.
	// There should be a single point where we check auth, and if no good, bail.

	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	if head == "signup" || head == "login" || head == "logout" { // also resetpw
		u.AuthRoutes.ServeHTTP(res, req, routeData)
	} else {
		if routeData.Authentication != nil && routeData.Authentication.UserAccount {
			ctx := req.Context()
			ctx = ctxWithAuthUserID(ctx, routeData.Authentication.UserID)
			u.serveLoggedInRoutes(res, req.WithContext(ctx), routeData)
		} else {
			u.serveRedirectToLogin(res, req, routeData)
		}
	}
}

func (u *UserRoutes) serveLoggedInRoutes(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	if routeData.Authentication.UserAccount == false {
		// log it too
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we dun fogged up
		return
	}

	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	switch head {
	case "":
		htmlBytes, err := dshostfrontend.FS.ReadFile("dist/index.html")
		if err != nil {
			returnError(res, err)
			return
		}
		res.Header().Set("Content-Type", "text/html; charset=utf-8")
		res.Write(htmlBytes)
	case "appspacelogin":
		routeData.URLTail = tail
		u.AppspaceLoginRoutes.ServeHTTP(res, req, routeData)
	case "api":
		head, tail = shiftpath.ShiftPath(tail)
		routeData.URLTail = tail
		switch head {
		case "admin":
			u.AdminRoutes.ServeHTTP(res, req, routeData)
		case "user":
			switch req.Method {
			case http.MethodGet:
				u.getUserData(res, req, routeData)
			case http.MethodPatch:
				u.setUserData(res, req, routeData)
			default:
				res.WriteHeader(http.StatusNotFound)
			}
		case "domainname":
			u.DomainRoutes.ServeHTTP(res, req, routeData)
		case "dropid":
			u.DropIDRoutes.ServeHTTP(res, req, routeData)
		case "application":
			u.ApplicationRoutes.ServeHTTP(res, req, routeData)
		case "appspace":
			u.AppspaceRoutes.ServeHTTP(res, req, routeData)
		case "remoteappspace":
			u.RemoteAppspaceRoutes.ServeHTTP(res, req, routeData)
		case "contact":
			u.ContactRoutes.ServeHTTP(res, req, routeData)
		case "migration-job":
			u.MigrationJobRoutes.ServeHTTP(res, req, routeData)
		default:
			http.Error(res, head+" not implemented", http.StatusNotImplemented)
		}
	case "twine":
		u.startTwineService(res, req, routeData)
	default:
		res.WriteHeader(http.StatusNotFound)
	}
}

func (u *UserRoutes) serveRedirectToLogin(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	if head == "" || head == "admin" {
		http.Redirect(res, req, "/login", http.StatusTemporaryRedirect)
	} else {
		res.WriteHeader(http.StatusUnauthorized)
	}
}

// UserData is single user
type UserData struct {
	Email   string `json:"email"`
	UserID  int    `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
}

// getUserData returns a json with {email: ""...""} I think, so far.
func (u *UserRoutes) getUserData(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// check if there is anything in routeData tail?
	user, err := u.UserModel.GetFromID(routeData.Authentication.UserID)
	if err != nil {
		returnError(res, err)
		return
	}

	isAdmin := u.UserModel.IsAdmin(user.UserID)

	userData := UserData{
		UserID:  int(user.UserID),
		Email:   user.Email,
		IsAdmin: isAdmin}

	userJSON, err := json.Marshal(userData)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(userJSON)
}

func (u *UserRoutes) setUserData(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail
	if head == "password" {
		u.changeUserPassword(res, req, routeData)
	} else {
		res.WriteHeader(http.StatusNotImplemented)
	}
}

func (u *UserRoutes) changeUserPassword(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var data PatchPasswordReq
	err = json.Unmarshal(body, &data)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	dsErr := validator.Password(data.Old)
	if dsErr != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	dsErr = validator.Password(data.New)
	if dsErr != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := u.UserModel.GetFromID(routeData.Authentication.UserID)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = u.UserModel.GetFromEmailPassword(user.Email, data.Old)
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = u.UserModel.UpdatePassword(user.UserID, data.New)
	if err != nil {
		returnError(res, err)
		return
	}

	res.WriteHeader(http.StatusOK)
}

const appspaceStatusService = 11
const migrationJobService = 12

// startTwineService connects a new twine instance to the twine services
func (u *UserRoutes) startTwineService(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	t, err := twine.NewWebsocketServer(res, req)
	if err != nil {
		u.getLogger("startTwineService, twine.NewWebsocketServer(res, req) ").Error(err)
		http.Error(res, "Failed to start Twine server", http.StatusInternalServerError)
		return
	}

	_, ok := <-t.ReadyChan
	if !ok {
		u.getLogger("startTwineService").Error(errors.New("Twine ReadyChan closed"))
		http.Error(res, "Failed to start Twine server", http.StatusInternalServerError)
		return
	}

	go u.AppspaceStatusTwine.Start(routeData.Authentication.UserID, t)
	go u.MigrationJobTwine.Start(routeData.Authentication.UserID, t)

	go func() {
		for m := range t.MessageChan {
			switch m.ServiceID() {
			case appspaceStatusService:
				go u.AppspaceStatusTwine.HandleMessage(m)
			case migrationJobService:
				go u.MigrationJobTwine.HandleMessage(m)
			default:
				u.getLogger("Twine incoming message").Error(fmt.Errorf("Service not found: %v", m.ServiceID()))
				m.SendError("Service not found")
			}
		}
	}()

}

func (u *UserRoutes) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("UserRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
