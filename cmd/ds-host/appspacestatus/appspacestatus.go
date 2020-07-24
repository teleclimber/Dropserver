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
// - is the sandbox completely down? (so a migration can run, etc...)

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
	// TODO lock status mux

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

func (s *AppspaceStatus) handleAppspacePause(ch <-chan domain.AppspacePausedEvent) {
	for p := range ch {
		s.statusMux.Lock()

		status, ok := s.status[p.AppspaceID]
		if ok {
			status.paused = p.Paused
			s.status[p.AppspaceID] = status
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
				s.status[d.AppspaceID] = s.loadStatus(d.AppspaceID) // reload everything because migration finished and might have changed a bunch of things.
			} else {
				status.migrating = true
				s.status[d.AppspaceID] = status
			}
		}
		s.statusMux.Unlock()
	}
}

// Oh dear, I can see it now: we will want to subscribe to any change in Ready() and Stopped()
// .. to keep the frontend up to date for ex.
// Which mean appspace status will also be its own event emitter?
// -> yes probably.

// WaitStopped returns when an appspace has stopped
func (s *AppspaceStatus) WaitStopped(appspaceID domain.AppspaceID) {
	// if s.Ready(appspaceID) {
	// 	return //wait that's not right... dead wrong in fact
	// }

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
