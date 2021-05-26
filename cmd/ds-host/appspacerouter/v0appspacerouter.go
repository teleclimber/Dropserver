package appspacerouter

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
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
	SandboxProxy   http.Handler // versioned?
	V0TokenManager interface {
		CheckToken(token string) (domain.V0AppspaceLoginToken, bool)
	}
	Authenticator interface {
		// should have ProcessLoginToken instead. Also removes dep on Token Manager
		// Exepct all this stuff is versioned.
		SetForAppspace(http.ResponseWriter, domain.ProxyID, domain.AppspaceID, string) (string, error)
	}
	RouteHitEvents interface {
		Send(*domain.AppspaceRouteHitEvent)
	}
	Config *domain.RuntimeConfig

	mux *chi.Mux
}

func (arV0 *V0) Init() {
	// basically it's all middleware
	// then a branch on handling via static file serve or sandbox
	// - route hit event
	// - get route config
	// - process login token
	// - load user
	// - authorize
	// - next handler splits on function versus static

	mux := chi.NewRouter()
	mux.Use(arV0.routeHit, arV0.loadRouteConfig, arV0.processLoginToken, arV0.loadAppspaceUser, arV0.authorizeRoute)

	mux.Handle("/*", http.HandlerFunc(arV0.handleRoute))

	arV0.mux = mux
}

func (arV0 *V0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	arV0.mux.ServeHTTP(w, r)
}

func (arV0 *V0) routeHit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statusW := statusRecorder{w, 0}
		next.ServeHTTP(&statusW, r)

		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)
		proxyID, _ := domain.CtxAppspaceUserProxyID(ctx)
		routeConfig, _ := domain.CtxRouteConfig(ctx)
		cred := struct {
			ProxyID domain.ProxyID
		}{proxyID}

		defer func(e domain.AppspaceRouteHitEvent) {
			if arV0.RouteHitEvents != nil {
				arV0.RouteHitEvents.Send(&e)
			}
		}(domain.AppspaceRouteHitEvent{
			AppspaceID:  appspace.AppspaceID,
			Request:     r,
			RouteConfig: &routeConfig,
			Credentials: cred,
			Authorized:  statusW.status != http.StatusForbidden, //redundant with status?
			Status:      statusW.status})
	})
}

