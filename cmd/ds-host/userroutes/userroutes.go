package userroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// route handler for when we know the route is for an app-space.
// Could be proxied to sandbox, or static file, or crud or whatever

// UserRoutes handles routes for appspaces.
type UserRoutes struct {
	ApplicationRoutes domain.RouteHandler
	Logger            domain.LogCLientI
}

// ^^ Also need access to sessions

// ServeHTTP handles http traffic to the user routes
func (u *UserRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	//var ok bool

	// beware that we may need to serve static things like the html for that route?
	// - shiftPath and test head for api,
	// - otherwise pass to static handler?

	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	if head == "api" {
		head, tail = shiftpath.ShiftPath(tail)
		switch head {
		case "application": //handle application route (separate file)
			routeData.URLTail = tail
			u.ApplicationRoutes.ServeHTTP(res, req, routeData)
		default: // bugger not implemented yet?
			http.Error(res, head+" not implemented", http.StatusNotImplemented)
		}

	} else {
		// static server
	}

}
