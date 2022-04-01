package sandbox

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestFindAppspace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	a1 := domain.AppspaceID(12)
	v1 := domain.AppVersion{
		AppID:   domain.AppID(45),
		Version: domain.Version("0.1.0")}
	v2 := domain.AppVersion{
		AppID:   domain.AppID(45),
		Version: domain.Version("0.2.0")}
	s1 := makeFindableSandbox(mockCtrl, domain.NewNullAppspaceID(a1), &v1, domain.SandboxReady, opAppspaceRun)
	s2 := makeFindableSandbox(mockCtrl, domain.NewNullAppspaceID(a1), &v2, domain.SandboxReady, opAppspaceRun)
	s1dead := makeFindableSandbox(mockCtrl, domain.NewNullAppspaceID(a1), &v1, domain.SandboxKilling, opAppspaceRun)
	wrongAppspace := makeFindableSandbox(mockCtrl, domain.NewNullAppspaceID(domain.AppspaceID(99)), &v1, domain.SandboxReady, opAppspaceRun)
	s1WrongOp := makeFindableSandbox(mockCtrl, domain.NewNullAppspaceID(a1), &v1, domain.SandboxReady, opAppspaceMigration)

	m := Manager{
		sandboxes: []domain.SandboxI{s1WrongOp, wrongAppspace, s1dead, s2, s1},
	}

	// find without specifying app version
	found, ok := m.findAppspaceSandbox(nil, a1)
	if !ok || found != s2 { // s2 is the first valid sandbox it will find
		t.Error("not found or wrong sandbox")
	}

	// specify app version
	found, ok = m.findAppspaceSandbox(&v1, a1)
	if !ok || found != s1 {
		t.Error("not found or wrong sandbox")
	}

	// specify non-existing app version
	vBad := domain.AppVersion{
		AppID:   domain.AppID(45),
		Version: domain.Version("0.13.0")}
	_, ok = m.findAppspaceSandbox(&vBad, a1)
	if ok {
		t.Error("should not have found a sandbox")
	}
}

func makeFindableSandbox(mockCtrl *gomock.Controller, appspaceID domain.NullAppspaceID, appVersion *domain.AppVersion, status domain.SandboxStatus, operation string) *domain.MockSandboxI {
	s := domain.NewMockSandboxI(mockCtrl)
	s.EXPECT().AppspaceID().AnyTimes().Return(appspaceID)
	s.EXPECT().AppVersion().AnyTimes().Return(appVersion)
	s.EXPECT().Status().AnyTimes().Return(status)
	s.EXPECT().Operation().AnyTimes().Return(operation)
	return s
}

