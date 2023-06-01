package appspacestatus

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
	"github.com/teleclimber/DropServer/internal/leaktest"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

func TestLoadStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{Paused: true}, nil)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersion(gomock.Any(), gomock.Any()).Return(domain.AppVersion{Schema: 3}, nil)

	appspaceInfoModel := testmocks.NewMockAppspaceInfoModel(mockCtrl)
	appspaceInfoModel.EXPECT().GetSchema(appspaceID).Return(4, nil)

	s := &AppspaceStatus{
		AppspaceModel:     appspaceModel,
		AppModel:          appModel,
		AppspaceInfoModel: appspaceInfoModel,
	}

	status := s.getData(appspaceID)
	if status.paused != true {
		t.Error("paused should be true")
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
	leaktest.GoroutineLeakCheck(t)

	appspaceID := domain.AppspaceID(7)

	cases := []struct {
		status statusData
		ready  bool
	}{{
		status: statusData{
			paused:           false,
			appVersionSchema: 3,
			dataSchema:       3,
			problem:          false},
		ready: true}, {
		status: statusData{
			paused:           false,
			tempPauses:       []tempPause{{reason: "migrating"}},
			appVersionSchema: 3,
			dataSchema:       3,
			problem:          false},
		ready: false}, {
		status: statusData{
			paused:           false,
			appVersionSchema: 3,
			dataSchema:       4, // mismatched data schema
			problem:          false},
		ready: false},
	}

	s := AppspaceStatus{}
	s.status = make(map[domain.AppspaceID]*status)

	for _, c := range cases {
		s.status[appspaceID] = &status{data: c.status}
		ready := s.Ready(appspaceID)
		if ready != c.ready {
			t.Errorf("Expected %v", c.ready)
		}
	}
}

func TestTempPause(t *testing.T) {
	leaktest.GoroutineLeakCheck(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	status1 := statusData{
		paused:           false,
		appVersionSchema: 3,
		dataSchema:       3,
		problem:          false}

	appspaceStatusEvents := testmocks.NewMockAppspaceStatusEvents(mockCtrl)
	appspaceStatusEvents.EXPECT().Send(appspaceID, gomock.Any()).AnyTimes()

	appspaceRouter := testmocks.NewMockAppspaceRouter(mockCtrl)
	appspaceRouter.EXPECT().SubscribeLiveCount(appspaceID, gomock.Any())
	appspaceRouter.EXPECT().UnsubscribeLiveCount(appspaceID, gomock.Any())
	s := AppspaceStatus{
		AppspaceRouter:       appspaceRouter,
		AppspaceStatusEvents: appspaceStatusEvents,
		status:               make(map[domain.AppspaceID]*status),
	}
	s.status[appspaceID] = &status{data: status1}

	if s.IsTempPaused(appspaceID) {
		t.Error("expected not temp paused")
	}

	doneCh := s.WaitTempPaused(appspaceID, "test")

	if s.Ready(appspaceID) {
		t.Error("should not be ready")
	}
	if !s.IsTempPaused(appspaceID) {
		t.Error("expected temp paused")
	}

	close(doneCh)

	time.Sleep(10 * time.Millisecond) // have to sleep because closing the chan does not take effect synchronously here.
	// can maybe change this if/when we have WaitReady

	if !s.Ready(appspaceID) {
		t.Error("should be ready")
	}
	if s.IsTempPaused(appspaceID) {
		t.Error("expected not temp paused")
	}
}

func TestMultiTempPause(t *testing.T) {
	leaktest.GoroutineLeakCheck(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	status1 := statusData{
		paused:           false,
		appVersionSchema: 3,
		dataSchema:       3,
		problem:          false}

	appspaceStatusEvents := testmocks.NewMockAppspaceStatusEvents(mockCtrl)
	appspaceStatusEvents.EXPECT().Send(appspaceID, gomock.Any()).Times(2)

	appspaceRouter := testmocks.NewMockAppspaceRouter(mockCtrl)
	appspaceRouter.EXPECT().SubscribeLiveCount(appspaceID, gomock.Any()).Times(2)
	appspaceRouter.EXPECT().UnsubscribeLiveCount(appspaceID, gomock.Any()).Times(2)
	s := AppspaceStatus{
		AppspaceRouter:       appspaceRouter,
		AppspaceStatusEvents: appspaceStatusEvents,
		status:               make(map[domain.AppspaceID]*status),
	}
	s.status[appspaceID] = &status{data: status1}

	allDone := make(chan struct{})

	doneCh1 := s.WaitTempPaused(appspaceID, "test1")

	go func() {
		doneCh2 := s.WaitTempPaused(appspaceID, "test2")
		time.Sleep(10 * time.Millisecond)
		if s.Ready(appspaceID) {
			t.Error("should not be ready")
		}
		close(doneCh2)
		time.Sleep(10 * time.Millisecond)
		close(allDone)
	}()

	if s.Ready(appspaceID) {
		t.Error("should not be ready")
	}

	time.Sleep(10 * time.Millisecond)

	close(doneCh1)
	<-allDone
	if !s.Ready(appspaceID) {
		t.Error("should be ready")
	}
}

func TestMigrationEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	status1 := statusData{
		paused:           false,
		appVersionSchema: 3,
		dataSchema:       3,
		problem:          false}
	event1 := domain.AppspaceStatusEvent{
		AppspaceID:       appspaceID,
		AppVersionSchema: 3,
		AppspaceSchema:   4,
		Paused:           false,
		Problem:          false,
	}

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{}, nil)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersion(gomock.Any(), gomock.Any()).Return(domain.AppVersion{Schema: 3}, nil)

	appspaceInfoModel := testmocks.NewMockAppspaceInfoModel(mockCtrl)
	appspaceInfoModel.EXPECT().GetSchema(appspaceID).Return(4, nil)

	appspaceStatusEvents := testmocks.NewMockAppspaceStatusEvents(mockCtrl)
	appspaceStatusEvents.EXPECT().Send(appspaceID, event1)

	s := AppspaceStatus{
		AppspaceModel:        appspaceModel,
		AppModel:             appModel,
		AppspaceInfoModel:    appspaceInfoModel,
		AppspaceStatusEvents: appspaceStatusEvents,
		status:               make(map[domain.AppspaceID]*status),
	}

	migrateChan := make(chan domain.MigrationJob)
	go s.handleMigrationJobUpdate(migrateChan)

	s.status[appspaceID] = &status{data: status1}

	migrateChan <- domain.MigrationJob{
		AppspaceID: appspaceID,
		Finished:   nulltypes.NewTime(time.Now(), false)} // send null Finished time, indicating ongoing migration

	time.Sleep(time.Millisecond * 20) // have to give the code in the goroutine a chance to change the status

	// the status hasn't changed yet.

	migrateChan <- domain.MigrationJob{
		AppspaceID: appspaceID,
		Finished:   nulltypes.NewTime(time.Now(), true), // send a valid Finished time to indicate migration is complete
	}

	time.Sleep(time.Millisecond * 20) // have to give the code in the goroutine a chance to change the status

}

