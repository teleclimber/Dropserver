package sandbox

// Keep this module but make significant changes to
// have it just manage deno processes?

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
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
	SandboxID       int
	Status          string
	Address         string // may not be necessary, just need port I think
	appspace        *domain.Appspace
	cmd             *exec.Cmd
	reverseListener *reverseListener
	statusSub       map[string][]chan bool
	Transport       http.RoundTripper
	appSpaceSession appSpaceSession
	killScore       float64 // rename. It's killScore perhaps
	Config          *domain.RuntimeConfig
	LogClient       domain.LogCLientI
}

// Should start() return a channel or something?
// or should callers just do go start()?
func (s *Sandbox) start() { // return an error, presumably?
	s.LogClient.Log(domain.INFO, nil, "Starting sandbox")

	// Here start should take necessary data about appspace
	// ..in order to pass in the right permissions to deno.
	// I think here we don't need to return?
	// just hold on until the process ends.
	// Instead push status into a channel or something?

	//cmd := exec.Command("node", "/root/ds-sandbox-runner.js", hostIP, rev_sock_path)
	// ..Will have to pass location of script somehow.
	// ..In prod it's relative to install dir, in testing it's....?

	s.reverseListener = newReverseListener(s.Config, s.SandboxID, s.onReverseMsg)

	cmd := exec.Command("node", s.Config.Exec.JSRunnerPath, s.reverseListener.socketPath)
	s.cmd = cmd
	// -> for Deno will have to pass permission flags for that sandbox.
	// The appspace is known at this point and should probably be passed to the runner.
	// the runner JS location is specified in some sort of runtime config
	// Note that ultimately we need to stick this in a Cgroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start() // returns right away
	if err != nil {
		log.Fatal(err)
	}

	go s.monitor(stdout, stderr)

	// we should also start reverse listener and wait for it to give the OK?
	// We need config to know where the unix sockets reside.
	// Do we pass config struct all the way to each rev listener? -> probably most consistent.
	//s.reverseListener = newReverseListener(s.Config, s.SandboxID, s.onReverseMsg)

	s.reverseListener.waitFor("hi") // wait this will block?? <- yes, yes it will

	// TODO: after "hi", we have to put status at "ready", but we need a status setter/trigger

}

// monitor waits for cmd to end
// It also collects output for use somewhere.
func (s *Sandbox) monitor(stdout io.ReadCloser, stderr io.ReadCloser) {
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

	wg.Wait()

	err := s.cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	s.Status = "dead"
}

func printLogs(r io.ReadCloser) {
	buf := make([]byte, 80)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			//logfn(buf[0:n])
			//println(string(buf[0:n]))
			fmt.Printf("%s", buf[0:n])
		}
		if err != nil {
			break
		}
	}
}

// Stop stops the container and its associated open connections
func (s *Sandbox) Stop(wg *sync.WaitGroup) {
	defer wg.Done()

	// reverse listener...

	// get state and then send kill signal?
	// Then loop over and check pid.
	// -> I think ds-sandbox-d had this nailed down.
	// update status and clean up after?

	// if sandbox status is killed, then do nothing, the killing system is working.

	// get status from pid, if running, send kill sig, then wait some.
	// follow up as with ds-sandbox-d

	s.Status = "killing"

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
	/////.....

	// after you kill, whether successful or not,
	// sandbox manager ought to remove the sandbox from sandboxes.
	// If had to forcekill then quarantine the

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

// GetAddress gets the IP address of the sandbox
func (s *Sandbox) GetAddress() string {
	return s.Address
}

// GetTransport gets the http transport of the sandbox
func (s *Sandbox) GetTransport() http.RoundTripper {
	return s.Transport
}

// GetLogClient retuns the Logging client
func (s *Sandbox) GetLogClient() domain.LogCLientI {
	return s.LogClient
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

func (s *Sandbox) waitFor(status string) {
	if s.Status == status {
		return
	}
	fmt.Println(s.SandboxID, "waiting for container status", status)

	if _, ok := s.statusSub[status]; !ok {
		s.statusSub[status] = []chan bool{}
	}
	statusMet := make(chan bool)
	s.statusSub[status] = append(s.statusSub[status], statusMet)
	<-statusMet
	delete(s.statusSub, status)
}

// The whole status system needs tightening up.
// we need to lock statusSubs
// ..and we need to have a setter for status that can trigger subs

// func (s *Sandbox) waitForDone(status string) { // I don't get what is the difference between this and waitFor?
// 	if subs, ok := s.statusSub[status]; ok {
// 		for _, wCh := range subs {
// 			wCh <- true
// 		}
// 		s.statusSub[status] = []chan bool{}
// 	}
// 	// then gotta empty / reset the channel.
// 	// though probably lock the array?
// }

func (s *Sandbox) onReverseMsg(msg string) {
	fmt.Println("onReverseMsg", msg, s.SandboxID)
}
