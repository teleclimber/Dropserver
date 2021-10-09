package appspacerouter

import (
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/getcleanhost"
)

// Hmm, this is kind of a misnomer now?
// This handles all requests that are not pointed to the ds-host user domain.
// So dropids and appspaces, and maybe even other things?
// Since dropid and apppsace domains can overlap, OK to handle them together

// AppspaceRouter handles routes for appspaces.
type AppspaceRouter struct {
	Authenticator interface {
		AppspaceUserProxyID(http.Handler) http.Handler
	} `checkinject:"required"`
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
	} `checkinject:"required"`
	AppspaceModel interface {
		GetFromDomain(string) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceStatus interface {
		Ready(domain.AppspaceID) bool
	} `checkinject:"required"`
	DropserverRoutes interface {
		Router() http.Handler
	} `checkinject:"required"`
	V0AppspaceRouter http.Handler `checkinject:"required"`

	liveCounterMux sync.Mutex
	liveCounter    map[domain.AppspaceID]int

	subscribersMux sync.Mutex
	subscribers    map[domain.AppspaceID][]chan<- int

	mux *chi.Mux
}

// Init initializes data structures
func (a *AppspaceRouter) Init() {
	a.liveCounter = make(map[domain.AppspaceID]int)
	a.subscribers = make(map[domain.AppspaceID][]chan<- int)

	mux := chi.NewRouter()
	mux.Use(a.loadAppspace)
	mux.Use(a.Authenticator.AppspaceUserProxyID)
	mux.Use(a.errorPage)
	mux.Use(a.appspaceAvailable, a.countRequest)
	mux.Use(a.loadApp)
	// Not sure we need all these middlewares for all routes.
	// - dropserver routes like get login token do not need appspace user, and may not care if available or to count request?
	//   -> actually may need available + count because there may be side-effects to appspace, like recording last used or whatever
	// - also does not need app and ap version

	// first match dropserver routes
	mux.Mount("/.dropserver", a.DropserverRoutes.Router())
	mux.Handle("/*", http.HandlerFunc(a.branchToVersionedRouters))

	a.mux = mux
}
func (a *AppspaceRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func (a *AppspaceRouter) loadAppspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: use of r.Host not good enough. see the requestHost function of https://github.com/go-chi/hostrouter
		// May need to determine host at server and stash it in context.
		host, err := getcleanhost.GetCleanHost(r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		appspace, err := a.AppspaceModel.GetFromDomain(host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if appspace == nil {
			w.WriteHeader(http.StatusNotFound)
			a.notFoundPage(w)
			return
		}

		r = r.WithContext(domain.CtxWithAppspaceData(r.Context(), *appspace))

		next.ServeHTTP(w, r)
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
			a.getLogger(appspace).AddNote("AppModel.GetFromID").Error(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		ctx := domain.CtxWithAppData(r.Context(), *app)

		appVersion, err := a.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
		if err != nil {
			a.getLogger(appspace).AddNote("AppModel.GetVersion").Error(err)
			http.Error(w, "App Version not found", http.StatusInternalServerError)
			return
		}
		ctx = domain.CtxWithAppVersionData(ctx, *appVersion)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *AppspaceRouter) branchToVersionedRouters(w http.ResponseWriter, r *http.Request) {
	// Here eventually we will branch off to different versions of appspace routers.
	a.V0AppspaceRouter.ServeHTTP(w, r)
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

func (a *AppspaceRouter) notFoundPage(w http.ResponseWriter) {
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
			subscribers[i] = subscribers[len(subscribers)-1]
			a.subscribers[appspaceID] = subscribers[:len(subscribers)-1]
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

// Or maybe this is an events thing?
// Or maybe just a singular event when no more requests?
//  -> no, make generic not specific to some other package's needs.
// Consider that future features might be ability to view live requests in owner frontend, etc...

func (a *AppspaceRouter) getLogger(appspace domain.Appspace) *record.DsLogger {
	return record.NewDsLogger().AppID(appspace.AppID).AppVersion(appspace.AppVersion).AppspaceID(appspace.AppspaceID).AddNote("AppspaceRouter")
}
