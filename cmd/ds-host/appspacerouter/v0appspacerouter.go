package appspacerouter

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// securityHeaders is where we would loosen CORS CSP and other headers for an appspace
// if that appspace is set to allow some cross-origin requests
func (a *AppspaceRouter) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// deny deny deny by default
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// do CORS header here too if appspace config allows for loosened access
		// HSTS?
		// HPKP (deprecated) Certificate transparency headers would be good
		next.ServeHTTP(w, r)
	})
}

func (a *AppspaceRouter) routeHit(next http.Handler) http.Handler {
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
			if a.RouteHitEvents != nil {
				a.RouteHitEvents.Send(&e)
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

func (a *AppspaceRouter) loadRouteConfig(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)

		route, err := a.AppRoutes.Match(appspace.AppID, appspace.AppVersion, r.Method, r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if route == (domain.AppRoute{}) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		ctx = domain.CtxWithRouteConfig(ctx, route)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *AppspaceRouter) loadAppspaceUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)
		proxyID, ok := domain.CtxAppspaceUserProxyID(ctx)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		appspaceUser, err := a.AppspaceUserModel.Get(appspace.AppspaceID, proxyID)
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

func (a *AppspaceRouter) authorizeRoute(next http.Handler) http.Handler {
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
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if routeConfig.Auth.Permission == "" {
			// no specific permission required.
			next.ServeHTTP(w, r)
			return
		}
		for _, p := range user.Permissions {
			if p == routeConfig.Auth.Permission {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.WriteHeader(http.StatusForbidden)
	})
}

// ServeHTTP handles http traffic to the appspace
func (a *AppspaceRouter) handleRoute(w http.ResponseWriter, r *http.Request) {
	routeConfig, _ := domain.CtxRouteConfig(r.Context())
	switch routeConfig.Type {
	case "function":
		a.SandboxProxy.ServeHTTP(w, r)
	case "static":
		a.serveFile(w, r)
	default:
		appspace, _ := domain.CtxAppspaceData(r.Context())
		a.getLogger(appspace.AppspaceID).Log("route type not implemented: " + routeConfig.Type)
		http.Error(w, "route type not implemented", http.StatusInternalServerError)
	}
}

// serveFile serves a file based on route config and the path of the request
// Possible scenarios:
// - config points to file -> serve that file
// - config points to dir:
//   - join request wildcard path (if config request is wildcard)
//   - if this points to file, serve file
//   - if dir, look for index.html? or ...?
func (a *AppspaceRouter) serveFile(w http.ResponseWriter, r *http.Request) {
	p, err := a.getConfigPath(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fileinfo, err := os.Stat(p)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// If the config points to a file, then always just serve that.
	if !fileinfo.IsDir() {
		serveFile(w, r, p)
		return
	}

	// Otherwise append request wildcard path and stat again
	p, err = a.joinBaseToRequest(p, r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fileinfo, err = os.Stat(p)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	//....
	if fileinfo.IsDir() {
		// either append index.html to p or send dir listing
		w.WriteHeader(http.StatusNotFound)
		return
	}

	serveFile(w, r, p)
}

// getConfigPath returns the actual path that the route config options intends to serve
// From route config + appspace/app locations, determine the path of the desired file on local system
func (a *AppspaceRouter) getConfigPath(r *http.Request) (string, error) {
	ctx := r.Context()
	routeConfig, _ := domain.CtxRouteConfig(ctx)
	root := ""
	p := routeConfig.Options.Path
	if strings.HasPrefix(p, "@appspace/") {
		appspace, ok := domain.CtxAppspaceData(ctx)
		if !ok {
			panic("appspaceRouter getFilePath: expected an appspace")
		}
		p = strings.TrimPrefix(p, "@appspace/")
		root = a.AppspaceLocation2Path.Files(appspace.LocationKey)
	} else if strings.HasPrefix(p, "@avatars/") {
		appspace, ok := domain.CtxAppspaceData(ctx)
		if !ok {
			panic("appspaceRouter getFilePath: expected an appspace")
		}
		p = strings.TrimPrefix(p, "@avatars/")
		root = a.AppspaceLocation2Path.Avatars(appspace.LocationKey)
	} else if strings.HasPrefix(p, "@app/") {
		appVersion, ok := domain.CtxAppVersionData(ctx)
		if !ok {
			panic("appspaceRouter getFilePath: expected an app version")
		}
		p = strings.TrimPrefix(p, "@app/")
		root = a.AppLocation2Path.Files(appVersion.LocationKey)
	} else {
		appspace, _ := domain.CtxAppspaceData(r.Context())
		a.getLogger(appspace.AppspaceID).Log("getFilePath() Path prefix not recognized: " + p) // This should be logged to appspace log, not general log
		return "", errors.New("path prefix not recognized")
	}

	// p is from app so untrusted. Check it doesn't breach the appspace or app root:
	configPath := filepath.Join(root, filepath.FromSlash(p))
	if !strings.HasPrefix(configPath, root) {
		appspace, _ := domain.CtxAppspaceData(r.Context())
		a.getLogger(appspace.AppspaceID).Log("getFilePath() route config path out of bounds: " + root)
		return "", errors.New("route config path out of bounds")
	}

	return configPath, nil
}

func (a *AppspaceRouter) joinBaseToRequest(basePath string, r *http.Request) (string, error) {
	ctx := r.Context()
	routeConfig, _ := domain.CtxRouteConfig(ctx)
	// Now determine the path of the file requested from the url
	urlTail := filepath.FromSlash(r.URL.Path)
	// have to remove the prefix from urltail.
	urlPrefix := routeConfig.Path.Path
	urlTail = strings.TrimPrefix(urlTail, urlPrefix) // doing this with strings like that is super error-prone!

	joinedPath := filepath.Join(basePath, urlTail)

	if !strings.HasPrefix(joinedPath, basePath) {
		return "", errors.New("path out of bounds")
	}

	return joinedPath, nil
}

func serveFile(w http.ResponseWriter, r *http.Request, p string) {
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.Error(w, "file open error", http.StatusInternalServerError)
		return
	}

	fileinfo, err := f.Stat()
	if err != nil {
		http.Error(w, "file stat error", http.StatusInternalServerError)
	}

	http.ServeContent(w, r, p, fileinfo.ModTime(), f)
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
