package migrateappspace

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"

	"github.com/teleclimber/DropServer/internal/nulltypes"
)

// Twine command values:
var migrateCommand = 11

// JobController handles appspace functionality
type JobController struct {
	MigrationJobModel domain.MigrationJobModel
	AppModel          interface {
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, domain.Error)
	}
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, domain.Error)
		SetVersion(domain.AppspaceID, domain.Version) domain.Error
	}
	AppspaceInfoModels interface {
		Get(domain.AppspaceID) domain.AppspaceInfoModel
	}
	AppspaceStatus interface {
		Ready(appspaceID domain.AppspaceID) bool
		WaitStopped(appspaceID domain.AppspaceID) // probably will have to be WaitStopped
	}
	SandboxManager domain.SandboxManagerI // regular appspace sandboxes
	SandboxMaker   SandboxMakerI

	runningJobs map[domain.JobID]*runningJob
	runningMux  sync.Mutex
	ticker      *time.Ticker
	stop        bool
	allStopped  chan struct{}

	fanIn     chan runningJobStatus
	ownerSubs map[domain.UserID]map[string]chan<- domain.MigrationStatusData

	// TODO need subscribe mux, no?
	subscribers []chan<- domain.MigrationStatusData
}

// Start allows jobs to run and can start the first job
// with a delay (in the future)
func (c *JobController) Start() { // maybe pass delay before start (we want c.stop = true to take effect right away)
	c.getLogger("Start()").Debug("starting")

	c.runningJobs = make(map[domain.JobID]*runningJob)
	c.fanIn = make(chan runningJobStatus)
	c.ownerSubs = make(map[domain.UserID]map[string]chan<- domain.MigrationStatusData)

	c.stop = false
	c.allStopped = make(chan struct{})

	go c.eventManifold()

	c.ticker = time.NewTicker(time.Minute)

	go func() {
		for {
			select {
			case <-c.allStopped:
				return
			case <-c.ticker.C:
				c.startNext()
			}
		}
	}()

	c.startNext()
}

// Stop blocks further jobs from starting
// and waits for the last job to finish before returning.
func (c *JobController) Stop() {
	c.stop = true

	c.ticker.Stop()

	c.runningMux.Lock()
	if len(c.runningJobs) == 0 {
		close(c.allStopped)
	}
	c.runningMux.Unlock()

	<-c.allStopped

	close(c.fanIn)
}

// Subscribe to running migration job events
func (c *JobController) Subscribe(ch chan<- domain.MigrationStatusData) {
	c.removeSubscriber(ch)
	c.subscribers = append(c.subscribers, ch)
}

// Unsubscribe to running migration job events
func (c *JobController) Unsubscribe(ch chan<- domain.MigrationStatusData) {
	c.removeSubscriber(ch)
}

func (c *JobController) removeSubscriber(ch chan<- domain.MigrationStatusData) {
	// get a feeling you'll need a mutex to cover subscribers?
	for i, s := range c.subscribers {
		if s == ch {
			c.subscribers[i] = c.subscribers[len(c.subscribers)-1]
			c.subscribers = c.subscribers[:len(c.subscribers)-1]
		}
	}
}

// SubscribeOwner returns current jobs for owner
// and pipes all updates to the provided channel.
// Is this currently running jobs? OR pending jobs too?
// -> I think it has to be running jobs, otherwise this is just a proxy to jobs model
func (c *JobController) SubscribeOwner(ownerID domain.UserID, subID string) (<-chan domain.MigrationStatusData, []domain.MigrationStatusData) {
	// lock running jobs so we can guarantee the updates are relative to the current jobs we return.
	c.runningMux.Lock()
	defer c.runningMux.Unlock()

	updateChan := make(chan domain.MigrationStatusData)
	if c.ownerSubs[ownerID] == nil {
		c.ownerSubs[ownerID] = make(map[string]chan<- domain.MigrationStatusData)
	}
	c.ownerSubs[ownerID][subID] = updateChan

	curStatus := []domain.MigrationStatusData{}
	for _, rj := range c.runningJobs {
		if rj.migrationJob.OwnerID == ownerID {
			curStatus = append(curStatus, makeMigrationStatusData(rj.getCurStatusData()))
		}
	}
	return updateChan, curStatus
}

