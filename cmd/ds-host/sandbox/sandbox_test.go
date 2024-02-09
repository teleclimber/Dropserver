package sandbox

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/runtimeconfig"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestNewNeverStartedTask(t *testing.T) {
	tracker := taskTracker{
		notifyCh: make(chan struct{}, 1),
	}

	if tracker.isTiedUp() {
		t.Error("expected to not be tied up")
	}

	ch := tracker.newTask()

	if !tracker.isTiedUp() {
		t.Error("expected to be tied up")
	}

	close(ch)

	<-tracker.notifyCh

	if tracker.isTiedUp() {
		t.Error("expected to not be tied upafter chan close")
	}

	tracker.mux.Lock()
	if tracker.cumul != 0 {
		t.Error("expected zero duration because we never started the task")
	}
	if !tracker.clumpStart.IsZero() || !tracker.clumpEnd.IsZero() {
		t.Error("expected zero clump times because we never started the task")
	}
	if !tracker.lastActive.IsZero() {
		t.Error("expected last active to be zero")
	}
	tracker.mux.Unlock()
}

func TestNewStartedTask(t *testing.T) {
	tracker := taskTracker{
		notifyCh: make(chan struct{}, 1),
	}

	ch := tracker.newTask()

	ch <- struct{}{}

	<-tracker.notifyCh

	if !tracker.isTiedUp() {
		t.Error("expected to be tied up")
	}

	tracker.mux.Lock()
	if tracker.clumpStart.IsZero() {
		t.Error("expected clump to have a start time")
	}
	if !tracker.clumpEnd.IsZero() {
		t.Error("expected clump to not have an end time.")
	}
	tracker.mux.Unlock()

	close(ch)

	<-tracker.notifyCh

	if tracker.isTiedUp() {
		t.Error("expected to not be tied upafter chan close")
	}

	tracker.mux.Lock()
	if tracker.cumul == 0 {
		t.Error("expected non-zero duration")
	}
	if !tracker.clumpStart.IsZero() || !tracker.clumpEnd.IsZero() {
		t.Error("expected zero clump times because we stopped the task")
	}
	if tracker.lastActive.IsZero() {
		t.Error("expected last active to be non-zero")
	}
	tracker.mux.Unlock()
}

func TestTrackMultipleTasks(t *testing.T) {
	tracker := taskTracker{
		notifyCh: make(chan struct{}, 1),
	}

	ch1 := tracker.newTask()
	ch1 <- struct{}{}

	<-tracker.notifyCh

	ch2 := tracker.newTask()
	ch2 <- struct{}{}

	<-tracker.notifyCh

	if !tracker.isTiedUp() {
		t.Error("expected to be tied up")
	}

	tracker.mux.Lock()
	if tracker.numActive != 2 {
		t.Error("expected 2 active tasks")
	}
	tracker.mux.Unlock()

	close(ch2)

	<-tracker.notifyCh

	if !tracker.isTiedUp() {
		t.Error("expected to be tied up")
	}

	tracker.mux.Lock()
	if tracker.numActive != 1 {
		t.Error("expected 1 active tasks")
	}
	if tracker.cumul != 0 {
		t.Error("expected zero duration because we never started the task")
	}
	if tracker.clumpStart.IsZero() {
		t.Error("expected clump start time")
	}
	clumpEnd := tracker.clumpEnd
	if clumpEnd.IsZero() {
		t.Error("clump end should be set")
	}
	if !tracker.lastActive.IsZero() {
		t.Error("expected last active to be zero")
	}
	tracker.mux.Unlock()

	close(ch1)

	<-tracker.notifyCh

	if tracker.isTiedUp() {
		t.Error("expected to not be tied upafter chan close")
	}

	tracker.mux.Lock()
	if tracker.cumul == 0 {
		t.Error("expected non-zero duration")
	}
	if clumpEnd == tracker.clumpEnd {
		t.Error("clump end was not updated on closing second taask")
	}
	if tracker.lastActive.IsZero() {
		t.Error("expected last active to be non-zero")
	}
	tracker.mux.Unlock()
}

