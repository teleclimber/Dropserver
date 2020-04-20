package appspaceroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// route handler for when we know the route is for an app-space.
// Could be proxied to sandbox, or static file, or crud or whatever

// V0 handles routes for appspaces.
type V0 struct {
	AppspaceRouteModels domain.AppspaceRouteModels
	DropserverRoutes    domain.RouteHandler // versioned
	SandboxProxy        domain.RouteHandler // versioned?
	Logger              domain.LogCLientI
}

// ^^ Also need access to sessions

// ServeHTTP handles http traffic to the appspace
func (r *V0) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {

	//... now shift path to get the first param and see if it is dropserver
	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	if head == "dropserver" {
		// handle with dropserver routes handler
		routeData.URLTail = tail
		r.DropserverRoutes.ServeHTTP(res, req, routeData)
	} else {
		routeModel := r.AppspaceRouteModels.GetV0(routeData.Appspace.AppspaceID)
		routeConfig, dsErr := routeModel.Match(req.Method, routeData.URLTail)
		if dsErr != nil {
			//..if not found then go 404 ... or do that automatically from errors?
			// here we don't log (for now) because it's not a system error to have the wrong route requested
			// Though the app owner may be interested in seeing the errors?
			dsErr.HTTPError(res)
			return
		}
		routeData.RouteConfig = routeConfig

		// TODO: check if appspace is paused
		// .. no need to go further if paused bit is set, I think.

		// Do we also check if there is a job started? Like migration?
		// Why go any further if there is? (in future maybe we can wait for migration to end and pass request then)

		// TODO: auth.
		// r.Authenticator.ForAppspace(appspaceID)
		// Consider that there are a number of possible results here:
		// - no cookie
		// - cookie but not authorized on this route (non-owner)
		// - non-owner, but authorized

		switch routeConfig.Handler.Type {
		case "function":
			r.SandboxProxy.ServeHTTP(res, req, routeData)
		default:
			http.Error(res, "route type not implemented", http.StatusInternalServerError)
		}
	}
}
