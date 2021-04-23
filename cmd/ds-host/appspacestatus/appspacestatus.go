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
// - is the appspace usable right now?
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

type tempPause struct {
	startCh chan struct{}
	reason  string
}

type statusData struct {
	paused           bool        // paused status is set in appspace DB
	tempPauses       []tempPause // pauses for appspace operations like migrations, backups, etc...
	dataSchema       int         // from appspace metadata
	appVersionSchema int         // from app files
	problem          bool        // Something went wrong, appsapce can't be used
}

// possible flags to add:
// - sandbox status
// - metadb file open
// - appspace db files open ?
// - file pointers from appspace files??? We can't, but it's the logical step after, and is a clue we should not go crazy here

// AppspaceStatus determines the status of an appspace
type AppspaceStatus struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	}
	AppspacePausedEvent interface {
		Subscribe(chan<- domain.AppspacePausedEvent)
	}
	AppModel interface {
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
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
	MigrationJobEvents interface {
		Subscribe(chan<- domain.MigrationJob)
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

	closedMux sync.Mutex
	closed    map[domain.AppspaceID]bool
}

// Init creates data structures and subscribes to events
func (s *AppspaceStatus) Init() {
	s.status = make(map[domain.AppspaceID]*status)
	s.closed = make(map[domain.AppspaceID]bool)

	asPausedCh := make(chan domain.AppspacePausedEvent)
	go s.handleAppspacePause(asPausedCh)
	s.AppspacePausedEvent.Subscribe(asPausedCh)

	asFilesCh := make(chan domain.AppspaceID)
	go s.handleAppspaceFiles(asFilesCh)
	s.AppspaceFilesEvents.Subscribe(asFilesCh)

	migrationJobsCh := make(chan domain.MigrationJob)
	go s.handleMigrationJobUpdate(migrationJobsCh)
	s.MigrationJobEvents.Subscribe(migrationJobsCh)

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

	if status.data.problem || status.data.paused || len(status.data.tempPauses) != 0 {
		return false
	}
	if status.data.appVersionSchema != status.data.dataSchema {
		return false
	}

	return true
}

// ^^ Wonder if we should have a WaitReady(ctx context.Context) bool
// That would wait for ready state (until context says no more)

// WaitTempPaused pauses the appspace and returns when appspace activity is stopped
// This function returns for only one caller at a time.
// It returns for the next caller when the returned channel is closed
func (s *AppspaceStatus) WaitTempPaused(appspaceID domain.AppspaceID, reason string) chan struct{} {
	startCh := s.getTempPause(appspaceID, reason)
	<-startCh

	finishCh := make(chan struct{})

	go func() {
		<-finishCh
		s.finishTempPause(appspaceID)
	}()

	s.WaitStopped(appspaceID)

	return finishCh
}

// IsTempPaused returns true if a temp pause is in effect
// It does not consider whether the appspace has effectively stopped
func (s *AppspaceStatus) IsTempPaused(appspaceID domain.AppspaceID) bool {
	status := s.getStatus(appspaceID)
	status.lock.Lock()
	defer status.lock.Unlock()
	return len(status.data.tempPauses) != 0
}

func (s *AppspaceStatus) getTempPause(appspaceID domain.AppspaceID, reason string) chan struct{} {
	status := s.getStatus(appspaceID)

	status.lock.Lock()
	defer status.lock.Unlock()
	tp := tempPause{
		startCh: make(chan struct{}),
		reason:  reason}
	status.data.tempPauses = append(status.data.tempPauses, tp)
	if len(status.data.tempPauses) == 1 {
		s.sendChangedEvent(appspaceID, status.data)
		close(tp.startCh)
	}
	return tp.startCh
}

// finishTempPause closes the 0th temp pause and starts the next one
// or sends a status change notification if there are none.
func (s *AppspaceStatus) finishTempPause(appspaceID domain.AppspaceID) {
	status := s.getStatus(appspaceID)

	status.lock.Lock()
	defer status.lock.Unlock()

	status.data.tempPauses = status.data.tempPauses[1:]
	if len(status.data.tempPauses) == 0 {
		s.sendChangedEvent(appspaceID, status.data)
		return
	}

	close(status.data.tempPauses[0].startCh) // start the next one
}

// We could have a flag for deletion / archive.
// ..that is set explicitly and that behaves like an explicit Pause.
// This could be set in DB and could have values 0, 1, 2 => (active, archive, delete)
// With this, migration would check status and ensure it's "active". Don't migrate an archived or deleted appspace.
// Then we can just use regular pause, with maybe a "waitPaused"

// Track causes appspace id to monitored and future events will be sent
// It returns an event struct that represents the current state
func (s *AppspaceStatus) Track(appspaceID domain.AppspaceID) domain.AppspaceStatusEvent {
	stat := s.getStatus(appspaceID)
	return getEvent(appspaceID, stat.data)
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
	data := statusData{
		tempPauses: make([]tempPause, 0, 3),
	}

	appspace, err := s.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		data.problem = true
		return data
	}
	data.paused = appspace.Paused

	// load appVersionSchema. Note that it should not change over time, so no need to subscribe.
	appVersion, err := s.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
	if err != nil {
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

func (s *AppspaceStatus) handleMigrationJobUpdate(ch <-chan domain.MigrationJob) {
	for d := range ch {
		status := s.getTrackedStatus(d.AppspaceID)
		if status != nil && d.Finished.Valid {
			s.updateStatus(d.AppspaceID, status) //reload whole status because migration might have changed many things
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
	// skip tempPauses because it's not determined by getData. It can only change via fn above.
	if cur.appVersionSchema != new.appVersionSchema {
		curStatus.data.appVersionSchema = new.appVersionSchema
		changed = true
	}
	if cur.dataSchema != new.dataSchema {
		curStatus.data.dataSchema = new.dataSchema
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
	go s.AppspaceStatusEvents.Send(appspaceID, getEvent(appspaceID, status))
}

func getEvent(appspaceID domain.AppspaceID, status statusData) domain.AppspaceStatusEvent {
	pReason := ""
	if len(status.tempPauses) != 0 {
		pReason = status.tempPauses[0].reason
	}
	return domain.AppspaceStatusEvent{
		AppspaceID:       appspaceID,
		Paused:           status.paused, // maybe add archived, deleted. Or put everything under an "active"
		TempPaused:       len(status.tempPauses) != 0,
		TempPauseReason:  pReason,
		AppVersionSchema: status.appVersionSchema,
		AppspaceSchema:   status.dataSchema,
		Problem:          status.problem}
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

// In addition to "stopped", I think we may need "ejected"
// Where ejected means that all files are closed, in particular
// - appspace meta db
// - other appspace dbs.
// For regular appspace data files, we can assume that if there are no ongoing requests
// ..no sandboxes, no cron jobs, then everything ought to be closed?

// What we realy need is to lock the ejected state
// meaning it's like tempPaused, but also nothing can open any of its files.
// so appspace metadb should check with appspaceStatus before opening?

// LockClosed sets the locked flag for the appspace until the return channel is closed.
func (s *AppspaceStatus) LockClosed(appspaceID domain.AppspaceID) (chan struct{}, bool) {
	s.closedMux.Lock()
	defer s.closedMux.Unlock()
	locked, ok := s.closed[appspaceID]
	if ok && locked {
		return nil, false
	}
	s.closed[appspaceID] = true
	ch := make(chan struct{})
	go s.unlockClosed(appspaceID, ch)
	return ch, true
}

func (s *AppspaceStatus) unlockClosed(appspaceID domain.AppspaceID, ch chan struct{}) {
	<-ch
	s.closedMux.Lock()
	defer s.closedMux.Unlock()
	s.closed[appspaceID] = false
}

// IsLockedClosed returns true if the appspace is locked
// and no files should be opened.
func (s *AppspaceStatus) IsLockedClosed(appspaceID domain.AppspaceID) bool {
	s.closedMux.Lock()
	defer s.closedMux.Unlock()
	locked, ok := s.closed[appspaceID]
	return ok && locked
}