// UnsubscribeOwner deletes owner/cookie from subscribers.
// It closes the channel too
func (c *JobController) UnsubscribeOwner(ownerID domain.UserID, subID string) {
	c.runningMux.Lock()
	defer c.runningMux.Unlock()
	subs, ok := c.ownerSubs[ownerID]
	if !ok {
		return
	}
	updateChan, ok := subs[subID]
	if !ok {
		return
	}
	close(updateChan)
	delete(c.ownerSubs[ownerID], subID)
}

// GetRunningJobs returns an array of currently started and unfinished jobs
func (c *JobController) GetRunningJobs() (rj []domain.MigrationStatusData) {
	c.runningMux.Lock()
	defer c.runningMux.Unlock()
	for _, job := range c.runningJobs {
		rj = append(rj, makeMigrationStatusData(job.getCurStatusData()))
	}
	return
}

var statString = map[domain.MigrationJobStatus]string{ // TODO: this is duplicated, particularly with json response
	domain.MigrationStarted:  "started",
	domain.MigrationRunning:  "running",
	domain.MigrationFinished: "finished"}

// eventManifold receives fanIn events and processes them accordingly.
// It shuts down when c.fanIn is closed
func (c *JobController) eventManifold() { // eventBus?
	for d := range c.fanIn {
		if d.errString.Valid {
			// TODO: put migration job id, appspace id, ...
			c.getLogger("eventManifold").Log("Run migration job: finished with error: " + d.errString.String)
			// ^^ This is more likely be app-level logging.
		} else {
			//c.Logger.Log(domain.INFO, nil, "Run migration job "+statString[d.status]+": "+strconv.Itoa(int(d.origJob.JobID))+" ")
			// ^^ successful migrations need not be logged to dev log.
		}

		// Clean up:
		if d.status == domain.MigrationFinished {
			dsErr := c.MigrationJobModel.SetFinished(d.origJob.JobID, d.errString)
			if dsErr != nil {
				//c.Logger.Log(domain.ERROR, nil, "Run migration job: failed to set finished: "+dsErr.PublicString())
				// ^^ this is already logged by model. No need to log.
				// But should probably warn user that something is not right.
			}

			c.runningMux.Lock()
			delete(c.runningJobs, d.origJob.JobID)

			if c.stop && len(c.runningJobs) == 0 && c.allStopped != nil {
				close(c.allStopped)
			}
			c.runningMux.Unlock()

			go c.startNext()
		}

		// Subscribers
		migrationStatusData := makeMigrationStatusData(d)

		subs, ok := c.ownerSubs[d.origJob.OwnerID]
		if ok {
			for _, updateChan := range subs {
				updateChan <- migrationStatusData
				// worried this may block or panic if we don't have all our ducks in a row
			}
		}

		for _, ch := range c.subscribers {
			ch <- migrationStatusData
		}
	}
}

func makeMigrationStatusData(s runningJobStatus) domain.MigrationStatusData {
	return domain.MigrationStatusData{
		JobID:      s.origJob.JobID,
		AppspaceID: s.origJob.AppspaceID,
		Status:     s.status,
		Started:    s.origJob.Started,
		Finished:   s.origJob.Finished,
		ErrString:  s.errString,
		CurSchema:  s.curSchema,
	}
}

// WakeUp tells the job controller to immediately look for a job to process
// Call this after inserting a new job with high priority to start that job right away
// (if possible, depending on load and other jobs in the queue)
func (c *JobController) WakeUp() {
	c.startNext()
}

