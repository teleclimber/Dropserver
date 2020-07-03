package userroutes

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/posener/wstest"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/leaktest"
)

func TestRandomString(t *testing.T) {
	str := randomString()
	if len(str) != 24 {
		t.Error("expected 24 chars")
	}
}

func TestStartStop(t *testing.T) {
	defer leaktest.GoroutineLeakCheck(t)()

	liveDataRoutes := &LiveDataRoutes{}
	liveDataRoutes.Init()
	liveDataRoutes.Stop()
}

func TestWSExpiredToken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	defer leaktest.GoroutineLeakCheck(t)()

	tokStr := "abc"
	uid := domain.UserID(7)

	jobCtl := domain.NewMockMigrationJobController(mockCtrl)

	liveDataRoutes := &LiveDataRoutes{
		JobController: jobCtl}
	liveDataRoutes.Init()
	liveDataRoutes.tokens[tokStr] = token{uid, time.Now().Add(-1 * time.Minute)}

	h := &testHandler{
		liveDataRoutes: liveDataRoutes,
		tokStrs:        []string{tokStr},
	}

	d := wstest.NewDialer(h)

	_, _, err := d.Dial("ws://example.org/ws", nil)
	if err == nil {
		t.Error("Expected an error connecting to websocket")
	}

	liveDataRoutes.Stop()
}

func TestWS(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	defer leaktest.GoroutineLeakCheck(t)()

	tokStr := "abc"
	uid := domain.UserID(7)

	jobID := domain.JobID(11)
	curStats := []domain.MigrationStatusData{
		{JobID: jobID, Status: domain.MigrationRunning},
	}

	updateChan := make(chan domain.MigrationStatusData)
	updateChanClosed := false

	jobCtl := domain.NewMockMigrationJobController(mockCtrl)
	jobCtl.EXPECT().SubscribeOwner(uid, tokStr).Return(updateChan, curStats)
	jobCtl.EXPECT().UnsubscribeOwner(uid, tokStr).Do(func(u domain.UserID, s string) {
		if !updateChanClosed {
			close(updateChan)
			updateChanClosed = true
		}
	}).MinTimes(1)

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetJob(jobID).Return(&domain.MigrationJob{JobID: jobID}, nil)

	liveDataRoutes := &LiveDataRoutes{
		JobController:     jobCtl,
		MigrationJobModel: migrationJobModel}
	liveDataRoutes.Init()
	liveDataRoutes.tokens[tokStr] = token{uid, time.Now().Add(time.Minute)}

	h := &testHandler{
		liveDataRoutes: liveDataRoutes,
		tokStrs:        []string{tokStr},
	}

	d := wstest.NewDialer(h)

	conn, _, err := d.Dial("ws://example.org/ws", nil)
	if err != nil {
		t.Error(err)
	}

	var initialStatusRx MigrationStatusResp
	err = conn.ReadJSON(&initialStatusRx)
	if err != nil {
		t.Error(err)
	}
	if initialStatusRx.MigrationJob.JobID != jobID {
		t.Error("expected job with correct job id")
	}

	stat := domain.MigrationStatusData{
		JobID:  jobID,
		Status: domain.MigrationRunning}
	updateChan <- stat

	var statRx MigrationStatusResp
	conn.ReadJSON(&statRx)
	if statRx.Status != "running" {
		t.Error("expected a status with migration running")
	}

	liveDataRoutes.Stop()
}

// I want to test what happens when the conn is closed unexpectedly from the remote.

func TestWSRemoteStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	defer leaktest.GoroutineLeakCheck(t)()

	tokStr := "abc"
	uid := domain.UserID(7)

	jobID := domain.JobID(11)
	curStats := []domain.MigrationStatusData{
		{JobID: jobID, Status: domain.MigrationRunning},
	}

	updateChan := make(chan domain.MigrationStatusData)
	updateChanClosed := false

	jobCtl := domain.NewMockMigrationJobController(mockCtrl)
	jobCtl.EXPECT().SubscribeOwner(uid, tokStr).Return(updateChan, curStats)
	jobCtl.EXPECT().UnsubscribeOwner(uid, tokStr).Do(func(u domain.UserID, s string) {
		if !updateChanClosed {
			close(updateChan)
			updateChanClosed = true
		}
	}).MinTimes(1)

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetJob(jobID).Return(&domain.MigrationJob{JobID: jobID}, nil)

	liveDataRoutes := &LiveDataRoutes{
		JobController:     jobCtl,
		MigrationJobModel: migrationJobModel}
	liveDataRoutes.Init()
	liveDataRoutes.wsConsts.writeWait = time.Second
	liveDataRoutes.wsConsts.pongWait = time.Second

	liveDataRoutes.tokens[tokStr] = token{uid, time.Now().Add(time.Minute)}

	h := &testHandler{
		liveDataRoutes: liveDataRoutes,
		tokStrs:        []string{tokStr},
	}

	d := wstest.NewDialer(h)

	conn, _, err := d.Dial("ws://example.org/ws", nil)
	if err != nil {
		t.Error(err)
	}

	// read some data to be sure the connection is established before we close it
	var initialStatusRx MigrationStatusResp
	err = conn.ReadJSON(&initialStatusRx)
	if err != nil {
		t.Error(err)
	}

	// close connection from remote side
	err = conn.Close()
	if err != nil {
		t.Error(err)
	}

	// The closing of conn should get caught by ping pong and result in cleanly closed client here.
	for range liveDataRoutes.clientClosed {
		if len(liveDataRoutes.clients) == 0 {
			break
		}
	}

	liveDataRoutes.Stop()
}

