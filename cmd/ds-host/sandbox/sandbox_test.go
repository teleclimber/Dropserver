package sandbox

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestImportMaps(t *testing.T) {
	s := &Sandbox{
		appspace: &domain.Appspace{
			LocationKey: "as-loc-13",
			AppspaceID:  domain.AppspaceID(13)},
		appVersion: &domain.AppVersion{
			LocationKey: "av-loc-77"},
		Config: &domain.RuntimeConfig{}}

	s.Config.Exec.AppspacesPath = "/temp/as-path"
	s.Config.Exec.AppsPath = "/temp/apps-path"
	s.Config.Exec.SandboxCodePath = "/temp/sandbox-code-path"

	b, err := s.makeImportMap()
	if err != nil {
		t.Error(err)
	}

	str := string(*b)
	t.Log(str)
	if !strings.Contains(str, "/av-loc-77/\"") {
		t.Error("expected path with trailing slash")
	}
}

// Ttest the status subscription system
func TestStatus(t *testing.T) {
	s := &Sandbox{
		status:    domain.SandboxStarting,
		statusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	s.SetStatus(domain.SandboxReady)

	s.WaitFor(domain.SandboxReady)
}

func TestStatusWait(t *testing.T) {
	s := &Sandbox{
		status:    domain.SandboxStarting,
		statusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.SetStatus(domain.SandboxReady)
	}()

	s.WaitFor(domain.SandboxReady)
}

func TestStatusWaitSkip(t *testing.T) {
	s := &Sandbox{
		status:    domain.SandboxStarting,
		statusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.SetStatus(domain.SandboxKilling)
	}()

	s.WaitFor(domain.SandboxReady)
}

func TestStatusNotReached(t *testing.T) {
	s := &Sandbox{
		status:    domain.SandboxStarting,
		statusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.SetStatus(domain.SandboxReady)
	}()

	go func() {
		s.WaitFor(domain.SandboxKilling)
		t.Error("should not have triggered this status")
	}()

	time.Sleep(200 * time.Millisecond)
}

func TestStatusWaitMultiple(t *testing.T) {
	s := &Sandbox{
		status:    domain.SandboxStarting,
		statusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.SetStatus(domain.SandboxKilling)
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

// func TestStatusSubRemoval(t *testing.T) {

// }

// test blocking channel?
// is that situation even possible?

// func TestCWD(t *testing.T) {
// 	_, caller, _, _ := runtime.Caller(0) // see https://stackoverflow.com/questions/23847003/golang-tests-and-working-directory
// 	fmt.Println("Test caller", caller)

// 	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Println("CWD:", dir)

// 	// let's try to touch the ds-runtime JS:
// 	jsRuntime := path.Join(dir, "../../../install/files/ds-sandbox-runner.js")
// 	_, err = os.Open(jsRuntime)
// 	if os.IsNotExist(err) {
// 		fmt.Println("got it wrong", jsRuntime)
// 	}
// 	if err != nil {
// 		fmt.Println("error:", err)
// 	}
// 	if err == nil {
// 		fmt.Println("looks good")
// 	}

// 	t.Fail()
// }

func TestRunnerScriptError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	scriptPath := path.Join(dir, "foobar.ts")

	err = ioutil.WriteFile(scriptPath, []byte("setTimeout(hello.world, 100);"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir
	cfg.Exec.SandboxRunnerPath = scriptPath
	cfg.Exec.AppspacesPath = dir

	s := &Sandbox{
		id:        7,
		status:    domain.SandboxStarting,
		statusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus),
		Config:    cfg}

	appVersion := &domain.AppVersion{}
	appspace := &domain.Appspace{}

	err = s.Start(appVersion, appspace)
	if err == nil {
		t.Error("expected error from sandbox")
	}

	s.WaitFor(domain.SandboxReady)

	if s.Status() == domain.SandboxReady {
		t.Error("sandbox status should be killing or dead")
	}

	s.Stop()
}

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir
	cfg.Exec.SandboxCodePath = getSandboxCodePath()
	cfg.Exec.SandboxRunnerPath = getSandboxRunnerPath()
	cfg.Exec.AppspacesPath = dir
	cfg.Exec.AppsPath = dir

	appVersion := &domain.AppVersion{
		LocationKey: "app-loc"}
	appspace := &domain.Appspace{
		AppspaceID:  domain.AppspaceID(13),
		LocationKey: "appspace-loc"}

	s := &Sandbox{
		id:         7,
		appspace:   appspace,
		appVersion: appVersion,
		status:     domain.SandboxStarting,
		Config:     cfg}

	err = s.Start(appVersion, appspace)
	if err != nil {
		s.Stop()
		t.Error(err)
	}

	if s.Status() != domain.SandboxReady {
		t.Error("sandbox status should be ready")
	}

	s.Stop()
}

func TestExecFn(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	appDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(appDir)

	appLocation := "app-loc"
	err = os.MkdirAll(path.Join(appDir, appLocation), 0755)
	if err != nil {
		t.Error(err)
	}

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir
	cfg.Exec.SandboxCodePath = getSandboxCodePath()
	cfg.Exec.SandboxRunnerPath = getSandboxRunnerPath()
	cfg.Exec.AppspacesPath = dir
	cfg.Exec.AppsPath = appDir

	appVersion := &domain.AppVersion{
		LocationKey: appLocation}
	appspace := &domain.Appspace{
		AppspaceID:  domain.AppspaceID(13),
		LocationKey: "appspace-loc"}

	scriptPath := path.Join(appDir, appLocation, "app.ts")

	err = ioutil.WriteFile(scriptPath, []byte("export function abc() { console.log('hello workd'); }"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	s := &Sandbox{
		id:         7,
		appspace:   appspace,
		appVersion: appVersion,
		status:     domain.SandboxStarting,
		Config:     cfg}

	err = s.Start(appVersion, appspace)
	if err != nil {
		s.Stop()
		t.Error(err)
	}

	if s.Status() != domain.SandboxReady {
		t.Error("sandbox status should be ready")
	}

	err = s.ExecFn(domain.AppspaceRouteHandler{
		File:     "@app/app.ts",
		Function: "abc",
	})
	if err != nil {
		t.Error(err)
	}

	s.Stop()
}

func TestExecForbiddenImport(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	forbiddenDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(forbiddenDir)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir
	cfg.Exec.SandboxCodePath = getSandboxCodePath()
	cfg.Exec.SandboxRunnerPath = getSandboxRunnerPath()
	cfg.Exec.AppspacesPath = dir
	cfg.Exec.AppsPath = dir

	appVersion := &domain.AppVersion{
		LocationKey: "app-loc"}
	appspace := &domain.Appspace{
		AppspaceID:  domain.AppspaceID(13),
		LocationKey: "appspace-loc"}

	scriptPath := path.Join(forbiddenDir, "bad.ts")

	err = ioutil.WriteFile(scriptPath, []byte("export function abc() { console.log('hello bad'); }"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	s := &Sandbox{
		id:         7,
		appspace:   appspace,
		appVersion: appVersion,
		status:     domain.SandboxStarting,
		Config:     cfg}

	err = s.Start(appVersion, appspace)
	if err != nil {
		s.Stop()
		t.Error(err)
	}

	if s.Status() != domain.SandboxReady {
		t.Error("sandbox status should be ready")
	}

	err = s.ExecFn(domain.AppspaceRouteHandler{
		File:     scriptPath,
		Function: "abc",
	})
	if err == nil {
		t.Error("Expected an error")
	}

	s.Stop()
}

func getSandboxCodePath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, "../../../resources/")
}
func getSandboxRunnerPath() string {
	return filepath.Join(getSandboxCodePath(), "ds-sandbox-runner.ts")
}