// startNext decides if it should start a job
// if so it finds the next job and starts it
func (c *JobController) startNext() {
	if c.stop {
		return
	}

	c.runningMux.Lock()
	defer c.runningMux.Unlock()

	if len(c.runningJobs) > 0 {
		return
	}

	jobs, dsErr := c.MigrationJobModel.GetPending()
	if dsErr != nil {
		//c.Logger.Log(domain.ERROR, nil, "Error getting pending jobs: "+dsErr.PublicString())
		// ^^ already logged by model.
		return
	}

	var runJob *domain.MigrationJob
	for _, j := range jobs {
		ok, _ := c.MigrationJobModel.SetStarted(j.JobID)
		// if dsErr != nil {
		// 	c.Logger.Log(domain.ERROR, nil, "Error setting job to started: "+dsErr.PublicString())
		// already logged by model
		// }
		if ok {
			// check if a job is already running on that appspace
			// TODO: wouldn't you check that before calling setStarted??
			appspaceJobExists := false
			for _, rj := range c.runningJobs {
				if rj.migrationJob.AppspaceID == j.AppspaceID {
					appspaceJobExists = true
					break
				}
			}
			if !appspaceJobExists {
				runJob = j
				break
			}
		}
	}

	if runJob != nil {
		c.getLogger("startNext()").Debug("Found job to run")
		rj := c.createRunningJob(runJob)
		rj.subscribeStatus(c.fanIn)
		rj.setStatus(domain.MigrationStarted)
		c.runningJobs[runJob.JobID] = rj
		go c.runJob(rj)
	}
}

// wonder if we need a job pre-run?
// - check if sandbox is needed (so we can manage resources) //later
// - tell appspace to gracefully shutdown, and wait til it does to actually start job (blocking appspace)

func (c *JobController) createRunningJob(job *domain.MigrationJob) *runningJob {
	return &runningJob{
		migrationJob: job,
		sandbox:      c.SandboxMaker.Make(),
		statusSubs:   make([]chan<- runningJobStatus, 0)}
}
func (c *JobController) runJob(job *runningJob) {
	defer job.setStatus(domain.MigrationFinished)

	appspaceID := job.migrationJob.AppspaceID

	c.AppspaceStatus.WaitStopped(appspaceID)

	c.SandboxManager.StopAppspace(appspaceID) // Even if there is no code to run for the migration the sandbox has to be restarted with new app version.

	var dsErr domain.Error

	job.appspace, dsErr = c.AppspaceModel.GetFromID(appspaceID)
	if dsErr != nil {
		job.errStr.SetString("Error getting appspace: " + dsErr.PublicString())
		return
		// job encountered error condition
		// if no rows, just means appspace has been deleted in he interim
		// otherwise, bigger problem.
	}

	infoModel := c.AppspaceInfoModels.Get(appspaceID)
	fromSchema, err := infoModel.GetSchema()
	if err != nil {
		job.errStr.SetString("Error getting current schema: " + err.Error())
		return
	}

	toVersion, dsErr := c.AppModel.GetVersion(job.appspace.AppID, job.migrationJob.ToVersion)
	if dsErr != nil {
		job.errStr.SetString("Error getting toVersion: " + dsErr.PublicString())
		return
		// if no rows, that means version was deleted
		// Job should have been deleted too. That's a program error
		// otherwise it's also an error
	}

	if toVersion.Schema == fromSchema {
		// any cleanup?
		// still need to stop the appspace sandbox and restart it with the correct app code
		// Also probably need to clear caches... of various... things?
		return
	}

	job.useVersion = toVersion
	job.fromSchema = job.curSchema
	job.toSchema = toVersion.Schema

	if job.toSchema < job.fromSchema {
		job.migrateDown = true
		job.useVersion, dsErr = c.AppModel.GetVersion(job.appspace.AppID, job.appspace.AppVersion)
		if dsErr != nil {
			job.errStr.SetString("Error getting fromVersion: " + dsErr.PublicString())
			return
			// if no rows, that means version was deleted even though appspaces were using it. That's a program error
		}
	}

	err = job.runMigration()
	if err != nil {
		// log to log for user
		// do rollbacks or whatever
		job.errStr.SetString("Error running Migration: " + err.Error())
		return
	}

	err = infoModel.SetSchema(job.toSchema)
	if err != nil {
		job.errStr.SetString("Error setting schema after Migration: " + err.Error())
		return
	}

	dsErr = c.AppspaceModel.SetVersion(appspaceID, toVersion.Version)
	if dsErr != nil {
		job.errStr.SetString("Error setting new Version: " + dsErr.PublicString())
		return
	}
}

func (c *JobController) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("JobController")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

// It seems a key thing is making it possible for sandbox proxy to query started jobs
// ..so as to avoid starting a sandbox while this is happening.
// Also there needs to be a place to store pending migrations so they can be started one by one.
// Can we just have a DB table 'migrations': *appspace_id*(PK), version, added_dt, started_dt, ended_dt
// -> might also need a priority system for jobs where user is waiting at the other end

