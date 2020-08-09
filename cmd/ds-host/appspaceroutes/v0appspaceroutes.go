package appspaceroutes

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// route handler for when we know the route is for an app-space.
// Could be proxied to sandbox, or static file, or crud or whatever

// V0 handles routes for appspaces.
type V0 struct {
	AppspaceRouteModels domain.AppspaceRouteModels
	DropserverRoutes    domain.RouteHandler // versioned
	SandboxProxy        domain.RouteHandler // versioned?
	Config              *domain.RuntimeConfig
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
		routeConfig, err := routeModel.Match(req.Method, routeData.URLTail)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if routeConfig == nil {
			http.Error(res, "No matching route", http.StatusNotFound)
			return
		}
		routeData.RouteConfig = routeConfig

		if !r.authorize(routeData) {
			// for now just send unauthorized
			http.Error(res, "not authorized", http.StatusUnauthorized)
			return
		}

		switch routeConfig.Handler.Type {
		case "function":
			r.SandboxProxy.ServeHTTP(res, req, routeData)
		case "file":
			r.serveFile(res, req, routeData)
		default:
			r.getLogger("ServeHTTP").Log("route type not implemented: " + routeConfig.Handler.Type)
			http.Error(res, "route type not implemented", http.StatusInternalServerError)
		}
	}
}

func (r *V0) authorize(routeData *domain.AppspaceRouteData) bool {
	switch routeData.RouteConfig.Auth.Type {
	case "owner":
		if routeData.Cookie == nil {
			return false
		}
		if routeData.Cookie.UserID != routeData.Appspace.OwnerID {
			return false
		}
		if routeData.Cookie.AppspaceID != routeData.Appspace.AppspaceID {
			return false
		}
		return true
	case "public":
		return true
	}

	r.getLogger("authorize").Log("Unrecognized route config auth type: " + routeData.RouteConfig.Auth.Type)

	return false
}

func (r *V0) serveFile(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	p, err := r.getFilePath(routeData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// check if p is a directory? If so handle as either dir listing or index.html?
	fileinfo, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(res, "file does not exist", http.StatusNotFound)
			return
		}
		// consider logging
		http.Error(res, "file stat error", http.StatusInternalServerError)
		return
	}

	if fileinfo.IsDir() {
		// either append index.html to p or send dir listing
		http.Error(res, "path is a directory", http.StatusNotFound)
		return
	}

	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(res, "file does not exist", http.StatusNotFound)
			return
		}
		http.Error(res, "file open error", http.StatusInternalServerError)
		return
	}

	http.ServeContent(res, req, p, fileinfo.ModTime(), f)
}

func (r *V0) getFilePath(routeData *domain.AppspaceRouteData) (string, error) {
	// from route config + appspace/app locations, determine the path of the desired file on local system
	routeHandler := routeData.RouteConfig.Handler
	root := ""
	p := routeHandler.Path
	if strings.HasPrefix(p, "@appspace/") {
		p = strings.TrimPrefix(p, "@appspace/")
		root = filepath.Join(r.Config.Exec.AppspacesFilesPath, routeData.Appspace.LocationKey)
	} else if strings.HasPrefix(p, "@app/") {
		p = strings.TrimPrefix(p, "@app/")
		root = filepath.Join(r.Config.Exec.AppsPath, routeData.AppVersion.LocationKey)
	} else {
		r.getLogger("getFilePath").Log("Path prefix not recognized: " + p)
		return "", errors.New("path prefix not recognized")
	}

	// p is from app so untrusted. Check it doesn't breach the appspace or app root:
	serveRoot := filepath.Join(root, filepath.FromSlash(p))
	if !strings.HasPrefix(serveRoot, root) {
		r.getLogger("getFilePath").Log("route config path out of bounds: " + root)
		return "", errors.New("route config path out of bounds")
	}

	// Now determine the path of the file requested from the url
	// I think UrlTail comes into play here.
	urlTail := filepath.FromSlash(routeData.URLTail)
	// have to remove the prefix from urltail.
	urlPrefix := routeData.RouteConfig.Path
	urlTail = strings.TrimPrefix(urlTail, urlPrefix) // doing this with strings like that is super error-prone!

	p = filepath.Join(serveRoot, urlTail)

	if !strings.HasPrefix(p, serveRoot) {
		return "", errors.New("path out of bounds")
	}

	return p, nil
}

func (r *V0) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0AppspaceRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
