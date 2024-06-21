package appspacerouter

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// Hmm, this is kind of a misnomer now?
// This handles all requests that are not pointed to the ds-host user domain.
// So dropids and appspaces, and maybe even other things?
// Since dropid and apppsace domains can overlap, OK to handle them together

// AppspaceRouter handles routes for appspaces.
type AppspaceRouter struct {
	Config   *domain.RuntimeConfig `checkinject:"required"`
	AppModel interface {
		GetFromID(domain.AppID) (domain.App, error)
		GetVersion(domain.AppID, domain.Version) (domain.AppVersion, error)
	} `checkinject:"required"`
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
	} `checkinject:"required"`
	AppspaceStatus interface {
		Ready(domain.AppspaceID) bool
	} `checkinject:"required"`
	DropserverRoutes interface {
		Router() http.Handler
	} `checkinject:"required"`
	AppRoutes interface {
		Match(appID domain.AppID, version domain.Version, method string, reqPath string) (domain.AppRoute, error)
	} `checkinject:"required"`
	SandboxProxy   http.Handler `checkinject:"required"` // versioned?
	RouteHitEvents interface {
		Send(*domain.AppspaceRouteHitEvent)
	} `checkinject:"optional"`
	AppLocation2Path interface {
		Files(string) string
	} `checkinject:"required"`
	AppspaceLocation2Path interface {
		Files(string) string
		Avatars(string) string
	} `checkinject:"required"`

	liveCounterMux sync.Mutex
	liveCounter    map[domain.AppspaceID]int

	subscribersMux sync.Mutex
	subscribers    map[domain.AppspaceID][]chan<- int
}

// Init initializes the router
func (a *AppspaceRouter) Init() {
	a.liveCounter = make(map[domain.AppspaceID]int)
	a.subscribers = make(map[domain.AppspaceID][]chan<- int)
}

func (a *AppspaceRouter) BuildRoutes(mux *chi.Mux) {
	mux.Use(a.errorPage)
	mux.Use(a.appspaceAvailable, a.countRequest)
	mux.Use(a.loadApp)
	mux.Mount("/.dropserver", a.DropserverRoutes.Router())
	mux.Route("/", func(r chi.Router) {
		r.Use(a.securityHeaders)
		r.Use(a.loadRouteConfig, a.routeHit, a.loadAppspaceUser, a.authorizeRoute)
		r.Handle("/*", http.HandlerFunc(a.handleRoute))
	})
}

func (a *AppspaceRouter) appspaceAvailable(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appspace, ok := domain.CtxAppspaceData(r.Context())
		if !ok {
			panic("expected appspace to exist on request context")
		}
		if !a.AppspaceStatus.Ready(appspace.AppspaceID) {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *AppspaceRouter) countRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appspace, ok := domain.CtxAppspaceData(r.Context())
		if !ok {
			panic("countRequest: expected appspace to exist on request context")
		}
		a.incrementLiveCount(appspace.AppspaceID)
		next.ServeHTTP(w, r)
		a.decrementLiveCount(appspace.AppspaceID)
	})
}

func (a *AppspaceRouter) loadApp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appspace, ok := domain.CtxAppspaceData(r.Context())
		if !ok {
			panic("loadApp: expected appspace to exist on request context")
		}

		app, err := a.AppModel.GetFromID(appspace.AppID)
		if err != nil { // do we differentiate between empty result vs other errors? -> No, if any kind of DB error occurs, the DB or model will log it.
			a.getLogger(appspace.AppspaceID).AddNote("AppModel.GetFromID").Error(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		ctx := domain.CtxWithAppData(r.Context(), app)

		appVersion, err := a.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
		if err != nil {
			a.getLogger(appspace.AppspaceID).AddNote("AppModel.GetVersion").Error(err)
			http.Error(w, "App Version not found", http.StatusInternalServerError)
			return
		}
		ctx = domain.CtxWithAppVersionData(ctx, appVersion)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// errorPage shows an HTML error page for certian statuses
func (a *AppspaceRouter) errorPage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statusW := statusRecorder{w, 0}
		next.ServeHTTP(&statusW, r)
		s := statusW.status
		if s != http.StatusForbidden && s != http.StatusNotFound && s != http.StatusServiceUnavailable {
			return
		}
		if !strings.Contains(r.Header.Get("accept"), "text/html") {
			return
		}

		switch statusW.status {
		case http.StatusForbidden:
			_, hasAuth := domain.CtxAppspaceUserProxyID(r.Context())
			a.forbiddenPage(w, hasAuth)
		case http.StatusNotFound:
			a.routeNotFoundPage(w)
		case http.StatusServiceUnavailable:
			a.unavailablePage(w)
		}
	})
}