func TestWSMultipleRemotes(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	defer leaktest.GoroutineLeakCheck(t)()

	tokStr1 := "abc"
	tokStr2 := "def"
	uid := domain.UserID(7)

	jobID := domain.JobID(11)
	curStats := []domain.MigrationStatusData{
		{JobID: jobID, Status: domain.MigrationRunning},
	}

	jobCtl := domain.NewMockMigrationJobController(mockCtrl)

	updateChan1 := make(chan domain.MigrationStatusData)
	updateChan1Closed := false
	jobCtl.EXPECT().SubscribeOwner(uid, tokStr1).Return(updateChan1, curStats)
	jobCtl.EXPECT().UnsubscribeOwner(uid, tokStr1).Do(func(u domain.UserID, s string) {
		if !updateChan1Closed {
			close(updateChan1)
			updateChan1Closed = true
		}
	}).MinTimes(1)

	updateChan2 := make(chan domain.MigrationStatusData)
	updateChan2Closed := false
	jobCtl.EXPECT().SubscribeOwner(uid, tokStr2).Return(updateChan2, curStats)
	jobCtl.EXPECT().UnsubscribeOwner(uid, tokStr2).Do(func(u domain.UserID, s string) {
		if !updateChan2Closed {
			close(updateChan2)
			updateChan2Closed = true
		}
	}).MinTimes(1)

	migrationJobModel := domain.NewMockMigrationJobModel(mockCtrl)
	migrationJobModel.EXPECT().GetJob(jobID).Return(&domain.MigrationJob{JobID: jobID}, nil).Times(2)

	liveDataRoutes := &LiveDataRoutes{
		JobController:     jobCtl,
		MigrationJobModel: migrationJobModel}
	liveDataRoutes.Init()

	liveDataRoutes.tokens[tokStr1] = token{uid, time.Now().Add(time.Minute)}
	liveDataRoutes.tokens[tokStr2] = token{uid, time.Now().Add(time.Minute)}

	h := &testHandler{
		liveDataRoutes: liveDataRoutes,
		tokStrs:        []string{tokStr1, tokStr2},
	}

	d1 := wstest.NewDialer(h)
	conn1, _, err := d1.Dial("ws://example.org/ws", nil)
	if err != nil {
		t.Error(err)
	}

	d2 := wstest.NewDialer(h)
	conn2, _, err := d2.Dial("ws://example.org/ws", nil)
	if err != nil {
		t.Error(err)
	}

	var initialStatusRx MigrationStatusResp
	err = conn1.ReadJSON(&initialStatusRx)
	if err != nil {
		t.Error(err)
	}

	err = conn2.ReadJSON(&initialStatusRx)
	if err != nil {
		t.Error(err)
	}

	stat := domain.MigrationStatusData{
		JobID:  jobID,
		Status: domain.MigrationRunning}
	updateChan1 <- stat
	updateChan2 <- stat

	var statRx MigrationStatusResp
	conn1.ReadJSON(&statRx)
	if statRx.Status != "running" {
		t.Error("conn1 expected a status with migration running")
	}
	conn2.ReadJSON(&statRx)
	if statRx.Status != "running" {
		t.Error("conn2 expected a status with migration running")
	}

	liveDataRoutes.Stop()
}

type testHandler struct {
	liveDataRoutes *LiveDataRoutes
	tokStrs        []string
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var tokStr string
	tokStr, h.tokStrs = h.tokStrs[0], h.tokStrs[1:]
	h.liveDataRoutes.startWsConn(w, r, tokStr)
}
