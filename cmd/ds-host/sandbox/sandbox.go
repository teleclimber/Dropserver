package sandbox

// Keep this module but make significant changes to
// have it just manage deno processes?

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/twine"
	"golang.org/x/sys/unix"
)

// OK so this is going to be significantly different.
// ..we may even need to delete entirely and create a fresh new module.
// - create sandbox on-demand on incoming request (this is a limitation from Deno, for now)
// - need to detect when it crashes/hangs
// - need to be able to send a die message
// - need to be able to kill if misbehaving
// - We need a appspace-api server on host for all api requests that emanate from appspaces.

type appSpaceSession struct {
	tasks      []*Task
	lastActive time.Time
	tiedUp     bool
}

// Task tracks the container being tied up for one request
type Task struct {
	Finished bool //build up with start time, elapsed and any other metadata
}

// Sandbox holds the data necessary to interact with the container
type Sandbox struct {
	id              int                  // getter only (const), unexported
	status          domain.SandboxStatus // getter/setter, so make it unexported.
	socketsDir      string
	cmd             *exec.Cmd
	twine           *twine.Twine
	services        *domain.ReverseServices
	statusMux       sync.Mutex
	statusSub       map[domain.SandboxStatus][]chan domain.SandboxStatus
	transport       http.RoundTripper
	appSpaceSession appSpaceSession // put a getter for that?
	killScore       float64         // this should not be here.
	Config          *domain.RuntimeConfig
	LogClient       domain.LogCLientI
}

// Start Should start() return a channel or something?
// or should callers just do go start()?
func (s *Sandbox) Start(appVersion *domain.AppVersion, appspace *domain.Appspace) { // TODO: return an error, presumably?
	s.LogClient.Log(domain.INFO, nil, "Starting sandbox")

	socketsDir, dsErr := makeSocketsDir(s.Config.Sandbox.SocketsDir, appspace.AppspaceID)
	if dsErr != nil {
		// just stop right here.
		// TODO return that error to caller
		return
	}
	s.socketsDir = socketsDir

	//fwdSock := path.Join(socketsDir, "fwd.sock") // forward socket is where the ds runner will create the sandbox server

	// Here start should take necessary data about appspace
	// ..in order to pass in the right permissions to deno.

	twineServer, err := twine.NewServer(path.Join(socketsDir, "rev.sock"))
	if err != nil {
		// handle this p
		return
	}
	s.twine = twineServer

	cmd := exec.Command(
		"deno",
		"run",
		"--unstable",    // needed for unix domain sockets
		"--allow-read",  // TODO app dir and appspace dir, and sockets
		"--allow-write", // TODO appspace dir, and fwd socket specifically? actually you have to be able to write to rev too!
		s.Config.Exec.JSRunnerPath,
		s.socketsDir,
		filepath.Join(s.Config.Exec.AppsPath, appVersion.LocationKey),
		filepath.Join(s.Config.Exec.AppspacesFilesPath, appspace.LocationKey),
	)
	s.cmd = cmd
	// Note that ultimately we need to stick this in a Cgroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		//log.Fatal(err)
		// TODO return error
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		//log.Fatal(err)
		// TODO return error
		return
	}

	err = cmd.Start() // returns right away
	if err != nil {
		//log.Fatal(err)
		// TODO return Error
		return
	}

	go s.monitor(stdout, stderr)

	_, ok := <-s.twine.ReadyChan
	if !ok {
		// failstart
		// handle and return.
		s.Stop()
		return
	}

	s.transport = http.DefaultTransport // really not sure what this means or what it's for anymore....
	s.SetStatus(domain.SandboxReady)
}

// monitor waits for cmd to end or an error gets sent
// It also collects output for use somewhere.
func (s *Sandbox) monitor(stdout io.ReadCloser, stderr io.ReadCloser) {

	go func() {
		for { // you need to be in a loop to keep the channel "flowing"
			err := <-s.twine.ErrorChan
			if err != nil {
				s.LogClient.Log(domain.WARN, nil, "Shutting sandbox down because of error on reverse listener")
				s.Stop()
			} else {
				break // errorChan was closed, so exit loop
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		printLogs(stdout)
	}()

	go func() {
		defer wg.Done()
		printLogs(stderr)
	}()

	//wg.Wait()

	err := s.cmd.Wait()
	if err != nil {
		log.Println("cmd.Wait returned error", err)
		s.Stop()
		// Here we should kill off reverse listener?
		// This is where we want to log things for the benefit of the dropapp user.
		// Also somehow whoever started the sandbox needs to know it exited with error
	}

	wg.Wait()

	s.SetStatus(domain.SandboxDead)
	// -> it may have crashed.
	// TODO ..should probably call regular shutdown procdure to clean everything up.

	// now kill the reverse channel? Otherwise we risk killing it shile it could still provide valuable data?
}

func printLogs(r io.ReadCloser) {
	buf := make([]byte, 80)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			fmt.Printf("%s", buf[0:n])
		}
		if err != nil {
			break
		}
	}
}

