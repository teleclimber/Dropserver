//go:build linux

package sandbox

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/runtimeconfig"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
	"github.com/teleclimber/DropServer/internal/leaktest"
)

func TestBwrapJsonStatus(t *testing.T) {
	leaktest.GoroutineLeakCheck(t)

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	b, err := NewBwrapJsonStatus(dir)
	if err != nil {
		t.Error(err)
	}

	bwrapArgs := []string{
		"--json-status-fd", "3",
		"--ro-bind", "/", "/",
		"true",
	}
	cmd := exec.Command("bwrap", bwrapArgs...)
	cmd.ExtraFiles = []*os.File{b.GetFile()}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Error(err)
	}
	go handleStd(stdout, "stdout")

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Error(err)
	}
	go handleStd(stderr, "stderr")

	err = cmd.Start() // returns right away
	if err != nil {
		t.Error(err)
	}

	_, ok := b.WaitPid()
	if !ok {
		t.Error("waitpid not OK")
	}

	cmd.Wait()
}

func TestStartAppOnlyBwrap(t *testing.T) {
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
	deno, err := getDenoAbsPath()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Sandbox.DenoPath = deno
	cfg.Sandbox.UseBubblewrap = true
	cfg.Sandbox.BwrapMapPaths = []string{"/usr/lib", "/etc", "/lib64"}
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
		LocationKey: appLoc}

	log := &testLogger2{
		log: func(source, message string) {
			t.Log(source + ": log: " + message)
		}}

	s := &Sandbox{
		ownerID:          ownerID,
		operation:        op,
		id:               7,
		appVersion:       appVersion,
		status:           domain.SandboxStarting,
		paths:            &paths{Config: cfg, AppLocation2Path: appl2p},
		Logger:           log,
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

	s.WaitFor(domain.SandboxDead)
}

func TestStartAppspaceBwrap(t *testing.T) {
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
	deno, err := getDenoAbsPath()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Sandbox.DenoPath = deno
	cfg.Sandbox.UseBubblewrap = true
	cfg.Sandbox.BwrapMapPaths = []string{"/usr/lib", "/etc", "/lib64"}
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")
	cfg.Exec.AppspacesPath = filepath.Join(dir, "appspaces")

	appl2p := &runtimeconfig.AppLocation2Path{Config: cfg}
	asl2p := &runtimeconfig.AppspaceLocation2Path{Config: cfg}

	ownerID := domain.UserID(22)
	op := opAppInit
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

	appVersion := &domain.AppVersion{
		AppID:       appID,
		Version:     version,
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
		Logger:                log,
		SandboxRuns:           sandboxRuns,
		Config:                cfg,
		AppLocation2Path:      appl2p,
		AppspaceLocation2Path: asl2p,
		waitStatusSub:         make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	os.MkdirAll(appl2p.Files(appLoc), 0700)
	os.MkdirAll(asl2p.Files(asLoc), 0700)
	os.MkdirAll(asl2p.Avatars(asLoc), 0700)

	appspaceTxt := []byte("appspace-data-5678")
	err = os.WriteFile(filepath.Join(asl2p.Files(asLoc), "asdat.txt"), appspaceTxt, 0600)
	if err != nil {
		t.Error(err)
	}
	// app code has to setCallback to trigger sandbox ready
	app_code := "//@ts-ignore\nwindow.DROPSERVER.appRoutes.setCallback();\n"
	app_code += "console.log(await Deno.readTextFile('/appspace-data/files/asdat.txt'));"
	app_code += "console.log('hw');"
	err = os.WriteFile(filepath.Join(appl2p.Files(appLoc), "app.ts"), []byte(app_code), 0600)
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

	s.WaitFor(domain.SandboxDead)
}

func handleStd(rc io.ReadCloser, source string) {
	buf := make([]byte, 1000)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			fmt.Println(source, string(buf[0:n]))
		}
		if err != nil {
			break
		}
	}
}

func getDenoAbsPath() (string, error) {
	fname, err := exec.LookPath("deno")
	if err != nil {
		return "", err
	}
	fname, err = filepath.Abs(fname)
	if err != nil {
		return "", err
	}
	fmt.Println("deno abs path:", fname)
	return fname, nil
}
