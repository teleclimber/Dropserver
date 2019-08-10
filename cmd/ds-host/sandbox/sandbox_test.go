package sandbox

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// First test the status subscription system
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

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger := domain.NewMockLogCLientI(mockCtrl)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = dir
	cfg.Exec.JSRunnerPath = getJSRuntimePath()

	s := &Sandbox{
		id:        7,
		status:    domain.SandboxStarting,
		LogClient: logger,
		Config:    cfg}

	logger.EXPECT().Log(domain.INFO, nil, gomock.Any())

	appVersion := &domain.AppVersion{}
	appspace := &domain.Appspace{}

	s.Start(appVersion, appspace)

	// OK, shut it down

	s.Stop()
}

func getJSRuntimePath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}

	jsRuntime := path.Join(dir, "../../../install/files/ds-sandbox-runner.js")

	return jsRuntime
}
