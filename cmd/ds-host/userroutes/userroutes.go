package userroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// UserRoutes handles routes for appspaces.
type UserRoutes struct {
	Authenticator     domain.Authenticator
	AuthRoutes        domain.RouteHandler
	ApplicationRoutes domain.RouteHandler
	Views             domain.Views
	Logger            domain.LogCLientI
}

// ServeHTTP handles http traffic to the user routes
func (u *UserRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	// Consider that apart from login routes, everything else requires authentication
	// Would like to make that abundantly clear in code structure.
	// There should be a single point where we check auth, and if no good, bail.

	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	if head == "signup" || head == "login" {
		u.AuthRoutes.ServeHTTP(res, req, routeData)
	} else {
		u.serveLoggedInRoutes(res, req, routeData)
	}
}

func (u *UserRoutes) serveLoggedInRoutes(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	// ok := u.Authenticator.GetForAccount(res, req, routeData)
	// if !ok {
	// 	fmt.Println("supposed to be authed but is not...")
	// 	return
	// }
	// TODO: do cookies get set???

	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	switch head {
	case "":
		u.Views.UserHome(res)
	case "api":
		// All the async routes essentially?
		head, tail = shiftpath.ShiftPath(tail)
		switch head {
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
