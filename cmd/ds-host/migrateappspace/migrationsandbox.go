package migrateappspace

//go:generate mockgen -destination=mocks.go -package=migrateappspace github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace MigrationSandobxMgrI,MigrationSandboxI
// ^^ remember to add new interfaces to list of interfaces to mock ^^

// What we still need:
// - get updates from sandbox
// - logs from sandbox
// - limit resource usage limits
// - some level of appspace api on host (yes, DB!!)

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"golang.org/x/sys/unix"
)

// MigrationSandobxMgrI makes it possible ot test migrations without and actual sandbox
type MigrationSandobxMgrI interface {
	CreateSandbox() MigrationSandboxI
}

// MigrationSandboxMgr is in charge of creating migration sandboxes.
type MigrationSandboxMgr struct {
	Config *domain.RuntimeConfig
	Logger domain.LogCLientI
}

// CreateSandbox creates a sandbox struct and returns it
func (m *MigrationSandboxMgr) CreateSandbox() *migrationSandbox {
	sandbox := migrationSandbox{
		Config: m.Config,
		Logger: m.Logger}

	return &sandbox
}

// MigrationSandboxI is an interface for the migration sandbox.
type MigrationSandboxI interface {
	Start(string, string, int, int) // TODO: return an error, presumably?
	Stop()
}

// migrationSandbox holds the data necessary to interact with the container
// --> does it need to be exported?
type migrationSandbox struct {
	cmd *exec.Cmd
	//reverseListener *reverseListener // commented out because not present in package, but likely we need it

	//statusSub chan migrationSandboxStatus	// no status, just run and return

	Config *domain.RuntimeConfig
	Logger domain.LogCLientI
}

// Start Should start() return a channel or something?
// or should callers just do go start()?
func (s *migrationSandbox) Start(appLocation string, appspaceLocation string, from int, to int) domain.Error {
	s.Logger.Log(domain.INFO, nil, "Starting sandbox")

	// Will need reverse listener:
	//var dsErr domain.Error
	// s.reverseListener, dsErr = newReverseListener(s.Config, s.id)
	// if dsErr != nil {
	// 	// just stop right here.
	// 	// TODO return that error to caller
	// 	return
	// }

	cmd := exec.Command(
		"node",
		s.Config.Exec.MigratorScriptPath,
		//s.reverseListener.socketPath,
		filepath.Join(s.Config.DataDir, "apps", appLocation), // TODO: This could lead to errors. Make apps dir a runtime config exec field?
		filepath.Join(s.Config.DataDir, "appspaces", appspaceLocation),
		strconv.Itoa(from),
		strconv.Itoa(to))
	s.cmd = cmd
	// -> for Deno will have to pass permission flags for that sandbox.
	// The appspace is known at this point and should probably be passed to the runner.
	// the runner JS location is specified in some sort of runtime config
	// Note that ultimately we need to stick this in a Cgroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errStr := "could not get stdout pipe " + err.Error()
		s.Logger.Log(domain.ERROR, nil, "migrationsandbox: "+errStr)
		return dserror.New(dserror.SandboxFailedStart, errStr)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		errStr := "could not get stderr pipe " + err.Error()
		s.Logger.Log(domain.ERROR, nil, "migrationsandbox: "+errStr)
		return dserror.New(dserror.SandboxFailedStart, errStr)
	}

	err = cmd.Start() // returns right away
	if err != nil {
		errStr := "error on cmd.Start " + err.Error()
		s.Logger.Log(domain.ERROR, nil, "migrationsandbox: "+errStr)
		return dserror.New(dserror.SandboxFailedStart, errStr)
	}

	return s.monitor(stdout, stderr)

	// since it's run and done, and we're supposedly going to get updates via reverse server
	// we don't need toreturn until we are done.

}

// monitor waits for cmd to end or an error gets sent
// It also collects output for use somewhere.
func (s *migrationSandbox) monitor(stdout io.ReadCloser, stderr io.ReadCloser) domain.Error {

	// go func() {
	// 	for { // you need to be in a loop to keep the channel "flowing"
	// 		dsErr := <-s.reverseListener.errorChan
	// 		if dsErr != nil {
	// 			s.Logger.Log(domain.WARN, nil, "Shutting sandbox down because of error on reverse listener")
	// 			s.Stop()
	// 		} else {
	// 			break // errorChan was closed, so exit loop
	// 		}
	// 	}
	// }()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		printLogs(stdout)
		fmt.Println("done printing stdout")
	}()

	go func() {
		defer wg.Done()
		printLogs(stderr)
	}()

	wg.Wait()

	err := s.cmd.Wait()
	if err != nil {
		errStr := "sandbox finished with error: " + err.Error()
		s.Logger.Log(domain.ERROR, nil, "migrationsandbox: "+errStr)
		return dserror.New(dserror.SandboxFailedStart, errStr)
	}

	return nil
}

func printLogs(r io.ReadCloser) {
	buf := make([]byte, 80)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			fmt.Printf("%s", buf[0:n]) //TODO: this should go to a log collection thing
		}
		if err != nil {
			break
		}
	}
}

// Stop stops the sandbox and its associated open connections
// Not clear we really need this, at least not as implemented
// We just need a kill for when it's misbehaving
func (s *migrationSandbox) Stop() {
	// reverse listener...

	// get state and then send kill signal?
	// Then loop over and check pid.
	// -> I think ds-sandbox-d had this nailed down.
	// update status and clean up after?

	// if sandbox status is killed, then do nothing, the killing system is working.

	// get status from pid, if running, send kill sig, then wait some.
	// follow up as with ds-sandbox-d

	// how do we avoid getting into a dysfunction acondition with Proxy?
	// Proxy should probably lock the mutex when it creates a task,
	// ..so that task count can be considered acurate in sandox manager's killing function

	//s.SetStatus(domain.SandboxKilling)

	err := s.kill(false)
	if err != nil {
		s.Logger.Log(domain.ERROR, nil, "Unable to kill sandbox")
		// force kill
		err = s.kill(true)
		if err != nil {
			// ???
			s.Logger.Log(domain.ERROR, nil, "Unable to FORCE kill sandbox")
		}
	}
	/////.....

	//s.reverseListener.close() // maybe wait until "dead"?
}

func (s *migrationSandbox) pidAlive() bool {
	process := s.cmd.Process
	// what does process look like before the cmd is started? Check for nil?
	// what does proces look like after the underlying process has dies?

	err := process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return false
}

// kill sandbox, which means send it the kill sig
// This should get picked up internally and it should shut itself down.
func (s *migrationSandbox) kill(force bool) domain.Error {
	process := s.cmd.Process

	sig := unix.SIGTERM
	if force {
		sig = unix.SIGKILL
	}
	err := process.Signal(sig)
	if err != nil {
		s.Logger.Log(domain.INFO, nil, fmt.Sprintf("kill: Error killing process. Force: %t", force))
	}

	var alive bool
	ms := 5
	if force {
		ms = 50
	}
	for i := 1; i < 21; i++ {
		time.Sleep(time.Duration(ms) * time.Millisecond)

		alive = s.pidAlive()
		if !alive {
			break
		}
	}

	if alive {
		return dserror.New(dserror.SandboxFailedToTerminate)
	}
	return nil
}
