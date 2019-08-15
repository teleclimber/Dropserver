package userroutes

import (
	"encoding/json"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// UserRoutes handles routes for appspaces.
type UserRoutes struct {
	Authenticator     domain.Authenticator
	AuthRoutes        domain.RouteHandler
	ApplicationRoutes domain.RouteHandler
	UserModel         domain.UserModel
	Views             domain.Views
	Logger            domain.LogCLientI
}

// ServeHTTP handles http traffic to the user routes
func (u *UserRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	// Consider that apart from login routes, everything else requires authentication
	// Would like to make that abundantly clear in code structure.
	// There should be a single point where we check auth, and if no good, bail.

	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	if head == "signup" || head == "login" || head == "logout" {
		u.AuthRoutes.ServeHTTP(res, req, routeData)
	} else {
		// Must be logged in to go past this point.
		dsErr := u.Authenticator.AccountAuthorized(res, req, routeData)
		if dsErr == nil {
			u.serveLoggedInRoutes(res, req, routeData)
		} else {
			http.Redirect(res, req, "/login", http.StatusFound)
		}
	}
}

// TODO: user.domain.tld/user.js returns 200 and no useful data. This is wrong.

func (u *UserRoutes) serveLoggedInRoutes(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	if routeData.Cookie.UserAccount == false {
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we dun fogged up
	}

	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	switch head {
	case "":
		u.Views.UserHome(res)
	case "api":
		// All the async routes essentially?
		head, tail = shiftpath.ShiftPath(tail)
		switch head {
		case "user-data":
			u.userData(res, req, routeData)
		case "application": //handle application route (separate file)
			routeData.URLTail = tail
			u.ApplicationRoutes.ServeHTTP(res, req, routeData)
		default:
			http.Error(res, head+" not implemented", http.StatusNotImplemented)
		}
		//case "....":
		// There will be other pages.
		// I suspect "manage applications" will be its own page
		// It's possible "/" page is more summary, and /appspaces will be its own page.
	}
}

// userData returns a json with {email: ""...""} I think, so far.
func (u *UserRoutes) userData(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	user, dsErr := u.UserModel.GetFromID(routeData.Cookie.UserID)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	var userData struct {
		Email string `json:"email"`
	}
	userData.Email = user.Email

	userJSON, err := json.Marshal(userData)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(userJSON)
}
