package appspacerouter

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// route handler for when we know the route is for an app-space.
// Could be proxied to sandbox, or static file, or crud or whatever

// V0 handles routes for appspaces.
type V0 struct {
	AppspaceRouteModels interface {
		GetV0(domain.AppspaceID) domain.V0RouteModel
	}
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
	}
	SandboxProxy   domain.RouteHandler // versioned?
	V0TokenManager interface {
		CheckToken(token string) (domain.V0AppspaceLoginToken, bool)
	}
	Authenticator interface {
		SetForAppspace(http.ResponseWriter, domain.ProxyID, domain.AppspaceID, string) (string, error)
	}
	RouteHitEvents interface {
		Send(*domain.AppspaceRouteHitEvent)
	}
	Config *domain.RuntimeConfig
}

// ServeHTTP handles http traffic to the appspace
func (r *V0) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	statusRes := statusRecorder{res, 0}

	cred := struct {
		ProxyID domain.ProxyID
	}{}
	authorized := false

	defer func() {
		if r.RouteHitEvents != nil {
			r.RouteHitEvents.Send(&domain.AppspaceRouteHitEvent{
				AppspaceID:  routeData.Appspace.AppspaceID,
				Request:     req,
				RouteConfig: routeData.RouteConfig,
				Credentials: cred,
				Authorized:  authorized,
				Status:      statusRes.status})
		}
	}()

	routeModel := r.AppspaceRouteModels.GetV0(routeData.Appspace.AppspaceID)
	routeConfig, err := routeModel.Match(req.Method, routeData.URLTail)
	if err != nil {
		http.Error(&statusRes, err.Error(), http.StatusInternalServerError)
		return
	}

	if routeConfig == nil {
		http.Error(&statusRes, "No matching route", http.StatusNotFound)
		return
	}
	routeData.RouteConfig = routeConfig

	auth, err := r.processLoginToken(&statusRes, req, routeData)
	if err != nil {
		http.Error(&statusRes, err.Error(), http.StatusInternalServerError)
		return
	}
	if auth.Authenticated {
		cred.ProxyID = auth.ProxyID
	} else if routeData.Authentication != nil {
		cred.ProxyID = routeData.Authentication.ProxyID
		auth = *routeData.Authentication
	}

	appspaceUser := domain.AppspaceUser{}
	if auth.Authenticated {
		appspaceUser, _ = r.AppspaceUserModel.Get(auth.AppspaceID, auth.ProxyID)
		// don't check error here. If user not found, will be dealt with in authorize functions below
	}

	if !r.authorizeAppspace(routeData, auth, appspaceUser) {
		// We probably need different messages for different situations:
		// - "unable to serve this page" if no auth or no such appspace (to avoid probing for appspace domains)
		// - "Not permitted" if user has proper auth for appspace, but not the required permission.
		http.Error(&statusRes, "No page to show", http.StatusUnauthorized)
		return
	}

	if !r.authorizePermission(routeData.RouteConfig.Auth.Permission, appspaceUser) {
		http.Error(&statusRes, "Not permitted", http.StatusUnauthorized)
		return
	}

	// if you got this far, route is authorized.
	authorized = true

	switch routeConfig.Handler.Type {
	case "function":
		r.SandboxProxy.ServeHTTP(&statusRes, req, routeData)
	case "file":
		r.serveFile(&statusRes, req, routeData)
	// case "db":
	// call to appspacedb. Since this is v0appspaceroutes, then the call will be to v0appspacedb
	default:
		r.getLogger("ServeHTTP").Log("route type not implemented: " + routeConfig.Handler.Type)
		http.Error(&statusRes, "route type not implemented", http.StatusInternalServerError)
	}
	//}
}

func (r *V0) processLoginToken(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) (domain.Authentication, error) {
	loginTokenValues := req.URL.Query()["dropserver-login-token"]
	if len(loginTokenValues) == 0 {
		return domain.Authentication{}, nil
	}
	if len(loginTokenValues) > 1 {
		return domain.Authentication{}, errors.New("multiple login tokens") // this should translate to http error "bad request"
	}

	token, ok := r.V0TokenManager.CheckToken(loginTokenValues[0]) // this looks corect but is badly named?
	if !ok {
		return domain.Authentication{}, nil // no matching token is not an error. It can happen is user reloads the page for ex.
	}

	if token.AppspaceID != routeData.Appspace.AppspaceID {
		// do nothing? How do we end up in this situation?
		return domain.Authentication{}, errors.New("wrong appspace")
	}

	cookieID, err := r.Authenticator.SetForAppspace(res, token.ProxyID, token.AppspaceID, routeData.Appspace.DomainName)
	if err != nil {
		return domain.Authentication{}, err
	}

	return domain.Authentication{
		Authenticated: true,
		ProxyID:       token.ProxyID,
		AppspaceID:    token.AppspaceID,
		CookieID:      cookieID}, nil
}

func (r *V0) authorizeAppspace(routeData *domain.AppspaceRouteData, auth domain.Authentication, user domain.AppspaceUser) bool {
	// And we'll have a bunch of attached creds for request:
	// - userID / contact ID
	// - API key (probably included in header)
	//   -> API key grants permissions
	// - secret link (or not, the secret link is the auth, that's it)

	// [if no user but api key, repeat for api key (look it up, get permissions, find match)]

	// if it's public route, then always authorized
	if routeData.RouteConfig.Auth.Allow == "public" {
		return true
	}

	// if not public, then route requires auth
	if !auth.Authenticated || auth.UserAccount || auth.AppspaceID != routeData.Appspace.AppspaceID {
		return false
	}

	if user == (domain.AppspaceUser{}) { // appspace user has zero-value (not found)
		return false
	}

	return true
}

func (r *V0) authorizePermission(requiredPermission string, user domain.AppspaceUser) bool {
	if requiredPermission == "" {
		// no specific permission required.
		return true
	}
	for _, p := range strings.Split(user.Permissions, ",") {
		if p == requiredPermission {
			return true
		}
	}

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
		http.Error(res, "file does not exist", http.StatusNotFound)
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
		root = filepath.Join(r.Config.Exec.AppspacesPath, routeData.Appspace.LocationKey, "files")
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

// See https://upgear.io/blog/golang-tip-wrapping-http-response-writer-for-middleware/

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}