func notFoundPage(w http.ResponseWriter) {
	setHTMLHeader(w)
	w.Write([]byte("<h1>404 Not Found</h1>"))
}

func (a *AppspaceRouter) routeNotFoundPage(w http.ResponseWriter) {
	setHTMLHeader(w)
	w.Write([]byte("<h1>404 Not Found</h1><p>This page was not found in this appspace</p>"))
}

func (a *AppspaceRouter) forbiddenPage(w http.ResponseWriter, p bool) {
	setHTMLHeader(w)
	w.Write([]byte("<h1>403 Forbidden</h1>"))
	if p {
		w.Write([]byte("<p>Insufficient permissions</p>"))
	} else {
		w.Write([]byte("<p>You may need to log in</p>"))
	}
}

func (a *AppspaceRouter) unavailablePage(w http.ResponseWriter) {
	setHTMLHeader(w)
	w.Write([]byte("<h1>503 Service Unavailable</h1><p>Appspace may be undergoing maintenance and should be back soon</p>"))
}

func setHTMLHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

func (a *AppspaceRouter) incrementLiveCount(appspaceID domain.AppspaceID) {
	a.liveCounterMux.Lock()
	defer a.liveCounterMux.Unlock()
	if _, ok := a.liveCounter[appspaceID]; !ok {
		a.liveCounter[appspaceID] = 0
	}
	a.liveCounter[appspaceID]++
	go a.emitLiveCount(appspaceID, a.liveCounter[appspaceID])
}
func (a *AppspaceRouter) decrementLiveCount(appspaceID domain.AppspaceID) {
	a.liveCounterMux.Lock()
	defer a.liveCounterMux.Unlock()
	if _, ok := a.liveCounter[appspaceID]; ok {
		a.liveCounter[appspaceID]--
		go a.emitLiveCount(appspaceID, a.liveCounter[appspaceID])
		if a.liveCounter[appspaceID] == 0 {
			delete(a.liveCounter, appspaceID)
		}
	}
}

// SubscribeLiveCount pushes the number of live requests for an appspace each time it changes
// It returns the current count
func (a *AppspaceRouter) SubscribeLiveCount(appspaceID domain.AppspaceID, ch chan<- int) int {
	a.UnsubscribeLiveCount(appspaceID, ch)
	a.subscribersMux.Lock()
	defer a.subscribersMux.Unlock()
	subscribers, ok := a.subscribers[appspaceID]
	if !ok {
		a.subscribers[appspaceID] = append([]chan<- int{}, ch)
	} else {
		a.subscribers[appspaceID] = append(subscribers, ch)
	}

	a.liveCounterMux.Lock()
	defer a.liveCounterMux.Unlock()
	count, ok := a.liveCounter[appspaceID]
	if !ok {
		return 0
	}
	return count
}

// UnsubscribeLiveCount unsubscribes
func (a *AppspaceRouter) UnsubscribeLiveCount(appspaceID domain.AppspaceID, ch chan<- int) {
	a.subscribersMux.Lock()
	defer a.subscribersMux.Unlock()
	subscribers, ok := a.subscribers[appspaceID]
	if !ok {
		return
	}
	for i, c := range subscribers {
		if c == ch {
			close(ch)
			subscribers[i] = subscribers[len(subscribers)-1]
			a.subscribers[appspaceID] = subscribers[:len(subscribers)-1]
			return
		}
	}
}

func (a *AppspaceRouter) emitLiveCount(appspaceID domain.AppspaceID, count int) {
	a.subscribersMux.Lock()
	defer a.subscribersMux.Unlock()
	subscribers, ok := a.subscribers[appspaceID]
	if !ok {
		return
	}
	for _, ch := range subscribers {
		ch <- count
	}
}

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
			if err == domain.ErrNoRowsInResultSet {
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

func (a *AppspaceRouter) getLogger(appspaceID domain.AppspaceID) *record.DsLogger {
	return record.NewDsLogger().AppspaceID(appspaceID).AddNote("AppspaceRouter")
}
