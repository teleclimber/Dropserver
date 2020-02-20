package appspaceroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// route handler for when we know the route is for an app-space.
// Could be proxied to sandbox, or static file, or crud or whatever

// AppspaceRoutes handles routes for appspaces.
type AppspaceRoutes struct {
	AppModel         domain.AppModel
	AppspaceModel    domain.AppspaceModel
	ASRoutesModel    domain.ASRoutesModel
	DropserverRoutes domain.RouteHandler
	SandboxProxy     domain.RouteHandler
	Logger           domain.LogCLientI
}

// ^^ Also need access to sessions

// ServeHTTP handles http traffic to the appspace
func (r *AppspaceRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	subdomains := *routeData.Subdomains
	appspaceSubdomain := subdomains[len(subdomains)-1]

	appspace, dsErr := r.AppspaceModel.GetFromSubdomain(appspaceSubdomain)
	if dsErr != nil && dsErr.Code() == dserror.NoRowsInResultSet {
		http.Error(res, "Appspace does not exist", http.StatusNotFound)
		r.Logger.Log(domain.ERROR, map[string]string{"app-space": appspaceSubdomain},
			"Appspace does not exist: "+appspaceSubdomain)
		return
	} else if dsErr != nil {
		dsErr.HTTPError(res)
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
		if dsErr != nil { // do we differentiate between empty result vs other errors?
			r.Logger.Log(domain.ERROR, map[string]string{"app-space": appspaceSubdomain, "app": string(appspace.AppID)},
				"App does not exist: "+string(appspace.AppID))
			dsErr.HTTPError(res)
			return
		}
		routeData.App = app

		appVersion, dsErr := r.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
		if dsErr != nil {
			// is this an internal error? It seems that if an appspace is using a version, that version has to exist?!?
			r.Logger.Log(domain.ERROR, map[string]string{"app-space": appspaceSubdomain, "app": string(appspace.AppID)},
				"App version does not exist: "+string(appspace.AppVersion))
			http.Error(res, "App Version not found", http.StatusInternalServerError)
			return
		}
		routeData.AppVersion = appVersion

		routeConfig, dsErr := r.ASRoutesModel.GetRouteConfig(appVersion, req.Method, routeData.URLTail)
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

		switch routeConfig.Type {
		case "function":
			r.SandboxProxy.ServeHTTP(res, req, routeData)
		default:
			http.Error(res, "route type not implemented", http.StatusInternalServerError)
		}
	}
}
