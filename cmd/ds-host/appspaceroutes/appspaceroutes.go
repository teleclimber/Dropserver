package appspaceroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// route handler for when we know the route is for an app-space.
// Could be proxied to sandbox, or static file, or crud or whatever

// AppspaceRoutes handles routes for appspaces.
type AppspaceRoutes struct {
	AppModel         domain.AppModel
	AppspaceModel    domain.AppspaceModel
	DropserverRoutes domain.RouteHandler
	SandboxProxy     domain.RouteHandler
	// TODO have a logger please
}

// ^^ Also need access to sessions

// ServeHTTP handles http traffic to the appspace
func (r *AppspaceRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	var ok bool

	subdomains := *routeData.Subdomains
	appspaceName := subdomains[len(subdomains)-1]

	appspace, ok := r.AppspaceModel.GetForName(appspaceName)
	if !ok {
		http.Error(res, "Appspace does not exist", http.StatusNotFound)
	} else {
		routeData.Appspace = appspace
	}

	var app *domain.App
	if ok {
		//... now shift path to get the first param and see if it is dropserver
		head, tail := shiftpath.ShiftPath(routeData.URLTail)
		if head == "dropserver" {
			// handle with dropserver routes handler
			routeData.URLTail = tail
			r.DropserverRoutes.ServeHTTP(res, req, routeData)
		} else {
			app, ok = r.AppModel.GetForName(appspace.AppName)
			if !ok {
				http.Error(res, "App does not exist", http.StatusInternalServerError)
				//TODO log this for sure
			} else {
				routeData.App = app
			}
		}
	}

	if ok && app != nil {
		// get config, match route, do auth, route type switch
		// lots of stuff here that we won't have implemented for a while.

		// For now assume it goes to sandbox
		r.SandboxProxy.ServeHTTP(res, req, routeData)

	}
}
