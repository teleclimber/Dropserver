package appspacerouter

import (
	"errors"
	"net/http"
	"net/url"
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
	AppspaceRouteModels interface {
		GetV0(domain.AppspaceID) domain.V0RouteModel
	}
	VxUserModels interface {
		GetV0(domain.AppspaceID) domain.V0UserModel
	}
	DropserverRoutes domain.RouteHandler // versioned
	SandboxProxy     domain.RouteHandler // versioned?
	AppspaceLogin    interface {
		Create(domain.AppspaceID, url.URL) domain.AppspaceLoginToken
		CheckRedirectToken(string) (domain.AppspaceLoginToken, error)
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

	//... now shift path to get the first param and see if it is dropserver
	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	if head == "dropserver" {
		// handle with dropserver routes handler
		routeData.URLTail = tail
		r.DropserverRoutes.ServeHTTP(&statusRes, req, routeData)
	} else {
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
		if auth != nil {
			cred.ProxyID = auth.ProxyID
			if !r.authorize(routeData, auth) {
				// If requester just logged in with a token but is not authroized, then just show as such.
				// Here we do not redirect because that would cause a redirect loop.
				http.Error(&statusRes, "Route unauthorized for user", http.StatusUnauthorized)
				return
			}
		} else {
			if routeData.Authentication != nil {
				cred.ProxyID = routeData.Authentication.ProxyID
			}
			if !r.authorize(routeData, routeData.Authentication) {
				// if request is for html, then redirect
				// if it's for json response then send an error code?
				u := *req.URL
				u.Host = req.Host // is this OK?
				token := r.AppspaceLogin.Create(routeData.Appspace.AppspaceID, u)
				http.Redirect(&statusRes, req, "//"+r.Config.Exec.UserRoutesDomain+r.Config.Exec.PortString+"/appspacelogin?asl="+token.LoginToken.Token, http.StatusTemporaryRedirect)
				// TODO: this is not right. Fix when we redo appspace logins.
				return
			}
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
	}
}

func (r *V0) processLoginToken(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) (*domain.Authentication, error) {
	loginTokenValues := req.URL.Query()["dropserver-login-token"]
	if len(loginTokenValues) == 0 {
		return nil, nil
	}
	if len(loginTokenValues) > 1 {
		return nil, errors.New("multiple login tokens") // this should translate to http error "bad request"
	}

	token, err := r.AppspaceLogin.CheckRedirectToken(loginTokenValues[0])
	if err != nil {
		//http.Error(res, err.Error(), http.StatusBadRequest)
		// maybe just ignore it? IT could be someone refreshed the page with the token.
		// -> although that possibility is exactly why we should use a separate route to do appspace login
		return nil, err
	}

	if token.AppspaceID != routeData.Appspace.AppspaceID {
		// do nothing? How do we end up in this situation?
		return nil, errors.New("wrong appspace")
	}

	cookieID, err := r.Authenticator.SetForAppspace(res, token.ProxyID, token.AppspaceID, routeData.Appspace.DomainName)
	if err != nil {
		return nil, err
	}

	return &domain.Authentication{
		ProxyID:    token.ProxyID,
		AppspaceID: token.AppspaceID,
		CookieID:   cookieID}, nil
}

func (r *V0) authorize(routeData *domain.AppspaceRouteData, auth *domain.Authentication) bool {
	// And we'll have a bunch of attached creds for request:
	// - userID / contact ID
	// - API key (probably included in header)
	//   -> API key grants permissions
	// - secret link (or not, the secret link is the auth, that's it)

	// [if no user but api key, repeat for api key (look it up, get permissions, find match)]

	if routeData.RouteConfig.Auth.Allow == "public" {
		return true
	}

	if auth == nil || auth.UserAccount || auth.AppspaceID != routeData.Appspace.AppspaceID {
		return false
	}

	if routeData.RouteConfig.Auth.Allow == "authorized" {
		userModel := r.VxUserModels.GetV0(auth.AppspaceID)
		appspaceUser, err := userModel.Get(auth.ProxyID)
		if err != nil {
			return false
		}
		if appspaceUser.ProxyID == "" { // appspace user has zero-value (not found)
			return false
		}

		requiredPermission := routeData.RouteConfig.Auth.Permission
		if requiredPermission == "" {
			// no specific permission required.
			return true
		}
		for _, p := range appspaceUser.Permissions {
			if p == requiredPermission {
				return true
			}
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
