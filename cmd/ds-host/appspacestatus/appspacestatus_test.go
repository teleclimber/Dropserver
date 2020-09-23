package appspacestatus

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

func TestLoadStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{Paused: true}, nil)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersion(gomock.Any(), gomock.Any()).Return(&domain.AppVersion{Schema: 3}, nil)

	migrationJobs := testmocks.NewMockMigrationJobController(mockCtrl)
	migrationJobs.EXPECT().GetRunningJobs().Return([]domain.MigrationStatusData{{AppspaceID: appspaceID}})

	appspaceInfoModels := testmocks.NewMockAppspaceInfoModels(mockCtrl)
	appspaceInfoModels.EXPECT().GetSchema(appspaceID).Return(4, nil)

	s := &AppspaceStatus{
		AppspaceModel:      appspaceModel,
		AppModel:           appModel,
		MigrationJobs:      migrationJobs,
		AppspaceInfoModels: appspaceInfoModels,
	}

	status := s.loadStatus(appspaceID)
	if status.paused != true {
		t.Error("paused should be true")
	}
	if !status.migrating {
		t.Error("Expected migrating to be true")
	}
	if status.dataSchema != 4 {
		t.Error("data schema should be 4")
	}
	if status.appVersionSchema != 3 {
		t.Error("app version schema should be 3")
	}
	if status.problem {
		t.Error("should not be a problem")
	}
}

func TestReady(t *testing.T) {
	appspaceID := domain.AppspaceID(7)

	cases := []struct {
		status appspaceStatus
		ready  bool
	}{{
		status: appspaceStatus{
			migrating:        false,
			paused:           false,
			appVersionSchema: 3,
			dataSchema:       3,
			problem:          false},
		ready: true}, {
		status: appspaceStatus{
			migrating:        true, //migrating
			paused:           false,
			appVersionSchema: 3,
			dataSchema:       3,
			problem:          false},
		ready: false}, {
		status: appspaceStatus{
			migrating:        false,
			paused:           false,
			appVersionSchema: 3,
			dataSchema:       4, // mismatched data schema
			problem:          false},
		ready: false},
	}

	s := AppspaceStatus{}
	s.status = make(map[domain.AppspaceID]appspaceStatus)

	for _, c := range cases {
		s.status[appspaceID] = c.status
		ready := s.Ready(appspaceID)
		if ready != c.ready {
			t.Errorf("Expected %v", c.ready)
		}
	}
}

func TestPauseEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	appspaceStatusEvents := testmocks.NewMockAppspaceStatusEvents(mockCtrl)
	appspaceStatusEvents.EXPECT().Send(appspaceID, domain.AppspaceStatusEvent{AppspaceID: appspaceID, Paused: true})

	s := AppspaceStatus{
		AppspaceStatusEvents: appspaceStatusEvents,
		status:               make(map[domain.AppspaceID]appspaceStatus),
	}

	pauseChan := make(chan domain.AppspacePausedEvent)
	go s.handleAppspacePause(pauseChan)

	migrateChan := make(chan domain.MigrationStatusData)
	go s.handleMigrationJobUpdate(migrateChan)

	s.status[appspaceID] = appspaceStatus{
		paused: false}

	pauseChan <- domain.AppspacePausedEvent{
		AppspaceID: appspaceID,
		Paused:     true}

	time.Sleep(time.Millisecond * 200) // have to give the code in the goroutine a chance to change the status

	status := s.getStatus(appspaceID)
	if !status.paused {
		t.Error("expected paused")
	}
}

func TestMigrationEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	status1 := appspaceStatus{
		migrating:        false,
		paused:           false,
		appVersionSchema: 3,
		dataSchema:       3,
		problem:          false}
	event1 := domain.AppspaceStatusEvent{
		AppspaceID:       appspaceID,
		AppVersionSchema: 3,
		AppspaceSchema:   3,
		Migrating:        true,
		Paused:           false,
		Problem:          false,
	}
	event2 := event1
	event2.Migrating = false
	event2.AppspaceSchema = 4

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{}, nil)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersion(gomock.Any(), gomock.Any()).Return(&domain.AppVersion{Schema: 3}, nil)

	migrationJobs := testmocks.NewMockMigrationJobController(mockCtrl)
	migrationJobs.EXPECT().GetRunningJobs().Return([]domain.MigrationStatusData{})

	appspaceInfoModels := testmocks.NewMockAppspaceInfoModels(mockCtrl)
	appspaceInfoModels.EXPECT().GetSchema(appspaceID).Return(4, nil)

	appspaceStatusEvents := testmocks.NewMockAppspaceStatusEvents(mockCtrl)
	appspaceStatusEvents.EXPECT().Send(appspaceID, event1)
	appspaceStatusEvents.EXPECT().Send(appspaceID, event2)

	s := AppspaceStatus{
		AppspaceModel:        appspaceModel,
		AppModel:             appModel,
		AppspaceInfoModels:   appspaceInfoModels,
		MigrationJobs:        migrationJobs,
		AppspaceStatusEvents: appspaceStatusEvents,
		status:               make(map[domain.AppspaceID]appspaceStatus),
	}

	migrateChan := make(chan domain.MigrationStatusData)
	go s.handleMigrationJobUpdate(migrateChan)

	s.status[appspaceID] = status1

	migrateChan <- domain.MigrationStatusData{
		AppspaceID: appspaceID,
		Finished:   nulltypes.NewTime(time.Now(), false)} // send null Finished time, indicating ongoing migration

	time.Sleep(time.Millisecond * 200) // have to give the code in the goroutine a chance to change the status

	status := s.getStatus(appspaceID)
	if !status.migrating {
		t.Error("expected migrating")
	}

	migrateChan <- domain.MigrationStatusData{
		AppspaceID: appspaceID,
		Finished:   nulltypes.NewTime(time.Now(), true), // send a valid Finished time to indicate migration is complete
	}

	time.Sleep(time.Millisecond * 200) // have to give the code in the goroutine a chance to change the status

	status = s.getStatus(appspaceID)
	if status.migrating {
		t.Error("expected not migrating anymore")
	}
}

func TestWaitStopped(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	appspaceRoutes := testmocks.NewMockAppspaceRoutes(mockCtrl)
	appspaceRoutes.EXPECT().SubscribeLiveCount(appspaceID, gomock.Any()).Do(
		func(asID domain.AppspaceID, ch chan<- int) {
			go func() {
				time.Sleep(time.Millisecond * 50)
				ch <- 1
				time.Sleep(time.Millisecond * 50)
				ch <- 0
			}()
		}).Return(2)
	appspaceRoutes.EXPECT().UnsubscribeLiveCount(appspaceID, gomock.Any())

	s := AppspaceStatus{
		AppspaceRoutes: appspaceRoutes}

	s.WaitStopped(appspaceID)
}