// Wonder if we should combine a number of appspace tasks, like:
// - migrate
// - install
// - backup
// - export data?
// - ...?
// .. maybe but wondering what the effect will be?
// Does it mean you can't run a backup task while there is a pending migration task?

// What to do about situation where we are waiting for both a sandbox and a job to start?
// Does anything take priority? Or is it that the first one to start will block the other until done?

// TODO: split job into parts based on the dropserver API version of the app version
// .. you'll have to stop the sandbox and migrate appspace meta db, then start new sandbox with new version libs

// Wondering if runningJob should have its own getLogger?
// Might cut down on message passing?

type runningJob struct {
	migrationJob *domain.MigrationJob
	appspace     *domain.Appspace
	useVersion   *domain.AppVersion
	fromSchema   int
	toSchema     int
	curSchema    int // not sure about this one
	migrateDown  bool
	sandbox      domain.SandboxI
	status       domain.MigrationJobStatus
	errStr       nulltypes.NullString
	statusSubs   []chan<- runningJobStatus
	//curStatusData domain.MigrationStatusData
	statusMux sync.Mutex
}

type runningJobStatus struct {
	origJob   *domain.MigrationJob
	status    domain.MigrationJobStatus
	errString nulltypes.NullString
	curSchema int
}

func (r *runningJob) runMigration() error {
	if r.fromSchema == r.toSchema {
		return nil
	}

	r.setStatus(domain.MigrationRunning)

	r.getLogger("runMigration()").Debug("about to start migration")

	defer r.sandbox.Stop()
	err := r.sandbox.Start(r.useVersion, r.appspace)
	if err != nil {
		// host level error? log it
		r.getLogger("runMigration, sandbox.Start()").Error(err)
		return err
	}
	r.sandbox.WaitFor(domain.SandboxReady)
	// sandbox may not be ready if it failed to start.
	// check status?
	// Maybe WaitFor could return an error if the status is changed to something that can never become Ready.

	// Here we should have a migration service
	// Send message to migrate from a -> b
	// .. which might be incremental  version-by-version?

	p := struct {
		FromSchema int `json:"from"`
		ToSchema   int `json:"to"`
	}{FromSchema: r.fromSchema,
		ToSchema: r.toSchema}

	payload, err := json.Marshal(p)
	if err != nil {
		// this is an ds host error
		r.getLogger("runMigration, jsonMarshal payload").Error(err)
		return err // "internal error"
	}

	sent, err := r.sandbox.SendMessage(sandbox.MigrateService, migrateCommand, &payload)
	if err != nil {
		r.getLogger("runMigration, sandbox.SendMessage").Error(err)
		return err
	}

	// we could get regular updates, like the current version number, etc...
	// But for now just wait for the response
	reply, err := sent.WaitReply()
	if err != nil {
		// This one probaly means the sandbox crashed or some such
		r.getLogger("runMigration, sandbox.SendMessage").Error(err)
		return err
	}

	if !reply.OK() {
		return reply.Error() // this is the only one that is not an internal error
	}

	return nil
}

func (r *runningJob) subscribeStatus(sub chan<- runningJobStatus) {
	r.statusMux.Lock()
	defer r.statusMux.Unlock()

	r.statusSubs = append(r.statusSubs, sub)
}

func (r *runningJob) setStatus(status domain.MigrationJobStatus) {
	r.statusMux.Lock()
	defer r.statusMux.Unlock()

	r.status = status
	switch status {
	case domain.MigrationStarted:
		r.migrationJob.Started = nulltypes.NewTime(time.Now(), true)
	case domain.MigrationFinished:
		r.migrationJob.Finished = nulltypes.NewTime(time.Now(), true)
	}

	statusData := r.getCurStatusData()

	for _, sub := range r.statusSubs {
		sub <- statusData
	}
}

func (r *runningJob) getCurStatusData() runningJobStatus {
	// do we need to lock ?
	return runningJobStatus{
		origJob:   r.migrationJob,
		status:    r.status,
		errString: r.errStr,
		curSchema: r.curSchema} // how does this get updated?
}

func (r *runningJob) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("runningJob")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