func (arV0 *V0) loadRouteConfig(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)

		routeModel := arV0.AppspaceRouteModels.GetV0(appspace.AppspaceID)
		routeConfig, err := routeModel.Match(r.Method, r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if routeConfig == nil {
			http.Error(w, "No matching route", http.StatusNotFound)
			return
		}

		ctx = domain.CtxWithRouteConfig(ctx, *routeConfig)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (arV0 *V0) processLoginToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginTokenValues := r.URL.Query()["dropserver-login-token"]
		if len(loginTokenValues) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		if len(loginTokenValues) > 1 {
			http.Error(w, "multiple login tokens", http.StatusBadRequest)
			return
		}

		token, ok := arV0.V0TokenManager.CheckToken(loginTokenValues[0]) //TODO CheckToken should take an appspace ID, naturally.
		if !ok {
			// no matching token is not an error. It can happen is user reloads the page for ex.
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)

		if token.AppspaceID != appspace.AppspaceID {
			// do nothing? How do we end up in this situation?
			// This is alarming. Log it, but continue on as if no matching token.
			arV0.getLogger("processLoginToken").AppspaceID(appspace.AppspaceID).Log(fmt.Sprintf("Got token for wrong appspace: %v", token))
			next.ServeHTTP(w, r)
			return
		}

		cookieID, err := arV0.Authenticator.SetForAppspace(w, token.ProxyID, token.AppspaceID, appspace.DomainName)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		ctx = domain.CtxWithAppspaceUserProxyID(ctx, token.ProxyID)
		ctx = domain.CtxWithSessionID(ctx, cookieID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (arV0 *V0) loadAppspaceUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)
		proxyID, ok := domain.CtxAppspaceUserProxyID(ctx)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		appspaceUser, err := arV0.AppspaceUserModel.Get(appspace.AppspaceID, proxyID)
		if err != nil {
			if err == sql.ErrNoRows {
				next.ServeHTTP(w, r)
			} else {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}

		ctx = domain.CtxWithAppspaceUserData(ctx, appspaceUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (arV0 *V0) authorizeRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// And we'll have a bunch of attached creds for request:
		// - userID / contact ID
		// - API key (probably included in header)
		//   -> API key grants permissions
		// - secret link (or not, the secret link is the auth, that's it)

		// [if no user but api key, repeat for api key (look it up, get permissions, find match)]
		ctx := r.Context()
		routeConfig, _ := domain.CtxRouteConfig(ctx)

		// if it's public route, then always authorized
		if routeConfig.Auth.Allow == "public" {
			next.ServeHTTP(w, r)
			return
		}

		user, ok := domain.CtxAppspaceUserData(ctx)
		if !ok {
			// no user, and route is not public, so forbidden
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if routeConfig.Auth.Permission == "" {
			// no specific permission required.
			next.ServeHTTP(w, r)
			return
		}
		for _, p := range strings.Split(user.Permissions, ",") {
			if p == routeConfig.Auth.Permission {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "forbidden", http.StatusForbidden)
	})
}

// ServeHTTP handles http traffic to the appspace
func (arV0 *V0) handleRoute(w http.ResponseWriter, r *http.Request) {
	routeConfig, _ := domain.CtxRouteConfig(r.Context())
	switch routeConfig.Handler.Type {
	case "function":
		arV0.SandboxProxy.ServeHTTP(w, r)
	case "file":
		arV0.serveFile(w, r)
	default:
		arV0.getLogger("ServeHTTP").Log("route type not implemented: " + routeConfig.Handler.Type)
		http.Error(w, "route type not implemented", http.StatusInternalServerError)
	}
}

func (arV0 *V0) serveFile(w http.ResponseWriter, r *http.Request) {
	// - left trim request path
	p, err := arV0.getFilePath(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check if p is a directory? If so handle as either dir listing or index.html?
	fileinfo, err := os.Stat(p)
	if err != nil {
		http.Error(w, "file does not exist", http.StatusNotFound)
		return
	}

	if fileinfo.IsDir() {
		// either append index.html to p or send dir listing
		http.Error(w, "path is a directory", http.StatusNotFound)
		return
	}

	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "file does not exist", http.StatusNotFound)
			return
		}
		http.Error(w, "file open error", http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, p, fileinfo.ModTime(), f)
}

// Need: request path, route handler path,  appspace location key, app version location key,
// It seems there are two parts to this:
// - left-trim request path to route handler path, so all that's left is the extra
// - then tack on the
func (arV0 *V0) getFilePath(r *http.Request) (string, error) {
	ctx := r.Context()
	routeConfig, _ := domain.CtxRouteConfig(ctx)
	// from route config + appspace/app locations, determine the path of the desired file on local system
	root := ""
	p := routeConfig.Handler.Path
	if strings.HasPrefix(p, "@appspace/") {
		appspace, ok := domain.CtxAppspaceData(ctx)
		if !ok {
			panic("v0appspaceRouter getFilePath: expected an appspace")
		}
		p = strings.TrimPrefix(p, "@appspace/")
		root = filepath.Join(arV0.Config.Exec.AppspacesPath, appspace.LocationKey, "data", "files")
	} else if strings.HasPrefix(p, "@app/") {
		appVersion, ok := domain.CtxAppVersionData(ctx)
		if !ok {
			panic("v0appspaceRouter getFilePath: expected an app version")
		}
		p = strings.TrimPrefix(p, "@app/")
		root = filepath.Join(arV0.Config.Exec.AppsPath, appVersion.LocationKey)
	} else {
		arV0.getLogger("getFilePath").Log("Path prefix not recognized: " + p) // This should be logged to appspace log, not general log
		return "", errors.New("path prefix not recognized")
	}

	// p is from app so untrusted. Check it doesn't breach the appspace or app root:
	serveRoot := filepath.Join(root, filepath.FromSlash(p))
	if !strings.HasPrefix(serveRoot, root) {
		arV0.getLogger("getFilePath").Log("route config path out of bounds: " + root)
		return "", errors.New("route config path out of bounds")
	}

	// Now determine the path of the file requested from the url
	urlTail := filepath.FromSlash(r.URL.Path)
	// have to remove the prefix from urltail.
	urlPrefix := routeConfig.Path
	urlTail = strings.TrimPrefix(urlTail, urlPrefix) // doing this with strings like that is super error-prone!

	p = filepath.Join(serveRoot, urlTail)

	if !strings.HasPrefix(p, serveRoot) {
		return "", errors.New("path out of bounds")
	}

	return p, nil
}

func (arV0 *V0) getLogger(note string) *record.DsLogger {
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