// Ttest the status subscription system
func TestStatus(t *testing.T) {
	s := &Sandbox{
		status:        domain.SandboxStarting,
		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	s.setStatus(domain.SandboxReady)

	s.WaitFor(domain.SandboxReady)
}

func TestStatusWait(t *testing.T) {
	s := &Sandbox{
		status:        domain.SandboxStarting,
		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.setStatus(domain.SandboxReady)
	}()

	s.WaitFor(domain.SandboxReady)
}

func TestStatusWaitSkip(t *testing.T) {
	s := &Sandbox{
		status:        domain.SandboxStarting,
		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.setStatus(domain.SandboxKilling)
	}()

	s.WaitFor(domain.SandboxReady)
}

func TestStatusNotReached(t *testing.T) {
	s := &Sandbox{
		status:        domain.SandboxStarting,
		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.setStatus(domain.SandboxReady)
	}()

	go func() {
		s.WaitFor(domain.SandboxKilling)
		t.Error("should not have triggered this status")
	}()

	time.Sleep(200 * time.Millisecond)
}

func TestStatusWaitMultiple(t *testing.T) {
	s := &Sandbox{
		status:        domain.SandboxStarting,
		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.setStatus(domain.SandboxKilling)
	}()

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			s.WaitFor(domain.SandboxReady)
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		s.WaitFor(domain.SandboxKilling)
		wg.Done()
	}()

	wg.Wait()
}