func TestGetStartStoppablesEmpty(t *testing.T) {
	sandboxes := make([]domain.SandboxI, 0)

	s := getStartStoppables(sandboxes)

	expected := startStopStatus{
		startables: []scored{},
		stoppables: []scored{},
		numRunning: 0,
		numDying:   0,
	}

	err := assertStartStopStatus(s, expected)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStartStoppablesOneStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var lastActive time.Time
	s1 := makeScoredSandbox(mockCtrl, domain.SandboxPrepared, opAppspaceRun, lastActive, false)
	sandboxes := []domain.SandboxI{s1}

	s := getStartStoppables(sandboxes)

	expected := startStopStatus{
		startables: []scored{{sandbox: s1, score: 10.0}},
		stoppables: []scored{},
		numRunning: 0,
		numDying:   0,
	}

	err := assertStartStopStatus(s, expected)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStartStoppablesStopDying(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var lastActive time.Time
	s1 := makeScoredSandbox(mockCtrl, domain.SandboxKilling, opAppspaceRun, lastActive, false) // already stopping
	lastActive = time.Now().Add(-1 * time.Second)
	s2 := makeScoredSandbox(mockCtrl, domain.SandboxReady, opAppspaceRun, lastActive, false)       // can be stopped
	s3 := makeScoredSandbox(mockCtrl, domain.SandboxReady, opAppspaceRun, lastActive, true)        // tied up, so can't be stopped
	s4 := makeScoredSandbox(mockCtrl, domain.SandboxReady, opAppInit, lastActive, false)           // app init, so can't be stopped
	s5 := makeScoredSandbox(mockCtrl, domain.SandboxReady, opAppspaceMigration, lastActive, false) // appspace migration, so can't be stopped
	sandboxes := []domain.SandboxI{s1, s2, s3, s4, s5}

	s := getStartStoppables(sandboxes)

	expected := startStopStatus{
		startables: []scored{},
		stoppables: []scored{{sandbox: s2, score: 1.0}},
		numRunning: 5, // the one shutting down counts as running
		numDying:   1,
	}

	err := assertStartStopStatus(s, expected)
	if err != nil {
		t.Error(err)
	}
}

func TestScoredSort(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var lastActive time.Time

	startMigration := makeScoredSandbox(mockCtrl, domain.SandboxPrepared, opAppspaceMigration, lastActive, false) // appspace migration: lower priority
	startRun := makeScoredSandbox(mockCtrl, domain.SandboxPrepared, opAppspaceRun, lastActive, false)             // appspace run: higher priority

	stop5 := makeScoredSandbox(mockCtrl, domain.SandboxReady, opAppspaceRun, time.Now().Add(-5*time.Second), false)
	stop1 := makeScoredSandbox(mockCtrl, domain.SandboxReady, opAppspaceRun, time.Now().Add(-1*time.Second), false)
	stop10 := makeScoredSandbox(mockCtrl, domain.SandboxReady, opAppspaceRun, time.Now().Add(-10*time.Second), false)

	sandboxes := []domain.SandboxI{startMigration, startRun, stop5, stop1, stop10}

	status := getStartStoppables(sandboxes)

	expected := startStopStatus{
		startables: []scored{{startRun, 10.0}, {startMigration, 0.0}},
		stoppables: []scored{{stop10, 10.0}, {stop5, 5.0}, {stop1, 1.0}},
		numRunning: 3,
		numDying:   0,
	}
	err := assertStartStopStatus(status, expected)
	if err != nil {
		t.Error(err)
	}
}

func makeScoredSandbox(mockCtrl *gomock.Controller, status domain.SandboxStatus, operation string, lastActive time.Time, tiedUp bool) *domain.MockSandboxI {
	s := domain.NewMockSandboxI(mockCtrl)
	s.EXPECT().Status().AnyTimes().Return(status)
	s.EXPECT().Operation().AnyTimes().Return(operation)
	s.EXPECT().TiedUp().AnyTimes().Return(tiedUp)
	s.EXPECT().LastActive().AnyTimes().Return(lastActive)
	return s
}

// Starting and stopping sandboxes:

func TestDoStartStop(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Scenario:
	// three running sandboxes, among them one stopping.
	// 1 startable.
	// result should be 1 being stopped and none started
	// to end up with 1 running, leaving room for the 1 startable and 1 overhead.

	config := domain.RuntimeConfig{}
	config.Sandbox.Num = 3

	s1 := makeUntouchedSandbox(mockCtrl)
	s2 := makeStopSandbox(mockCtrl)
	status := startStopStatus{
		startables: []scored{{s1, 10.0}},
		stoppables: []scored{{s2, 1.0}},
		numRunning: 3,
		numDying:   1,
	}

	doStartStop(&config, status)
}

func TestDoStartStop2(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Scenario:
	// 2 running sandboxes, among them one stopping.
	// 2 startable.
	// result should be 1 being started and one stopping (in addition to the already stopping one)
	// to allow starting the second sandbox after, and leave 1 overhead

	config := domain.RuntimeConfig{}
	config.Sandbox.Num = 3

	s1 := makeStartSandbox(mockCtrl)
	s2 := makeUntouchedSandbox(mockCtrl)
	s3 := makeStopSandbox(mockCtrl)
	s4 := makeUntouchedSandbox(mockCtrl)
	status := startStopStatus{
		startables: []scored{{s1, 10.0}, {s2, 10.0}},
		stoppables: []scored{{s3, 1.0}, {s4, 1.0}},
		numRunning: 2,
		numDying:   1,
	}

	doStartStop(&config, status)
}

func makeUntouchedSandbox(mockCtrl *gomock.Controller) *domain.MockSandboxI {
	s := domain.NewMockSandboxI(mockCtrl)
	return s
}
func makeStartSandbox(mockCtrl *gomock.Controller) *domain.MockSandboxI {
	s := domain.NewMockSandboxI(mockCtrl)
	s.EXPECT().Start()
	return s
}
func makeStopSandbox(mockCtrl *gomock.Controller) *domain.MockSandboxI {
	s := domain.NewMockSandboxI(mockCtrl)
	s.EXPECT().Graceful()
	return s
}

// assert functions:
func assertStartStopStatus(a, b startStopStatus) error {
	if a.numRunning != b.numRunning {
		return fmt.Errorf("numRunning different: %v, %v", a.numRunning, b.numRunning)
	}
	if a.numDying != b.numDying {
		return fmt.Errorf("numDying different: %v, %v", a.numDying, b.numDying)
	}
	err := assertSliceScored(a.startables, b.startables)
	if err != nil {
		return err
	}
	err = assertSliceScored(a.stoppables, b.stoppables)
	if err != nil {
		return err
	}
	return nil
}

func assertSliceScored(a, b []scored) error {
	if len(a) != len(b) {
		return fmt.Errorf("different lengths for slice of scored: %v, %v", len(a), len(b))
	}
	for i := range a {
		err := assertScored(a[i], b[i])
		if err != nil {
			return err
		}
	}
	return nil
}
func assertScored(a, b scored) error {
	if math.Abs(a.score-b.score) > 0.2 { // some scores are based on time, so look for approximately correct
		return fmt.Errorf("different scores: %v, %v", a.score, b.score)
	}
	if a.sandbox != b.sandbox {
		return fmt.Errorf("andboxes diefferent %v, %v", a.sandbox, b.sandbox)
	}
	return nil
}

// Test of test mocks:
func TestMockSandboxEqual(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	lastActive := time.Now().Add(-1 * time.Second)
	s1 := makeScoredSandbox(mockCtrl, domain.SandboxStarting, opAppspaceRun, lastActive, false)
	s2 := s1
	if s1 != s2 {
		t.Error("same sandbox not equal!")
	}

	s3 := makeScoredSandbox(mockCtrl, domain.SandboxStarting, opAppspaceRun, lastActive, false)
	if s1 == s3 {
		t.Error("identical sandbox considered equal!")
	}
}
