package appspaceroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// route handler for when we know the route is for an app-space.
// Could be proxied to sandbox, or static file, or crud or whatever

// AppspaceRoutes handles routes for appspaces.
type AppspaceRoutes struct {
	AppModel           domain.AppModel
	AppspaceModel      domain.AppspaceModel
	RouteModelsManager domain.AppspaceRouteModels
	V0                 domain.RouteHandler
	Logger             domain.LogCLientI
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

	// This is where we branch off into different API versions for serving appspace traffic
	r.V0.ServeHTTP(res, req, routeData)

}
