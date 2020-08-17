package userroutes

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// UserRoutes handles routes for appspaces.
type UserRoutes struct {
	AuthRoutes        domain.RouteHandler
	ApplicationRoutes domain.RouteHandler
	AppspaceRoutes    domain.RouteHandler
	AdminRoutes       domain.RouteHandler
	LiveDataRoutes    domain.RouteHandler
	UserModel         domain.UserModel
	Views             domain.Views
	Validator         domain.Validator
}

// ServeHTTP handles http traffic to the user routes
func (u *UserRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	// Consider that apart from login routes, everything else requires authentication
	// Would like to make that abundantly clear in code structure.
	// There should be a single point where we check auth, and if no good, bail.

	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	if head == "signup" || head == "appspacelogin" || head == "login" || head == "logout" { // also resetpw
		u.AuthRoutes.ServeHTTP(res, req, routeData)
	} else if head == "live" {
		routeData.URLTail = tail
		u.LiveDataRoutes.ServeHTTP(res, req, routeData)
	} else {
		if routeData.Authentication != nil && routeData.Authentication.UserAccount {
			u.serveLoggedInRoutes(res, req, routeData)
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
		u.Views.UserHome(res)
	case "admin":
		u.Views.Admin(res)
	case "api":
		// All the async routes essentially?
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
		case "application":
			u.ApplicationRoutes.ServeHTTP(res, req, routeData)
		case "appspace":
			u.AppspaceRoutes.ServeHTTP(res, req, routeData)
		default:
			http.Error(res, head+" not implemented", http.StatusNotImplemented)
		}
	//case "....":
	// There will be other pages.
	// I suspect "manage applications" will be its own page
	// It's possible "/" page is more summary, and /appspaces will be its own page.
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

// getUserData returns a json with {email: ""...""} I think, so far.
func (u *UserRoutes) getUserData(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// check if there is anything in routeData tail?
	user, dsErr := u.UserModel.GetFromID(routeData.Authentication.UserID)
	if dsErr != nil {
		dsErr.HTTPError(res)
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

	dsErr := u.Validator.Password(data.Old)
	if dsErr != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	dsErr = u.Validator.Password(data.New)
	if dsErr != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	user, dsErr := u.UserModel.GetFromID(routeData.Authentication.UserID)
	if dsErr != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, dsErr = u.UserModel.GetFromEmailPassword(user.Email, data.Old)
	if dsErr != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	dsErr = u.UserModel.UpdatePassword(user.UserID, data.New)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	res.WriteHeader(http.StatusOK)
}
