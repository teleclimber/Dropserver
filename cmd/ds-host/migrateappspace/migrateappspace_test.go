package migrateappspace

import (
	"sync"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

func TestRunningJobStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	controller := &JobController{}
	job := &domain.MigrationJob{}
	rj := controller.createRunningJob(job)

	fanIn := make(chan domain.MigrationStatusData)
	rj.subscribeStatus(fanIn)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		statusData := <-fanIn
		if statusData.Status != domain.MigrationStarted {
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

	appModel := domain.NewMockAppModel(mockCtrl)
	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)
	sandboxManager := domain.NewMockSandboxManagerI(mockCtrl)
	migrationSandboxMgr := NewMockMigrationSandobxMgrI(mockCtrl)
	migrationSandbox := NewMockMigrationSandboxI(mockCtrl)

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
	appModel.EXPECT().GetVersion(appID, fromVersion).Return(&domain.AppVersion{
		AppID:   appID,
		Version: fromVersion,
		Schema:  1,
	}, nil)
	appModel.EXPECT().GetVersion(appID, toVersion).Return(&domain.AppVersion{
		AppID:       appID,
		Version:     toVersion,
		Schema:      2,
		LocationKey: "to-location",
	}, nil)

	sandboxManager.EXPECT().StopAppspace(appspaceID).Return()

	// migrationsandbox if schemas are different
	migrationSandbox.EXPECT().Start("to-location", "appspace-location", 1, 2)
	migrationSandboxMgr.EXPECT().CreateSandbox().Return(migrationSandbox)

	appspaceModel.EXPECT().SetVersion(appspaceID, toVersion).Return(nil)

	c := &JobController{
		AppspaceModel:       appspaceModel,
		AppModel:            appModel,
		SandboxManager:      sandboxManager,
		MigrationSandboxMgr: migrationSandboxMgr,
	}

	rj := c.createRunningJob(job)

	c.runJob(rj)
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
		AppspaceID: appspaceID,
	}

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{j}, nil)
	migrationJobModel.EXPECT().SetStarted(1).Return(true, nil)

	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(nil, dserror.New(dserror.NoRowsInResultSet))

	c := &JobController{
		MigrationJobModel: migrationJobModel,
		AppspaceModel:     appspaceModel,
		runningJobs:       make(map[domain.AppspaceID]*runningJob),
		fanIn:             make(chan domain.MigrationStatusData, 10),
	}

	c.startNext()

	rj := c.runningJobs[appspaceID]

	sub := make(chan domain.MigrationStatusData)
	rj.subscribeStatus(sub)

	var s domain.MigrationStatusData
	for s = range sub {
		if s.Status == domain.MigrationFinished {
			break
		}
	}

	if !s.ErrString.Valid {
		t.Error("expected an error because we returned and error in get appspace")
	}
}

// ^^ Should test when setStarted returns not OK
// and when appspace job is already running.

func TestEventManifold(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	log := domain.NewMockLogCLientI(mockCtrl)
	log.EXPECT().Log(domain.ERROR, gomock.Any(), gomock.Any())

	c := &JobController{
		Logger:      log,
		runningJobs: make(map[domain.AppspaceID]*runningJob),
		fanIn:       make(chan domain.MigrationStatusData, 10),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.eventManifold()
		wg.Done()
	}()

	c.fanIn <- domain.MigrationStatusData{
		ErrString: nulltypes.NewString("boo!", true),
	}

	close(c.fanIn)

	wg.Wait()
}

func TestEventManifoldFinished(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().SetFinished(1, gomock.Any())

	log := domain.NewMockLogCLientI(mockCtrl)
	log.EXPECT().Log(domain.ERROR, gomock.Any(), gomock.Any())

	c := &JobController{
		MigrationJobModel: migrationJobModel,
		Logger:            log,
		runningJobs:       make(map[domain.AppspaceID]*runningJob),
		fanIn:             make(chan domain.MigrationStatusData, 10),
		stop:              true, // prevents startNext from running again
	}

	appspaceID := domain.AppspaceID(7)

	rj := c.createRunningJob(&domain.MigrationJob{
		JobID:      1,
		AppspaceID: appspaceID,
	})

	c.runningJobs[appspaceID] = rj

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.eventManifold()
		wg.Done()
	}()

	c.fanIn <- domain.MigrationStatusData{
		MigrationJob: rj.migrationJob,
		Status:       domain.MigrationFinished,
		ErrString:    nulltypes.NewString("boo!", true),
	}

	close(c.fanIn)

	wg.Wait()
}

func TestFullStartStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{}, nil)

	log := domain.NewMockLogCLientI(mockCtrl)

	c := &JobController{
		MigrationJobModel: migrationJobModel,
		Logger:            log}

	c.Start()
	c.Stop()
}

func TestFullStartStopWithJob(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetPending().Return([]*domain.MigrationJob{}, nil)
	migrationJobModel.EXPECT().SetFinished(1, gomock.Any())

	log := domain.NewMockLogCLientI(mockCtrl)
	log.EXPECT().Log(domain.INFO, gomock.Any(), gomock.Any())

	appspaceID := domain.AppspaceID(7)

	c := &JobController{
		MigrationJobModel: migrationJobModel,
		Logger:            log}

	rj := c.createRunningJob(&domain.MigrationJob{
		JobID:      1,
		AppspaceID: appspaceID,
	})

	c.Start()

	c.runningJobs[appspaceID] = rj

	go func() {
		c.fanIn <- domain.MigrationStatusData{
			MigrationJob: rj.migrationJob,
			Status:       domain.MigrationFinished,
			ErrString:    nulltypes.NewString("", false),
		}
	}()

	c.Stop()
}

// ^^ test close all, etc...
