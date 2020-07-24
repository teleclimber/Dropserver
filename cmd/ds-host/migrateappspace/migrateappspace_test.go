package migrateappspace

import (
	"sync"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/nulltypes"
	"github.com/teleclimber/DropServer/internal/twine"
)

func TestGetRunningJobs(t *testing.T) {
	c := &JobController{
		runningJobs: make(map[domain.JobID]*runningJob)}

	appspaceID := domain.AppspaceID(7)

	c.runningJobs[domain.JobID(5)] = &runningJob{
		migrationJob: &domain.MigrationJob{
			AppspaceID: appspaceID,
		},
	}

	jobs := c.GetRunningJobs()
	if len(jobs) != 1 {
		t.Error("expected one job")
	}
	if jobs[0].AppspaceID != appspaceID {
		t.Error("bad appspace ID")
	}
}

func TestSubscribe(t *testing.T) {
	c := &JobController{}
	ch := make(chan<- domain.MigrationStatusData)

	c.Subscribe(ch)
	if len(c.subscribers) != 1 {
		t.Error("expected 1 subscriber")
	}

	c.Unsubscribe(ch)
	if len(c.subscribers) != 0 {
		t.Error("expected no subscribers")
	}
}

func TestSubscribeEvent(t *testing.T) {
	c := &JobController{
		fanIn: make(chan runningJobStatus)}
	go c.eventManifold()
	ch := make(chan domain.MigrationStatusData)
	c.Subscribe(ch)

	appspaceID := domain.AppspaceID(7)

	go func() {
		c.fanIn <- runningJobStatus{
			origJob: &domain.MigrationJob{
				AppspaceID: appspaceID,
			},
		}
		close(c.fanIn)
	}()

	status := <-ch
	if status.AppspaceID != appspaceID {
		t.Error("expected correct appspace id")
	}

	c.Unsubscribe(ch)
}

func TestSubscribeOwner(t *testing.T) {
	c := &JobController{}

	c.runningJobs = make(map[domain.JobID]*runningJob)
	c.ownerSubs = make(map[domain.UserID]map[string]chan<- domain.MigrationStatusData)

	ownerID := domain.UserID(7)
	sessionID := "abc"
	c.SubscribeOwner(ownerID, sessionID)
	if len(c.ownerSubs[ownerID]) != 1 {
		t.Error("expected one subscriber in there")
	}
	c.UnsubscribeOwner(ownerID, sessionID)
	if len(c.ownerSubs[ownerID]) != 0 {
		t.Error("expected no subscribers in there")
	}
}

func TestSubscribeOwnerDouble(t *testing.T) {
	c := &JobController{}

	c.runningJobs = make(map[domain.JobID]*runningJob)
	c.ownerSubs = make(map[domain.UserID]map[string]chan<- domain.MigrationStatusData)

	ownerID := domain.UserID(7)
	sessionID1 := "abc"
	sessionID2 := "def"
	c.SubscribeOwner(ownerID, sessionID1)
	if len(c.ownerSubs[ownerID]) != 1 {
		t.Error("expected one subscriber in there")
	}
	c.SubscribeOwner(ownerID, sessionID2)
	if len(c.ownerSubs[ownerID]) != 2 {
		t.Error("expected two subscribers in there")
	}
	c.UnsubscribeOwner(ownerID, sessionID1)
	if len(c.ownerSubs[ownerID]) != 1 {
		t.Error("expected 1 subscribers in there")
	}
}

func TestSubscribeOwnerWithRunningJob(t *testing.T) {
	c := &JobController{}

	c.runningJobs = make(map[domain.JobID]*runningJob)
	c.ownerSubs = make(map[domain.UserID]map[string]chan<- domain.MigrationStatusData)

	ownerID := domain.UserID(7)
	sessionID := "abc"
	jobID := domain.JobID(11)
	appspaceID := domain.AppspaceID(22)
	c.runningJobs[jobID] = &runningJob{
		migrationJob: &domain.MigrationJob{
			JobID:      jobID,
			AppspaceID: appspaceID,
			OwnerID:    ownerID,
		},
		status: domain.MigrationRunning,
	}

	_, stats := c.SubscribeOwner(ownerID, sessionID)
	if len(c.ownerSubs[ownerID]) != 1 {
		t.Error("expected one subscriber in there")
	}
	if len(stats) != 1 {
		t.Error("expected one current status")
	}
	if stats[0].Status != domain.MigrationRunning {
		t.Error("expected status of job to be migration running")
	}
	c.UnsubscribeOwner(ownerID, sessionID)
	if len(c.ownerSubs[ownerID]) != 0 {
		t.Error("expected no subscribers in there")
	}
}

func TestRunningJobStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sandboxMaker := NewMockSandboxMakerI(mockCtrl)
	sandboxMaker.EXPECT().Make()

	controller := &JobController{
		SandboxMaker: sandboxMaker}
	job := &domain.MigrationJob{}
	rj := controller.createRunningJob(job)

	fanIn := make(chan runningJobStatus)
	rj.subscribeStatus(fanIn)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		statusData := <-fanIn
		if statusData.status != domain.MigrationStarted {
			t.Error("expected status to be started")
		}
		wg.Done()
	}()

	rj.setStatus(domain.MigrationStarted)

	wg.Wait()
}

func TestRunJob(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	sandboxManager := domain.NewMockSandboxManagerI(mockCtrl)

	appID := domain.AppID(7)
	appspaceID := domain.AppspaceID(11)
	fromVersion := domain.Version("0.0.1")
	toVersion := domain.Version("0.0.2")

	job := &domain.MigrationJob{
		JobID:      1,
		AppspaceID: appspaceID,
		ToVersion:  toVersion,
	}

	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{
		AppspaceID:  appspaceID,
		AppID:       appID,
		AppVersion:  fromVersion,
		LocationKey: "appspace-location",
	}, nil)
	appModel.EXPECT().GetVersion(appID, toVersion).Return(&domain.AppVersion{
		AppID:       appID,
		Version:     toVersion,
		Schema:      2,
		LocationKey: "to-location",
	}, nil)

	sandboxManager.EXPECT().StopAppspace(appspaceID).Return()

	appspaceModel.EXPECT().SetVersion(appspaceID, toVersion).Return(nil)

	infoModel := domain.NewMockAppspaceInfoModel(mockCtrl)
	infoModel.EXPECT().GetSchema().Return(1, nil)
	infoModel.EXPECT().SetSchema(2).Return(nil)
	infoModels := testmocks.NewMockAppspaceInfoModels(mockCtrl)
	infoModels.EXPECT().Get(appspaceID).Return(infoModel)

	replyMessage := twine.NewMockReceivedReplyI(mockCtrl)
	replyMessage.EXPECT().OK().Return(true)

	sentMessage := twine.NewMockSentMessageI(mockCtrl)
	sentMessage.EXPECT().WaitReply().Return(replyMessage, nil)

	sandbox := domain.NewMockSandboxI(mockCtrl)
	sandbox.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil)
	sandbox.EXPECT().WaitFor(gomock.Any())
	sandbox.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(sentMessage, nil)
	sandbox.EXPECT().Stop()

	sandboxMaker := NewMockSandboxMakerI(mockCtrl)
	sandboxMaker.EXPECT().Make().Return(sandbox)

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().WaitStopped(appspaceID)

	c := &JobController{
		AppspaceModel:      appspaceModel,
		AppModel:           appModel,
		AppspaceInfoModels: infoModels,
		SandboxManager:     sandboxManager,
		SandboxMaker:       sandboxMaker,
		AppspaceStatus:     appspaceStatus,
	}

	rj := c.createRunningJob(job)

	c.runJob(rj)

	if rj.errStr.Valid {
		t.Error(rj.errStr.Value())
	}
}

func TestStartNextStopped(t *testing.T) {
	c := &JobController{
		stop: true}

	c.startNext()
}
func TestStartNextNoJobs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{}, nil)

	c := &JobController{
		MigrationJobModel: migrationJobModel,
	}

	c.startNext()

	//
	// appspaceModel := domain.NewMockAppspaceModel(mockCtrl)
	// appspaceModel.EXPECT().GetFromID(appspaceID).Return(nil, dserror.New(dserror.NoRowsInResultSet))
}