func TestRunnerScriptError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")
	cfg.Exec.AppspacesPath = filepath.Join(dir, "appspaces")

	appl2p := &runtimeconfig.AppLocation2Path{Config: cfg}
	asl2p := &runtimeconfig.AppspaceLocation2Path{Config: cfg}

	ownerID := domain.UserID(22)
	op := opAppspaceRun
	appID := domain.AppID(33)
	version := domain.Version("0.1.2")
	appspaceID := domain.AppspaceID(11)
	appLoc := "app5678"
	asLoc := "as1234"

	sandboxRuns := testmocks.NewMockSandboxRuns(mockCtrl)
	sandboxRuns.EXPECT().Create(domain.SandboxRunIDs{
		Instance:   "ds-host",
		LocalID:    7,
		OwnerID:    ownerID,
		Operation:  op,
		AppID:      appID,
		Version:    version,
		AppspaceID: domain.NewNullAppspaceID(appspaceID),
	}, gomock.Any()).Return(456, nil)
	sandboxRuns.EXPECT().End(456, gomock.Any(), gomock.Any())

	s := &Sandbox{
		id:                    7,
		ownerID:               ownerID,
		operation:             op,
		appVersion:            &domain.AppVersion{AppID: appID, Version: version, Entrypoint: "app.ts", LocationKey: appLoc},
		appspace:              &domain.Appspace{AppspaceID: appspaceID, LocationKey: asLoc},
		status:                domain.SandboxStarting,
		paths:                 &paths{Config: cfg, AppLocation2Path: appl2p, AppspaceLocation2Path: asl2p},
		waitStatusSub:         make(map[domain.SandboxStatus][]chan domain.SandboxStatus),
		SandboxRuns:           sandboxRuns,
		Config:                cfg,
		AppLocation2Path:      appl2p,
		AppspaceLocation2Path: asl2p}

	os.MkdirAll(appl2p.Files(appLoc), 0700)
	os.MkdirAll(asl2p.Files(asLoc), 0700)

	err = os.WriteFile(filepath.Join(appl2p.Files("app5678"), "app.ts"), []byte("setTimeout(hello.world, 100);"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = s.doStart()
	if err == nil {
		t.Error("expected error from sandbox")
	}

	s.WaitFor(domain.SandboxReady)

	if s.Status() == domain.SandboxReady {
		t.Error("sandbox status should be killing or dead")
	}

	s.Kill()

	s.WaitFor(domain.SandboxCleanedUp)
}

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir
	cfg.Exec.SandboxCodePath = getSandboxCodePath()
	cfg.Exec.AppsPath = getAppsPath()
	cfg.Exec.AppspacesPath = getAppspacesPath()

	appl2p := &runtimeconfig.AppLocation2Path{Config: cfg}
	asl2p := &runtimeconfig.AppspaceLocation2Path{Config: cfg}

	ownerID := domain.UserID(22)
	op := opAppspaceRun
	appID := domain.AppID(33)
	version := domain.Version("0.1.2")
	appspaceID := domain.AppspaceID(11)
	appLoc := "app-as"
	asLoc := "basic"

	defer cleanApp(appLoc)
	defer cleanAppspace(asLoc)

	sandboxRuns := testmocks.NewMockSandboxRuns(mockCtrl)
	sandboxRuns.EXPECT().Create(domain.SandboxRunIDs{
		Instance:   "ds-host",
		LocalID:    7,
		OwnerID:    ownerID,
		Operation:  op,
		AppID:      appID,
		Version:    version,
		AppspaceID: domain.NewNullAppspaceID(appspaceID),
	}, gomock.Any()).Return(456, nil)
	sandboxRuns.EXPECT().End(456, gomock.Any(), gomock.Any())

	appVersion := &domain.AppVersion{
		AppID:       appID,
		Version:     version,
		Entrypoint:  "app.ts",
		LocationKey: appLoc}
	appspace := &domain.Appspace{
		AppspaceID:  appspaceID,
		LocationKey: asLoc}

	log := &testLogger2{
		log: func(source, message string) {
			t.Log(source + ": log: " + message)
		}}

	s := &Sandbox{
		ownerID:               ownerID,
		operation:             op,
		id:                    7,
		appspace:              appspace,
		appVersion:            appVersion,
		status:                domain.SandboxStarting,
		paths:                 &paths{Config: cfg, AppLocation2Path: appl2p, AppspaceLocation2Path: asl2p},
		SandboxRuns:           sandboxRuns,
		Config:                cfg,
		AppLocation2Path:      appl2p,
		AppspaceLocation2Path: asl2p,
		Logger:                log,
		waitStatusSub:         make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	err = s.doStart()
	if err != nil {
		t.Fatal(err)
		s.Kill()
	}

	s.WaitFor(domain.SandboxReady)

	if s.Status() != domain.SandboxReady {
		t.Fatal("sandbox status should be ready")
	}

	time.Sleep(time.Second)

	s.Graceful()

	s.WaitFor(domain.SandboxCleanedUp)
}

func TestStartAppOnly(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir
	cfg.Exec.SandboxCodePath = getSandboxCodePath()
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")

	appl2p := &runtimeconfig.AppLocation2Path{Config: cfg}

	ownerID := domain.UserID(22)
	op := opAppInit
	appID := domain.AppID(33)
	version := domain.Version("0.1.2")
	appLoc := "app5678"

	sandboxRuns := testmocks.NewMockSandboxRuns(mockCtrl)
	sandboxRuns.EXPECT().Create(domain.SandboxRunIDs{
		Instance:   "ds-host",
		LocalID:    7,
		OwnerID:    ownerID,
		Operation:  op,
		AppID:      appID,
		Version:    version,
		AppspaceID: domain.NewNullAppspaceID(),
	}, gomock.Any()).Return(456, nil)
	sandboxRuns.EXPECT().End(456, gomock.Any(), gomock.Any())

	appVersion := &domain.AppVersion{
		AppID:       appID,
		Version:     version,
		Entrypoint:  "app.ts",
		LocationKey: appLoc}

	s := &Sandbox{
		ownerID:          ownerID,
		operation:        op,
		id:               7,
		appVersion:       appVersion,
		status:           domain.SandboxStarting,
		paths:            &paths{Config: cfg, AppLocation2Path: appl2p},
		SandboxRuns:      sandboxRuns,
		Config:           cfg,
		AppLocation2Path: appl2p,
		waitStatusSub:    make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	os.MkdirAll(appl2p.Files(appLoc), 0700)

	// app code has to setCallback to trigger sandbox ready
	app_code := []byte("//@ts-ignore\nwindow.DROPSERVER.appRoutes.setCallback(); console.log('hw');")
	err = os.WriteFile(filepath.Join(appl2p.Files(appLoc), "app.ts"), app_code, 0600)
	if err != nil {
		t.Error(err)
	}

	err = s.doStart()
	if err != nil {
		t.Fatal(err)
		s.Kill()
	}

	s.WaitFor(domain.SandboxReady)

	if s.Status() != domain.SandboxReady {
		t.Fatal("sandbox status should be ready")
	}

	time.Sleep(time.Second)

	s.Graceful()

	s.WaitFor(domain.SandboxCleanedUp)
}

// TEST DISABLED bc ExecFn not functional rn.
// func __TestExecFn(t *testing.T) {
// 	dir, err := os.MkdirTemp("", "")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	al2p := appl2p{base: dir}
// 	appLocation := "app-loc"

// 	os.MkdirAll(al2p.Files(appLocation), 0700)
// 	os.MkdirAll(filepath.Join(dir, "appspace-loc"), 0700)

// 	cfg := &domain.RuntimeConfig{}
// 	cfg.Sandbox.SocketsDir = dir
// 	cfg.Exec.SandboxCodePath = getSandboxCodePath()
// 	cfg.Exec.AppspacesPath = dir

// 	appVersion := &domain.AppVersion{
// 		LocationKey: appLocation}
// 	appspace := &domain.Appspace{
// 		AppspaceID:  domain.AppspaceID(13),
// 		LocationKey: "appspace-loc"}

// 	scriptPath := path.Join(al2p.Files(appLocation), "app.ts")

// 	err = os.WriteFile(scriptPath, []byte("export function abc() { console.log('hello workd'); }"), 0644)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	log := &testLogger2{
// 		log: func(source, message string) {
// 			t.Log("log: " + message)
// 		}}

// 	s := &Sandbox{
// 		id:            7,
// 		appspace:      appspace,
// 		appVersion:    appVersion,
// 		status:        domain.SandboxStarting,
// 		Config:        cfg,
// 		Logger:        log,
// 		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

// 	err = s.doStart()
// 	if err != nil {
// 		s.Kill()
// 		t.Error(err)
// 	}

// 	s.WaitFor(domain.SandboxReady)

// 	if s.Status() != domain.SandboxReady {
// 		t.Error("sandbox status should be ready")
// 	}

// 	err = s.ExecFn(domain.AppspaceRouteHandler{
// 		File:     "@app/app.ts",
// 		Function: "abc",
// 	})
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	s.Graceful()
// }

// There should be a test that hits the appspace's data files.

// TODO Test fails. It depends on ExecFn, so it can't pass until that's fixed.
// TEST DISABLED
// func __TestExecForbiddenImport(t *testing.T) {
// 	dir, err := os.MkdirTemp("", "")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	os.MkdirAll(filepath.Join(dir, "appspace-loc"), 0700)

// 	forbiddenDir, err := os.MkdirTemp("", "")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(forbiddenDir)

// 	cfg := &domain.RuntimeConfig{}
// 	cfg.Sandbox.SocketsDir = dir
// 	cfg.Exec.SandboxCodePath = getSandboxCodePath()
// 	cfg.Exec.AppspacesPath = dir

// 	appVersion := &domain.AppVersion{
// 		LocationKey: "app-loc"}
// 	appspace := &domain.Appspace{
// 		AppspaceID:  domain.AppspaceID(13),
// 		LocationKey: "appspace-loc"}

// 	scriptPath := path.Join(forbiddenDir, "bad.ts")

// 	err = os.WriteFile(scriptPath, []byte("export function abc() { console.log('hello bad'); }"), 0644)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	s := &Sandbox{
// 		id:            7,
// 		appspace:      appspace,
// 		appVersion:    appVersion,
// 		status:        domain.SandboxStarting,
// 		Config:        cfg,
// 		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

// 	err = s.doStart()
// 	if err != nil {
// 		s.Kill()
// 		t.Error(err)
// 	}

// 	s.WaitFor(domain.SandboxReady)

// 	if s.Status() != domain.SandboxReady {
// 		t.Error("sandbox status should be ready")
// 	}

// 	err = s.ExecFn(domain.AppspaceRouteHandler{
// 		File:     scriptPath,
// 		Function: "abc",
// 	})
// 	if err == nil {
// 		t.Error("Expected an error")
// 	}

// 	s.Graceful()
// }

func getSandboxCodePath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, "../../../denosandboxcode/")
}