// Stop stops the sandbox and its associated open connections
func (s *Sandbox) Stop() {
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

	s.SetStatus(domain.SandboxKilling)

	// TODO send a command on sandboxService to shutdown.

	err := s.kill(false)
	if err != nil {
		s.LogClient.Log(domain.ERROR, nil, "Unable to kill sandbox")
		// force kill
		err = s.kill(true)
		if err != nil {
			// ???
			s.LogClient.Log(domain.ERROR, nil, "Unable to FORCE kill sandbox")
		}
	}

	s.twine.Stop()
	// the shutdown command sent to via twine should automatically call graceful shutdown on this side.
	// But still ned to handle the case that the Twine server never got going because it never got "hi" because client died prior to sending it.

	// after you kill, whether successful or not,
	// sandbox manager ought to remove the sandbox from sandboxes.
	// If had to forcekill then quarantine the

	// Not sure if we can do this now, or have to wait til dead...
	cleanSocketsDir(s.socketsDir)
	// ^^ capture error and log or somesuch.

}

func (s *Sandbox) pidAlive() bool {
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
func (s *Sandbox) kill(force bool) domain.Error {
	process := s.cmd.Process

	sig := unix.SIGTERM
	if force {
		sig = unix.SIGKILL
	}
	err := process.Signal(sig)
	if err != nil {
		s.LogClient.Log(domain.INFO, nil, fmt.Sprintf("kill: Error killing process. Force: %t", force))
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

// basic getters

// ID returns the ID of the sandbox
func (s *Sandbox) ID() int {
	return s.id
}

// Status returns the status of the Sandbox
func (s *Sandbox) Status() domain.SandboxStatus {
	s.statusMux.Lock()
	defer s.statusMux.Unlock()
	return s.status
}

// GetTransport gets the http transport of the sandbox
func (s *Sandbox) GetTransport() http.RoundTripper {
	return s.transport
}

// GetLogClient retuns the Logging client
func (s *Sandbox) GetLogClient() domain.LogCLientI {
	return s.LogClient
}

// TiedUp returns the appspaceSession
func (s *Sandbox) TiedUp() bool {
	return s.appSpaceSession.tiedUp
}

// LastActive returns the last used time
func (s *Sandbox) LastActive() time.Time {
	return s.appSpaceSession.lastActive
}

// TaskBegin adds a new task to session tasks and returns it
func (s *Sandbox) TaskBegin() chan bool {
	reqTask := Task{}
	s.appSpaceSession.tasks = append(s.appSpaceSession.tasks, &reqTask)
	s.appSpaceSession.lastActive = time.Now()
	s.appSpaceSession.tiedUp = true

	ch := make(chan bool)

	// go func here that blocks on chanel.
	go func() {
		<-ch
		reqTask.Finished = true
		s.appSpaceSession.lastActive = time.Now()
		s.appSpaceSession.tiedUp = s.isTiedUp()
	}()

	return ch //instead of returning a task we should return a chanel.
}

func (s *Sandbox) isTiedUp() (tiedUp bool) {
	for _, task := range s.appSpaceSession.tasks {
		if !task.Finished {
			tiedUp = true
			break
		}
	}
	return
}

// SetStatus sets the status
func (s *Sandbox) SetStatus(status domain.SandboxStatus) {
	s.statusMux.Lock()
	defer s.statusMux.Unlock()

	if status > s.status {
		s.status = status
		for stat, subs := range s.statusSub {
			if stat <= s.status {
				for _, sub := range subs {
					sub <- s.status // this might block if nobody is actually waiting on the channel?
				}
				delete(s.statusSub, stat)
			}
		}
	}
}

// WaitFor blocks until status is met, or greater status is met
func (s *Sandbox) WaitFor(status domain.SandboxStatus) {
	s.statusMux.Lock()
	if s.status >= status {
		s.statusMux.Unlock()
		return
	}
	fmt.Println(s.id, "waiting for sandbox status", status)

	if _, ok := s.statusSub[status]; !ok {
		s.statusSub[status] = []chan domain.SandboxStatus{}
	}
	statusMet := make(chan domain.SandboxStatus)
	s.statusSub[status] = append(s.statusSub[status], statusMet)

	s.statusMux.Unlock()

	<-statusMet
}

/////////////
func cleanSocketsDir(sockDir string) domain.Error {
	if err := os.RemoveAll(sockDir); err != nil {
		return dserror.FromStandard(err)
	}
	return nil
}
func makeSocketsDir(baseDir string, appspaceID domain.AppspaceID) (string, domain.Error) {
	sockDir := getSocketsDir(baseDir, appspaceID)

	dsErr := cleanSocketsDir(sockDir)
	if dsErr != nil {
		return "", dsErr
	}

	if err := os.MkdirAll(sockDir, 0700); err != nil {
		return "", dserror.FromStandard(err)
	}

	return sockDir, nil
}

func getSocketsDir(baseDir string, appspaceID domain.AppspaceID) string {
	return path.Join(baseDir, fmt.Sprintf("as-%d", appspaceID))
}
