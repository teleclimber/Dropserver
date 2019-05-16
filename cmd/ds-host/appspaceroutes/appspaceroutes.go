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
	Logger           domain.LogCLientI
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
		r.Logger.Log(domain.ERROR, map[string]string{"app-space": appspaceName},
			"Appspace does not exist: "+appspaceName)
		return
	}
	routeData.Appspace = appspace

	//... now shift path to get the first param and see if it is dropserver
	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	if head == "dropserver" {
		// handle with dropserver routes handler
		routeData.URLTail = tail
		r.DropserverRoutes.ServeHTTP(res, req, routeData)
	} else {
		app, dsErr := r.AppModel.GetFromID(appspace.AppID)
		if dsErr != nil {
			//http.Error(res, "App does not exist", http.StatusInternalServerError)
			r.Logger.Log(domain.ERROR, map[string]string{"app-space": appspaceName, "app": appspace.AppName},
				"App does not exist: "+appspace.AppName)
			dsErr.HTTPError(res)
			return
		}
		routeData.App = app

		r.SandboxProxy.ServeHTTP(res, req, routeData)
	}
}
