package appspacestatus

import (
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// takes signals from various places
// and determines if an appspace can be used at this moment.
//
// Signals:
// - appspace paused
// - [appspace stopped] (for deletion)// for now deletion begins with pausing
// - host going down
// - app version schema == appspace data schema
// - appspace metadata "mid-migration" flag
// - migration job active

// Note that this simply responds to the question: "can this request/cron/whatever proceed"?
// It does not acutally do any action. It does not shut down sandboxes or anything.
// It just collects the data necessary to respond to the question:
// - is the sandbox usable right now?
// - is the sandbox completely down? (so a migration can run, data can be copied, etc...)

// It caches the status so it can respond quickly to numerous queries
// And it also allows subscriptions for realtime queries

type appspaceStatus struct {
	// add a mux to protect all these values on a per-appspace basis?
	paused           bool
	migrating        bool
	dataSchema       int
	appVersionSchema int
	problem          bool
}

// AppspaceStatus determines the status of an appspace
type AppspaceStatus struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, domain.Error)
	}
	AppspacePausedEvent interface {
		Subscribe(chan<- domain.AppspacePausedEvent)
	}
	AppModel interface {
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, domain.Error)
	}

	AppspaceInfoModels interface {
		GetSchema(domain.AppspaceID) (int, error)
	}

	AppspaceRoutes interface {
		SubscribeLiveCount(domain.AppspaceID, chan<- int) int
		UnsubscribeLiveCount(domain.AppspaceID, chan<- int)
	}
	MigrationJobs interface {
		GetRunningJobs() []domain.MigrationStatusData
	}
	MigrationJobsEvents interface {
		Subscribe(chan<- domain.MigrationStatusData)
	}

	//AppVersionEvent for when an app version can change its schema/whatever live
	// This is only relevant in ds-dev and can be left nil in prod.
	AppVersionEvents interface {
		Subscribe(chan<- domain.AppID)
	}

	AppspaceStatusEvents interface {
		Send(domain.AppspaceID, domain.AppspaceStatusEvent)
	}

	hostStopMux sync.Mutex
	hostStop    bool

	statusMux sync.Mutex
	status    map[domain.AppspaceID]appspaceStatus
}

// Init creates data structures and subscribes to events
func (s *AppspaceStatus) Init() {
	s.status = make(map[domain.AppspaceID]appspaceStatus)

	asPausedCh := make(chan domain.AppspacePausedEvent)
	go s.handleAppspacePause(asPausedCh)
	s.AppspacePausedEvent.Subscribe(asPausedCh)

	migrationJobsCh := make(chan domain.MigrationStatusData)
	go s.handleMigrationJobUpdate(migrationJobsCh)
	s.MigrationJobsEvents.Subscribe(migrationJobsCh)

	if s.AppVersionEvents != nil {
		appVersionCh := make(chan domain.AppID)
		go s.handleAppVersionEvent(appVersionCh)
		s.AppVersionEvents.Subscribe(appVersionCh)
	}
}

// SetHostStop sets the hostStop flag.
// When set to true Ready() returns false for all appspaces.
func (s *AppspaceStatus) SetHostStop(stop bool) {
	s.hostStopMux.Lock()
	defer s.hostStopMux.Unlock()

	s.hostStop = stop
}
func (s *AppspaceStatus) getHostStop() bool {
	s.hostStopMux.Lock()
	defer s.hostStopMux.Unlock()

	return s.hostStop
}

// Ready returns true if there is nothing that prevents the appspace from being used right now.
func (s *AppspaceStatus) Ready(appspaceID domain.AppspaceID) bool {
	if s.getHostStop() {
		return false
	}

	status := s.getStatus(appspaceID)

	// TODO lock status mux
	if status.problem || status.paused || status.migrating {
		return false
	}
	if status.appVersionSchema != status.dataSchema {
		return false
	}

	return true
}

// Wonder if there should be a ReadyLock() func
// ..that sets a lock on ready state if appspace is ready
// ..so that the caller can be sure that if it is ready, it remains ready
// ..until it has a chance to perform some op that causes it to be non-stopped?
// The problem: appspace routes gets a request, checks Ready(), gets true
// At the same time a migration job starts and checks "Stopped()", gets true because nothing is "active"
// (the job starting cause Ready() to return false)
// The request proceeds but fails because job has shut down sandbox.
// -> perhaps the solution is for appspaceRoutes to register the route
//    ..then ask if it can proceed.

func (s *AppspaceStatus) getStatus(appspaceID domain.AppspaceID) appspaceStatus {
	s.statusMux.Lock()
	defer s.statusMux.Unlock()

	status, ok := s.status[appspaceID]
	if !ok {
		status = s.loadStatus(appspaceID)
		s.status[appspaceID] = status
	}
	return status
}

