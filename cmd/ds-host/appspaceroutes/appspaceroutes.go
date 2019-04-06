package appspaceroutes

import (
	"fmt"
	"net/http"
	"strings"

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

	appspaceName, ok := getAppspaceName(req.Host)
	if !ok {
		http.Error(res, "Error getting appspace from host string", http.StatusInternalServerError)
		//TODO log an error please
	}

	var appspace *domain.Appspace
	if ok {
		// use appspace model to get
		fmt.Println(appspaceName)

		appspace, ok = r.AppspaceModel.GetForName(appspaceName)
		if !ok {
			http.Error(res, "Appspace does not exist", http.StatusNotFound)
		} else {
			routeData.Appspace = appspace
		}
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

func getAppspaceName(host string) (appspace string, ok bool) {
	// here we need to know something about the configuration
	// how many domain levels to ignore?
	// also, consider that it might be a third-party domain.
	// so: explode host into pieces,
	// ..walk known host domain pieces [org, dropserver]
	// at end of walk if still matching, that first one is your app-space
	// ... for now. we may do appspace.username.dropserver.org later?

	// this may need to be pulled out and put in top level server handler,
	// ..since it will have to detect user.<root-domain>, etc.

	ok = true

	rootHost := [2]string{"develop", "dropserver"} //TODO do not hard-code this, obviously
	numRoot := 2

	host = strings.Split(host, ":")[0] // in case host includes port
	hostPieces := strings.Split(host, ".")
	numPieces := len(hostPieces)

	if numPieces <= numRoot {
		ok = false
	} else {
		for i, p := range rootHost {
			if hostPieces[numPieces-i-1] != p {
				ok = false
				break
			}
		}
	}

	if ok {
		appspace = hostPieces[numPieces-numRoot-1]
	}

	return
}
