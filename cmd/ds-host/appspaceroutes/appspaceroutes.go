package appspaceroutes

import (
	"net/http"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// route handler for when we know the route is for an app-space.
// Could be proxied to sandbox, or static file, or crud or whatever

// AppspaceRoutes handles routes for appspaces.
type AppspaceRoutes struct {
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, domain.Error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, domain.Error)
	}
	AppspaceModel interface {
		GetFromSubdomain(string) (*domain.Appspace, domain.Error)
	}
	AppspaceStatus interface {
		Ready(domain.AppspaceID) bool
	}
	RouteModelsManager domain.AppspaceRouteModels
	V0                 domain.RouteHandler

	liveCounterMux sync.Mutex
	liveCounter    map[domain.AppspaceID]int

	subscribersMux sync.Mutex
	subscribers    map[domain.AppspaceID][]chan<- int
}

// Init initializes data structures
func (r *AppspaceRoutes) Init() {
	r.liveCounter = make(map[domain.AppspaceID]int)
	r.subscribers = make(map[domain.AppspaceID][]chan<- int)
}

// ^^ Also need access to sessions

// ServeHTTP handles http traffic to the appspace
func (r *AppspaceRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	subdomains := *routeData.Subdomains
	appspaceSubdomain := subdomains[len(subdomains)-1]

	appspace, dsErr := r.AppspaceModel.GetFromSubdomain(appspaceSubdomain)
	if dsErr != nil && dsErr.Code() == dserror.NoRowsInResultSet {
		http.Error(res, "Appspace does not exist", http.StatusNotFound)
		return
	} else if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	if !r.AppspaceStatus.Ready(appspace.AppspaceID) {
		http.Error(res, "Appspace unavailable", http.StatusServiceUnavailable)
		return
	}

	r.incrementLiveCount(appspace.AppspaceID)
	defer r.decrementLiveCount(appspace.AppspaceID)

	routeData.Appspace = appspace

	app, dsErr := r.AppModel.GetFromID(appspace.AppID)
	if dsErr != nil { // do we differentiate between empty result vs other errors? -> No, if any kind of DB error occurs, the DB or model will log it.
		r.getLogger(appspace).Log("Error: App does not exist") // this is an actua system error: an appspace is missing its app.
		dsErr.HTTPError(res)
		return
	}
	routeData.App = app

	appVersion, dsErr := r.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
	if dsErr != nil {
		r.getLogger(appspace).Log("Error: AppVersion does not exist")
		http.Error(res, "App Version not found", http.StatusInternalServerError)
		return
	}
	routeData.AppVersion = appVersion

	// This is where we branch off into different API versions for serving appspace traffic
	r.V0.ServeHTTP(res, req, routeData)
}

func (r *AppspaceRoutes) incrementLiveCount(appspaceID domain.AppspaceID) {
	r.liveCounterMux.Lock()
	defer r.liveCounterMux.Unlock()
	if _, ok := r.liveCounter[appspaceID]; !ok {
		r.liveCounter[appspaceID] = 0
	}
	r.liveCounter[appspaceID]++
	go r.emitLiveCount(appspaceID, r.liveCounter[appspaceID])
}
func (r *AppspaceRoutes) decrementLiveCount(appspaceID domain.AppspaceID) {
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
func (r *AppspaceRoutes) SubscribeLiveCount(appspaceID domain.AppspaceID, ch chan<- int) int {
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
func (r *AppspaceRoutes) UnsubscribeLiveCount(appspaceID domain.AppspaceID, ch chan<- int) {
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

func (r *AppspaceRoutes) emitLiveCount(appspaceID domain.AppspaceID, count int) {
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

func (r *AppspaceRoutes) getLogger(appspace *domain.Appspace) *record.DsLogger {
	return record.NewDsLogger().AppID(appspace.AppID).AppVersion(appspace.AppVersion).AppspaceID(appspace.AppspaceID).AddNote("AppspaceRoutes")
}