func (s *AppspaceStatus) loadStatus(appspaceID domain.AppspaceID) (status appspaceStatus) {
	appspace, dsErr := s.AppspaceModel.GetFromID(appspaceID)
	if dsErr != nil {
		status.problem = true
		return
	}
	status.paused = appspace.Paused

	jobs := s.MigrationJobs.GetRunningJobs()
	for _, job := range jobs {
		if job.AppspaceID == appspaceID && !job.Finished.Valid {
			status.migrating = true
		}
	}

	// load appVersionSchema. Note that it should not change over time, so no need to subscribe.
	appVersion, dsErr := s.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
	if dsErr != nil {
		status.problem = true
		return
	}
	status.appVersionSchema = appVersion.Schema

	// load data schema
	// Head's up: there is a chance that the meta db isn't created yet
	// But that's not an error, and we should just get a 0
	// (note info model returns zero if no schema set, but returns error if no db present (I think?))
	// Note that you don't need to subscribe, since change should only be possible via migration.
	schema, err := s.AppspaceInfoModels.GetSchema(appspaceID)
	if err != nil {
		status.problem = true
	}
	status.dataSchema = schema

	return
}

// What would need to happen to make appspace status fully event-driven?
// ..meaning that it could emit a ready/ stopped/ whatever event that accounts for all possible changes?
// - We have pause event from the appspace model
// - Migration events should tell you whether it is currently migrating (what about if it's about to migrate? I think that's covered)
// - App version schema is a constant, unless you are in ds-dev where it can change at any time
// - appspace schema can only change on conclusion of migration, so migration events cover that.

// So we mostly just need to rig something up for app version changes.
// .. which will only be used by ds-dev

// Also need to become and event emitter

func (s *AppspaceStatus) handleAppspacePause(ch <-chan domain.AppspacePausedEvent) {
	for p := range ch {
		s.statusMux.Lock()

		status, ok := s.status[p.AppspaceID]
		if ok {
			status.paused = p.Paused
			s.updateStatus(p.AppspaceID, status)
		}
		s.statusMux.Unlock()
	}
}

func (s *AppspaceStatus) handleMigrationJobUpdate(ch <-chan domain.MigrationStatusData) {
	for d := range ch {
		s.statusMux.Lock()
		status, ok := s.status[d.AppspaceID]
		if ok {
			if d.Finished.Valid {
				s.updateStatus(d.AppspaceID, s.loadStatus(d.AppspaceID)) // reload everything because migration finished and might have changed a bunch of things.
			} else {
				status.migrating = true
				s.updateStatus(d.AppspaceID, status)
			}
		}
		s.statusMux.Unlock()
	}
}

// Since this is used in ds-dev only, we'll cheat a bit
func (s *AppspaceStatus) handleAppVersionEvent(ch <-chan domain.AppID) {
	for range ch {
		s.statusMux.Lock()
		for appspaceID := range s.status {
			s.updateStatus(appspaceID, s.loadStatus(appspaceID))
		}
		s.statusMux.Unlock()
	}
}

func (s *AppspaceStatus) updateStatus(appspaceID domain.AppspaceID, newStatus appspaceStatus) {
	status, ok := s.status[appspaceID]
	if ok && statusChanged(status, newStatus) {
		s.status[appspaceID] = newStatus
		go s.AppspaceStatusEvents.Send(appspaceID, domain.AppspaceStatusEvent{
			AppspaceID:       appspaceID,
			AppVersionSchema: newStatus.appVersionSchema,
			AppspaceSchema:   newStatus.dataSchema,
			Migrating:        newStatus.migrating,
			Paused:           newStatus.paused,
			Problem:          newStatus.problem})
	}
}

func statusChanged(old, new appspaceStatus) bool {
	if old.paused != new.paused {
		return true
	}
	if old.migrating != new.migrating {
		return true
	}
	if old.appVersionSchema != new.appVersionSchema {
		return true
	}
	if old.dataSchema != new.dataSchema {
		return true
	}
	if old.problem != new.problem {
		return true
	}

	return false
}

// Oh dear, I can see it now: we will want to subscribe to any change in Ready() and Stopped()
// .. to keep the frontend up to date for ex.
// Which mean appspace status will also be its own event emitter?
// -> yes probably.

// not sure how to handle the stopping / ejecting transients.
// A constant subscription to live route count doesn't seem right (potential for many events)
// Could maybe have ZeroActivity event, that only fires from route when there are no more routes.
// The idea would be to subscribe when you are expecting a shutdown, and just wait for the signal
// The appspace routes handler would send 0 immediately on subscription if current count is zero.
// (the count shouldn't go up if route handler knows that the appspace is closing down).

// If yo do want more details, there are route hit events too.

// WaitStopped returns when an appspace has stopped
func (s *AppspaceStatus) WaitStopped(appspaceID domain.AppspaceID) {

	// check with cron

	ch := make(chan int)
	count := s.AppspaceRoutes.SubscribeLiveCount(appspaceID, ch)
	if count == 0 {
		return
	}
	for count = range ch {
		if count == 0 {
			s.AppspaceRoutes.UnsubscribeLiveCount(appspaceID, ch)
			close(ch)
		}
	}
}