func TestWaitStopped(t *testing.T) {
	leaktest.GoroutineLeakCheck(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	appspaceRouter := testmocks.NewMockAppspaceRouter(mockCtrl)
	appspaceRouter.EXPECT().SubscribeLiveCount(appspaceID, gomock.Any()).Do(
		func(asID domain.AppspaceID, ch chan<- int) {
			go func() {
				time.Sleep(time.Millisecond * 50)
				ch <- 1
				time.Sleep(time.Millisecond * 50)
				ch <- 0
			}()
		}).Return(2)
	appspaceRouter.EXPECT().UnsubscribeLiveCount(appspaceID, gomock.Any()).Do(
		func(asID domain.AppspaceID, ch chan<- int) {
			close(ch)
		})

	s := AppspaceStatus{
		AppspaceRouter: appspaceRouter}

	s.WaitStopped(appspaceID)
}

func TestLockClosed(t *testing.T) {
	leaktest.GoroutineLeakCheck(t)

	s := AppspaceStatus{
		closed: make(map[domain.AppspaceID]bool),
	}

	appspaceID := domain.AppspaceID(7)

	if s.IsLockedClosed(appspaceID) {
		t.Error("should not be locked closed")
	}
	ch, ok := s.LockClosed(appspaceID)
	if !ok {
		t.Error("expected OK")
	}
	if !s.IsLockedClosed(appspaceID) {
		t.Error("should be locked closed")
	}
	_, ok = s.LockClosed(appspaceID)
	if ok {
		t.Error("expected not ok")
	}
	close(ch)

	// need to sleep for closed channel to take effect.
	// We may find we need a WaitUnlockedClosed or something?
	// Or have state take closed into account, and use WaitClosed
	time.Sleep(10 * time.Millisecond)
	if s.IsLockedClosed(appspaceID) {
		t.Error("should not be locked closed")
	}
}
