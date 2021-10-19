package appspaceops

import (
	"errors"
	"sync"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
	"github.com/teleclimber/DropServer/internal/nulltypes"
	"github.com/teleclimber/twine-go/twine/mock_twine"
)

func TestRunningJobStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	controller := &MigrationJobController{}

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
	sandboxManager := testmocks.NewMockSandboxManager(mockCtrl)

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

	infoModel := testmocks.NewMockAppspaceInfoModel(mockCtrl)
	infoModel.EXPECT().GetSchema(appspaceID).Return(1, nil)
	infoModel.EXPECT().SetSchema(appspaceID, 2).Return(nil)

	replyMessage := mock_twine.NewMockReceivedReplyI(mockCtrl)
	replyMessage.EXPECT().OK().Return(true)

	sentMessage := mock_twine.NewMockSentMessageI(mockCtrl)
	sentMessage.EXPECT().WaitReply().Return(replyMessage, nil)

	sandbox := domain.NewMockSandboxI(mockCtrl)
	sandbox.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(sentMessage, nil)
	sandbox.EXPECT().Graceful()

	sandboxMaker := testmocks.NewMockSandboxMaker(mockCtrl)
	sandboxMaker.EXPECT().ForMigration(gomock.Any(), gomock.Any()).Return(sandbox, nil)

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().WaitTempPaused(appspaceID, "migrating").Return(make(chan struct{}))

	backupAppspace := testmocks.NewMockBackupAppspace(mockCtrl)
	backupAppspace.EXPECT().BackupNoPause(appspaceID).Return("some-zip-file.zip", nil)

	c := &MigrationJobController{
		AppspaceModel:     appspaceModel,
		AppModel:          appModel,
		AppspaceInfoModel: infoModel,
		BackupAppspace:    backupAppspace,
		SandboxManager:    sandboxManager,
		SandboxMaker:      sandboxMaker,
		AppspaceStatus:    appspaceStatus,
	}

	rj := c.createRunningJob(job)

	c.runJob(rj)

	if rj.errStr.Valid {
		t.Error(rj.errStr.Value())
	}
}

func TestStartNextStopped(t *testing.T) {
	c := &MigrationJobController{
		stop: true}

	c.startNext()
}
func TestStartNextNoJobs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := testmocks.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{}, nil)

	c := &MigrationJobController{
		MigrationJobModel: migrationJobModel,
	}

	c.startNext()
}

func TestStartNextOneJob(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	j := &domain.MigrationJob{
		JobID:      1,
		AppspaceID: appspaceID}

	migrationJobModel := testmocks.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{j}, nil)
	migrationJobModel.EXPECT().SetStarted(j.JobID).Return(true, nil)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(nil, errors.New("nada"))

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().WaitTempPaused(appspaceID, "migrating").Return(make(chan struct{}))

	sandboxManager := testmocks.NewMockSandboxManager(mockCtrl)
	sandboxManager.EXPECT().StopAppspace(appspaceID).Return()

	c := &MigrationJobController{
		MigrationJobModel: migrationJobModel,
		AppspaceModel:     appspaceModel,
		AppspaceStatus:    appspaceStatus,
		SandboxManager:    sandboxManager,
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

	c := &MigrationJobController{
		runningJobs: make(map[domain.JobID]*runningJob),
		fanIn:       make(chan runningJobStatus, 10)}

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

	migrationJobModel := testmocks.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().SetFinished(domain.JobID(1), gomock.Any())

	c := &MigrationJobController{
		MigrationJobModel: migrationJobModel,
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

	migrationJobModel := testmocks.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{}, nil)

	c := &MigrationJobController{
		MigrationJobModel: migrationJobModel}

	c.Start()
	c.Stop()
}

func TestFullStartStopWithJob(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := testmocks.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{}, nil)
	migrationJobModel.EXPECT().SetFinished(domain.JobID(1), gomock.Any())

	appspaceID := domain.AppspaceID(7)

	c := &MigrationJobController{
		MigrationJobModel: migrationJobModel}

	rj := c.createRunningJob(&domain.MigrationJob{
		JobID:      1,
		AppspaceID: appspaceID,
	})

	c.Start()

	// I don't get what this test is doing?
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
