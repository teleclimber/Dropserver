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

// I'd like to add an ejected status too. Signals:
// - it's paused (maybe? maybe not? Want to eject to make backups without writing in DB tath it's paused.)
// - all requests have finished
// - any cron function call has finished running
// - sandbox is terminated
// - not migrating (and migration system needs to check to see if we're trying to eject appspace)
// - appspace meta db file closed
// - all appspace db files closed

// We should deepend the status object,
// and give it a more fine-grained protection against race conditions
// -> a top level statusMux for getting the status object a
// .. and a per-appspace mux to fiddle with the status

type status struct {
	data statusData
	lock sync.Mutex
}

type statusData struct {
	paused           bool // paused status is set in appspace DB
	tempPaused       bool // set via a function call to make it so the system knows we're trying to eject
	dataSchema       int  // from appspace metadata
	appVersionSchema int  // from app files
	migrating        bool // when a migration job starts
	problem          bool // Something went wrong, appsapce can't be used
}

// live jobs and requests should be booleans.
// Otherwise the "status" will fire constantly?
// Also posible we need a sceond "level" to status
// ..because even if bool, each request could switch status from false to true and back.
// Yeah very questionable that we want liveRequests and live Jobs in status
// Just look at the other fields: they only change raraley, hhile live* can change many times per second.

// I think the rigth way to do this is:
// - remove live* from status data
// - have a WaitStopped function that subscribes to jobs and requests
//   ..and can tell anybody who aslked for WaitStopped when it goes to zero.
//   ..then it unsubscribes to the live* events.

// possible flags to add:
// - sandbox status
// - metadb file open
// - appspace db files open ?
// - file pointers from appspace files??? We can't, but it's the logical step after, and is a clue we should not go crazy here

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

	AppspaceFilesEvents interface {
		Subscribe(chan<- domain.AppspaceID)
	}
	AppspaceRouter interface {
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
	status    map[domain.AppspaceID]*status
}

// Init creates data structures and subscribes to events
func (s *AppspaceStatus) Init() {
	s.status = make(map[domain.AppspaceID]*status)

	asPausedCh := make(chan domain.AppspacePausedEvent)
	go s.handleAppspacePause(asPausedCh)
	s.AppspacePausedEvent.Subscribe(asPausedCh)

	asFilesCh := make(chan domain.AppspaceID)
	go s.handleAppspaceFiles(asFilesCh)
	s.AppspaceFilesEvents.Subscribe(asFilesCh)

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
	status.lock.Lock()
	defer status.lock.Unlock()

	if status.data.problem || status.data.paused || status.data.tempPaused || status.data.migrating {
		return false
	}
	if status.data.appVersionSchema != status.data.dataSchema {
		return false
	}

	return true
}

// SetTempPause sets the tempPaused flag on appspace status
func (s *AppspaceStatus) SetTempPause(appspaceID domain.AppspaceID, paused bool) {
	status := s.getStatus(appspaceID)

	status.lock.Lock()
	defer status.lock.Unlock()
	if status.data.tempPaused != paused {
		status.data.tempPaused = paused
		s.sendChangedEvent(appspaceID, status.data)
	}
}

func (s *AppspaceStatus) getStatus(appspaceID domain.AppspaceID) *status {
	s.statusMux.Lock()
	defer s.statusMux.Unlock()

	stat, ok := s.status[appspaceID]
	if !ok {
		stat = &status{
			data: s.getData(appspaceID)}
		s.status[appspaceID] = stat
	}
	return stat
}

func (s *AppspaceStatus) getTrackedStatus(appspaceID domain.AppspaceID) *status {
	s.statusMux.Lock()
	defer s.statusMux.Unlock()

	status, ok := s.status[appspaceID]
	if ok {
		return status
	}
	return nil
}

func (s *AppspaceStatus) getData(appspaceID domain.AppspaceID) statusData {
	data := statusData{}

	appspace, dsErr := s.AppspaceModel.GetFromID(appspaceID)
	if dsErr != nil {
		data.problem = true
		return data
	}
	data.paused = appspace.Paused

	jobs := s.MigrationJobs.GetRunningJobs()
	for _, job := range jobs {
		if job.AppspaceID == appspaceID && !job.Finished.Valid {
			data.migrating = true
		}
	}

	// load appVersionSchema. Note that it should not change over time, so no need to subscribe.
	appVersion, dsErr := s.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
	if dsErr != nil {
		data.problem = true
		return data
	}
	data.appVersionSchema = appVersion.Schema

	// load data schema
	// Head's up: there is a chance that the meta db isn't created yet
	// But that's not an error, and we should just get a 0
	// (note info model returns zero if no schema set, but returns error if no db present (I think?))
	// Note that you don't need to subscribe, since change should only be possible via migration.
	schema, err := s.AppspaceInfoModels.GetSchema(appspaceID)
	if err != nil {
		data.problem = true
	}
	data.dataSchema = schema

	return data
}

