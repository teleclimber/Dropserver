package appspacerouter

import (
	"context"
	"net/http"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/getcleanhost"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// Hmm, this is kind of a misnomer now?
// This handles all requests that are not pointed to the ds-host user domain.
// So dropids and appspaces, and maybe even other things?
// Since dropid and apppsace domains can overlap, OK to handle them together

// Let's try some new context-driven stuff
type ctxKey string

const urlTailCtxKey = ctxKey("url tail")

func getURLTail(ctx context.Context) string {
	t, ok := ctx.Value(urlTailCtxKey).(string)
	if !ok {
		return ""
	}
	return t
}

// AppspaceRouter handles routes for appspaces.
type AppspaceRouter struct {
	Authenticator interface {
		Authenticate(*http.Request) domain.Authentication //TODO tempoaray!!
	}
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
	}
	AppspaceModel interface {
		GetFromDomain(string) (*domain.Appspace, error)
	}
	AppspaceStatus interface {
		Ready(domain.AppspaceID) bool
	}
	DropserverRoutes   http.Handler
	RouteModelsManager domain.AppspaceRouteModels
	V0AppspaceRouter   domain.RouteHandler

	liveCounterMux sync.Mutex
	liveCounter    map[domain.AppspaceID]int

	subscribersMux sync.Mutex
	subscribers    map[domain.AppspaceID][]chan<- int
}

// Init initializes data structures
func (r *AppspaceRouter) Init() {
	r.liveCounter = make(map[domain.AppspaceID]int)
	r.subscribers = make(map[domain.AppspaceID][]chan<- int)
}

// ^^ Also need access to sessions

// ServeHTTP handles http traffic to the appspace
func (r *AppspaceRouter) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	host, err := getcleanhost.GetCleanHost(req.Host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	head, tail := shiftpath.ShiftPath(req.URL.Path)
	if head == ".dropserver" {
		ctx := context.WithValue(req.Context(), urlTailCtxKey, tail)
		r.DropserverRoutes.ServeHTTP(res, req.WithContext(ctx))
		return
	}

	appspace, err := r.AppspaceModel.GetFromDomain(host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if appspace == nil {
		http.Error(res, "Appspace does not exist: "+host, http.StatusNotFound)
		return
	}

	if !r.AppspaceStatus.Ready(appspace.AppspaceID) {
		http.Error(res, "Appspace unavailable", http.StatusServiceUnavailable)
		return
	}

	r.incrementLiveCount(appspace.AppspaceID)
	defer r.decrementLiveCount(appspace.AppspaceID)

	auth := r.Authenticator.Authenticate(req)

	routeData := &domain.AppspaceRouteData{ //curently using AppspaceRouteData for user routes as well
		URLTail:        req.URL.Path,
		Authentication: &auth}

	routeData.Appspace = appspace

	app, err := r.AppModel.GetFromID(appspace.AppID)
	if err != nil { // do we differentiate between empty result vs other errors? -> No, if any kind of DB error occurs, the DB or model will log it.
		r.getLogger(appspace).Log("Error: App does not exist") // this is an actua system error: an appspace is missing its app.
		http.Error(res, err.Error(), 500)
		return
	}
	routeData.App = app

	appVersion, err := r.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
	if err != nil {
		r.getLogger(appspace).Log("Error: AppVersion does not exist")
		http.Error(res, "App Version not found", http.StatusInternalServerError)
		return
	}
	routeData.AppVersion = appVersion

	// This is where we branch off into different API versions for serving appspace traffic
	r.V0AppspaceRouter.ServeHTTP(res, req, routeData)
}

func (r *AppspaceRouter) incrementLiveCount(appspaceID domain.AppspaceID) {
	r.liveCounterMux.Lock()
	defer r.liveCounterMux.Unlock()
	if _, ok := r.liveCounter[appspaceID]; !ok {
		r.liveCounter[appspaceID] = 0
	}
	r.liveCounter[appspaceID]++
	go r.emitLiveCount(appspaceID, r.liveCounter[appspaceID])
}
func (r *AppspaceRouter) decrementLiveCount(appspaceID domain.AppspaceID) {
	r.liveCounterMux.Lock()
	defer r.liveCounterMux.Unlock()
	if _, ok := r.liveCounter[appspaceID]; ok {
		r.liveCounter[appspaceID]--
		go r.emitLiveCount(appspaceID, r.liveCounter[appspaceID])
		if r.liveCounter[appspaceID] == 0 {
			delete(r.liveCounter, appspaceID)
		}
	}
}

// SubscribeLiveCount pushes the number of live requests for an appspace each time it changes
// It returns the current count
func (r *AppspaceRouter) SubscribeLiveCount(appspaceID domain.AppspaceID, ch chan<- int) int {
	r.UnsubscribeLiveCount(appspaceID, ch)
	r.subscribersMux.Lock()
	defer r.subscribersMux.Unlock()
	subscribers, ok := r.subscribers[appspaceID]
	if !ok {
		r.subscribers[appspaceID] = append([]chan<- int{}, ch)
	} else {
		r.subscribers[appspaceID] = append(subscribers, ch)
	}

	r.liveCounterMux.Lock()
	defer r.liveCounterMux.Unlock()
	count, ok := r.liveCounter[appspaceID]
	if !ok {
		return 0
	}
	return count
}

// UnsubscribeLiveCount unsubscribes
func (r *AppspaceRouter) UnsubscribeLiveCount(appspaceID domain.AppspaceID, ch chan<- int) {
	r.subscribersMux.Lock()
	defer r.subscribersMux.Unlock()
	subscribers, ok := r.subscribers[appspaceID]
	if !ok {
		return
	}
	for i, c := range subscribers {
		if c == ch {
			subscribers[i] = subscribers[len(subscribers)-1]
			r.subscribers[appspaceID] = subscribers[:len(subscribers)-1]
		}
	}
}

func (r *AppspaceRouter) emitLiveCount(appspaceID domain.AppspaceID, count int) {
	r.subscribersMux.Lock()
	defer r.subscribersMux.Unlock()
	subscribers, ok := r.subscribers[appspaceID]
	if !ok {
		return
	}
	for _, ch := range subscribers {
		go func(c chan<- int) {
			c <- count
		}(ch)
	}
}

// Or maybe this is an events thing?
// Or maybe just a singular event when no more requests?
//  -> no, make generic not specific to some other package's needs.
// Consider that future features might be ability to view live requests in owner frontend, etc...

func (r *AppspaceRouter) getLogger(appspace *domain.Appspace) *record.DsLogger {
	return record.NewDsLogger().AppID(appspace.AppID).AppVersion(appspace.AppVersion).AppspaceID(appspace.AppspaceID).AddNote("AppspaceRouter")
}