func TestStartNextOneJob(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	j := &domain.MigrationJob{
		JobID:      1,
		AppspaceID: appspaceID}

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{j}, nil)
	migrationJobModel.EXPECT().SetStarted(j.JobID).Return(true, nil)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(nil, dserror.New(dserror.NoRowsInResultSet))

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().WaitStopped(appspaceID)

	sandboxMaker := NewMockSandboxMakerI(mockCtrl)
	sandboxMaker.EXPECT().Make()

	sandboxManager := domain.NewMockSandboxManagerI(mockCtrl)
	sandboxManager.EXPECT().StopAppspace(appspaceID).Return()

	c := &JobController{
		MigrationJobModel: migrationJobModel,
		AppspaceModel:     appspaceModel,
		AppspaceStatus:    appspaceStatus,
		SandboxManager:    sandboxManager,
		SandboxMaker:      sandboxMaker,
		runningJobs:       make(map[domain.JobID]*runningJob),
		fanIn:             make(chan runningJobStatus, 10),
	}

	c.startNext()

	rj := c.runningJobs[j.JobID]

	sub := make(chan runningJobStatus)
	rj.subscribeStatus(sub)

	var s runningJobStatus
	for s = range sub {
		if s.status == domain.MigrationFinished {
			break
		}
	}

	if !s.errString.Valid {
		t.Error("expected an error because we returned and error in get appspace")
	}
}

// ^^ Should test when setStarted returns not OK
// and when appspace job is already running.

func TestEventManifold(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c := &JobController{
		runningJobs: make(map[domain.JobID]*runningJob),
		fanIn:       make(chan runningJobStatus, 10),
		ownerSubs:   make(map[domain.UserID]map[string]chan<- domain.MigrationStatusData)}

	j := &domain.MigrationJob{
		JobID:   11,
		OwnerID: domain.UserID(7)}

	c.runningJobs[j.JobID] = &runningJob{
		migrationJob: j}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.eventManifold()
		wg.Done()
	}()

	c.fanIn <- runningJobStatus{
		origJob:   j,
		errString: nulltypes.NewString("boo!", true)}

	close(c.fanIn)

	wg.Wait()
}

func TestEventManifoldFinished(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().SetFinished(domain.JobID(1), gomock.Any())

	sandboxMaker := NewMockSandboxMakerI(mockCtrl)
	sandboxMaker.EXPECT().Make()

	c := &JobController{
		MigrationJobModel: migrationJobModel,
		SandboxMaker:      sandboxMaker,
		runningJobs:       make(map[domain.JobID]*runningJob),
		fanIn:             make(chan runningJobStatus, 10),
		stop:              true, // prevents startNext from running again
	}

	appspaceID := domain.AppspaceID(7)

	rj := c.createRunningJob(&domain.MigrationJob{
		JobID:      1,
		AppspaceID: appspaceID,
	})

	c.runningJobs[rj.migrationJob.JobID] = rj

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.eventManifold()
		wg.Done()
	}()

	c.fanIn <- runningJobStatus{
		origJob:   rj.migrationJob,
		status:    domain.MigrationFinished,
		errString: nulltypes.NewString("boo!", true),
	}

	close(c.fanIn)

	wg.Wait()
}

func TestFullStartStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{}, nil)

	c := &JobController{
		MigrationJobModel: migrationJobModel}

	c.Start()
	c.Stop()
}

func TestFullStartStopWithJob(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{}, nil)
	migrationJobModel.EXPECT().SetFinished(domain.JobID(1), gomock.Any())

	appspaceID := domain.AppspaceID(7)

	sandboxMaker := NewMockSandboxMakerI(mockCtrl)
	sandboxMaker.EXPECT().Make()

	c := &JobController{
		MigrationJobModel: migrationJobModel,
		SandboxMaker:      sandboxMaker}

	rj := c.createRunningJob(&domain.MigrationJob{
		JobID:      1,
		AppspaceID: appspaceID,
	})

	c.Start()

	c.runningJobs[rj.migrationJob.JobID] = rj

	go func() {
		c.fanIn <- runningJobStatus{
			origJob:   rj.migrationJob,
			status:    domain.MigrationFinished,
			errString: nulltypes.NewString("", false),
		}
	}()

	c.Stop()
}