func (s *AppspaceStatus) handleAppspacePause(ch <-chan domain.AppspacePausedEvent) {
	for p := range ch {
		status := s.getTrackedStatus(p.AppspaceID)
		if status != nil {
			status.lock.Lock()
			if status.data.paused != p.Paused {
				status.data.paused = p.Paused
				s.sendChangedEvent(p.AppspaceID, status.data)
			}
			status.lock.Unlock()
		}
	}
}

func (s *AppspaceStatus) handleAppspaceFiles(ch <-chan domain.AppspaceID) {
	for appspaceID := range ch {
		status := s.getTrackedStatus(appspaceID)
		if status != nil {
			s.updateStatus(appspaceID, status)
		}
	}
}

func (s *AppspaceStatus) handleMigrationJobUpdate(ch <-chan domain.MigrationStatusData) {
	for d := range ch {
		status := s.getTrackedStatus(d.AppspaceID)
		if status != nil {
			if d.Finished.Valid {
				s.updateStatus(d.AppspaceID, status) //reload whole status because migration might have changed many things
			} else {
				status.lock.Lock()
				if !status.data.migrating {
					status.data.migrating = true
					s.sendChangedEvent(d.AppspaceID, status.data)
				}
				status.lock.Unlock()
			}
		}
	}
}

// Since this is used in ds-dev only, we'll cheat a bit
func (s *AppspaceStatus) handleAppVersionEvent(ch <-chan domain.AppID) {
	for range ch {
		s.statusMux.Lock()
		for appspaceID, status := range s.status {
			s.updateStatus(appspaceID, status) // reload everything
		}
		s.statusMux.Unlock()
	}
}

func (s *AppspaceStatus) updateStatus(appspaceID domain.AppspaceID, curStatus *status) {
	curStatus.lock.Lock()
	defer curStatus.lock.Unlock()
	cur := curStatus.data

	new := s.getData(appspaceID)
	changed := false

	if cur.paused != new.paused {
		curStatus.data.paused = new.paused
		changed = true
	}
	// skip tempPaused because it's not determined by getData. It can only change via SetTempPaused fn above.
	if cur.appVersionSchema != new.appVersionSchema {
		curStatus.data.appVersionSchema = new.appVersionSchema
		changed = true
	}
	if cur.dataSchema != new.dataSchema {
		curStatus.data.dataSchema = new.dataSchema
		changed = true
	}
	if cur.migrating != new.migrating {
		curStatus.data.migrating = new.migrating
		changed = true
	}
	if cur.problem != new.problem {
		curStatus.data.problem = new.problem
		changed = true
	}

	if changed {
		s.sendChangedEvent(appspaceID, curStatus.data)
	}
}

func (s *AppspaceStatus) sendChangedEvent(appspaceID domain.AppspaceID, status statusData) {
	go s.AppspaceStatusEvents.Send(appspaceID, domain.AppspaceStatusEvent{
		AppspaceID:       appspaceID,
		Paused:           status.paused,
		TempPaused:       status.tempPaused,
		AppVersionSchema: status.appVersionSchema,
		AppspaceSchema:   status.dataSchema,
		Migrating:        status.migrating,
		Problem:          status.problem})
}

// WaitStopped returns when an appspace has stopped
// Meant to be used in conjunction with an appspace blocking function (pause, tempPause, migrate, etc...)
// If you want more details, there are route hit events too.
func (s *AppspaceStatus) WaitStopped(appspaceID domain.AppspaceID) {

	// check with cron jobs too when we have them

	ch := make(chan int)
	count := s.AppspaceRouter.SubscribeLiveCount(appspaceID, ch)
	if count == 0 {
		return
	}
	for count = range ch {
		if count == 0 {
			s.AppspaceRouter.UnsubscribeLiveCount(appspaceID, ch)
			close(ch)
		}
	}
}
