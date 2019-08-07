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
		Status:    statusStarting,
		statusSub: make(map[statusInt][]chan statusInt)}

	s.setStatus(statusReady)

	s.waitFor(statusReady)
}

func TestStatusWait(t *testing.T) {
	s := &Sandbox{
		Status:    statusStarting,
		statusSub: make(map[statusInt][]chan statusInt)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.setStatus(statusReady)
	}()

	s.waitFor(statusReady)
}

func TestStatusWaitSkip(t *testing.T) {
	s := &Sandbox{
		Status:    statusStarting,
		statusSub: make(map[statusInt][]chan statusInt)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.setStatus(statusKilling)
	}()

	s.waitFor(statusReady)
}

func TestStatusNotReached(t *testing.T) {
	s := &Sandbox{
		Status:    statusStarting,
		statusSub: make(map[statusInt][]chan statusInt)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.setStatus(statusReady)
	}()

	go func() {
		s.waitFor(statusKilling)
		t.Error("should not have triggered this status")
	}()

	time.Sleep(200 * time.Millisecond)
}

func TestStatusWaitMultiple(t *testing.T) {
	s := &Sandbox{
		Status:    statusStarting,
		statusSub: make(map[statusInt][]chan statusInt)}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.setStatus(statusKilling)
	}()

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			s.waitFor(statusReady)
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		s.waitFor(statusKilling)
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
		SandboxID: 7,
		Status:    statusStarting,
		LogClient: logger,
		Config:    cfg}

	logger.EXPECT().Log(domain.INFO, nil, gomock.Any())

	s.start()

	// OK, shut it down

	s.Stop()
}

// This is really testing the whole thing, including the node side runtime.

func getJSRuntimePath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}

	jsRuntime := path.Join(dir, "../../../install/files/ds-sandbox-runner.js")

	return jsRuntime
}