func getAppsPath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, "/testfixtures/apps/")
}

func cleanApp(loc string) {
	dir := filepath.Join(getAppsPath(), loc)
	os.Remove(filepath.Join(dir, "bootstrap.js"))
}

func getAppspacesPath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, "/testfixtures/appspaces/")
}

func cleanAppspace(loc string) {
	dir := filepath.Join(getAppspacesPath(), loc)
	os.Remove(filepath.Join(dir, "import-paths.json"))
	os.RemoveAll(filepath.Join(dir, "deno-dir"))
}

// test logger
type testLogger2 struct {
	log func(source string, message string)
}

func (l *testLogger2) Log(source string, message string) {
	l.log(source, message)
}

// l2p Location2Path standin
// type appl2p struct {
// 	base string
// }

// func (l *appl2p) Base(loc string) string {
// 	return filepath.Join(l.base, loc)
// }
// func (l *appl2p) Meta(loc string) string {
// 	return filepath.Join(l.base, loc)
// }
// func (l *appl2p) Files(loc string) string {
// 	return filepath.Join(l.base, loc, "app")
// }
// func (l *appl2p) ImportMap(loc string) string {
// 	return filepath.Join(l.base, loc, "import-map.json")
// }

// // l2p Location2Path standin
// type appspacel2p struct {
// 	base string
// }

// func (l *appspacel2p) Base(loc string) string {
// 	return filepath.Join(l.base, loc)
// }
// func (l *appspacel2p) Files(loc string) string {
// 	return filepath.Join(l.base, loc, "app")
// }
// func (l *appspacel2p) Data(loc string) string {
// 	return filepath.Join(l.base, loc, "app")
// }
// func (l *appspacel2p) Avatars(loc string) string {
// 	return filepath.Join(l.base, loc, "avatars")
// }
// func (l *appspacel2p) ImportMap(loc string) string {
// 	return filepath.Join(l.base, loc, "import-map.json")
// }
