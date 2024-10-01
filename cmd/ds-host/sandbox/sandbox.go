package sandbox

// Keep this module but make significant changes to
// have it just manage deno processes?

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
	"golang.org/x/sys/unix"
)

// killDelay delay before killing sandbox to allow it to return errors
const killDelay = 500 * time.Millisecond

// Temporary hard coded memory.high values for individual sandboxes:
const sbStartMb = 400
const sbRunMb = 128

type taskTracker struct { // expand on this to track total tied up time
	mux        sync.Mutex
	numTask    int // number of tasks created that have not ended
	numActive  int // active tasks have started but not ended
	lastActive time.Time
	cumul      time.Duration
	clumpStart time.Time
	clumpEnd   time.Time
	notifyCh   chan struct{}
}

func (t *taskTracker) newTask() chan struct{} {
	ch := make(chan struct{})
	start := time.Time{}
	end := time.Time{}

	go func() {
		for range ch {
			t.mux.Lock()
			if start.IsZero() {
				start = time.Now()
				t.numActive++
				if t.notifyCh != nil {
					t.notifyCh <- struct{}{}
				}
			}
			if t.clumpStart.IsZero() {
				t.clumpStart = start // set the beginning of a clump if there is no clump.
			}
			t.mux.Unlock()
		}

		// When channel is closed, means task ended
		t.mux.Lock()
		defer t.mux.Unlock()
		end = time.Now()
		t.numTask--
		t.numActive--
		// Set clump end, but only if start was actually set
		if !start.IsZero() && (t.clumpEnd.IsZero() || end.After(t.clumpEnd)) {
			t.clumpEnd = end
		}
		// If there are zero active tasks, calculate cumul time and zero out clumps.
		if t.numActive == 0 {
			t.cumul += t.clumpEnd.Sub(t.clumpStart)
			t.lastActive = t.clumpEnd
			t.clumpStart = time.Time{}
			t.clumpEnd = time.Time{}
		}
		if t.notifyCh != nil {
			t.notifyCh <- struct{}{}
		}
	}()

	t.mux.Lock()
	defer t.mux.Unlock()
	t.numTask++

	return ch
}

func (t *taskTracker) isTiedUp() bool {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.numTask != 0
}

func (t *taskTracker) getCumulDuration() time.Duration {
	t.mux.Lock()
	defer t.mux.Unlock()
	c := t.cumul
	if !t.clumpStart.IsZero() {
		// There are some active tasks, calculate cumul from clump start to current time.
		c += time.Since(t.clumpStart)
	}
	return c
}

type runDBIDData struct {
	id int
	ok bool
}

type BwrapStatusJsonI interface {
	GetFile() *os.File
	WaitPid() (int, bool)
}

// local sandbox service:
const sandboxService = 11

// remote exec fn service
const executeService = 12
const execFnCommand = 11

// Sandbox holds the data necessary to interact with the container
type Sandbox struct {
	id          int //local id
	operation   string
	ownerID     domain.UserID
	appVersion  *domain.AppVersion
	appspace    *domain.Appspace
	SandboxRuns interface {
		Create(run domain.SandboxRunIDs, start time.Time) (int, error)
		End(int, time.Time, domain.SandboxRunData) error
	}
	CGroups interface {
		CreateCGroup(domain.CGroupLimits) (string, error)
		AddPid(string, int) error
		SetLimits(string, domain.CGroupLimits) error
		GetMetrics(string) (domain.CGroupData, error)
		RemoveCGroup(string) error
	}
	Logger   interface{ Log(string, string) }
	cmd      *exec.Cmd
	twine    *twine.Twine
	Services domain.ReverseServiceI
	outProxy interface {
		Port() int
		Stop()
	}
	outProxyMITM     bool // not used yet
	statusMux        sync.Mutex
	status           domain.SandboxStatus
	statusSub        []chan domain.SandboxStatus
	waitStatusSub    map[domain.SandboxStatus][]chan domain.SandboxStatus
	transport        http.RoundTripper
	taskTracker      taskTracker
	inspect          bool
	AppLocation2Path interface {
		Meta(string) string
		Files(string) string
		DenoDir(string) string
	}
	AppspaceLocation2Path interface {
		Base(string) string
		Data(string) string
		Files(string) string
		Avatars(string) string
		DenoDir(string) string
	}
	Config *domain.RuntimeConfig

	paths  *paths
	cGroup string
}

func NewSandbox(id int, operation string, ownerID domain.UserID, appVersion *domain.AppVersion, appspace *domain.Appspace) *Sandbox {
	s := &Sandbox{
		id:            id,
		ownerID:       ownerID,
		operation:     operation,
		appVersion:    appVersion,
		appspace:      appspace,
		status:        domain.SandboxPrepared,
		statusSub:     make([]chan domain.SandboxStatus, 0),
		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus),
		paths:         &paths{}}

	return s
}

func (s *Sandbox) OwnerID() domain.UserID {
	return s.ownerID
}
func (s *Sandbox) AppspaceID() (a domain.NullAppspaceID) {
	if s.appspace != nil {
		a.Set(s.appspace.AppspaceID)
	}
	return
}
func (s *Sandbox) AppVersion() *domain.AppVersion {
	return s.appVersion
}
func (s *Sandbox) Operation() string {
	return s.operation
}

// SetInspect sets the inspect flag which will cause the sandbox to start with --inspect-brk
func (s *Sandbox) SetInspect(inspect bool) {
	s.inspect = inspect
}

// SetImportMap sets items to include in the generated import map
func (s *Sandbox) SetImportMapExtras(extras map[string]string) {
	s.paths.importMapExtras = extras
}

// Start sets the status to starting
// and begins the start process in a goroutine.
func (s *Sandbox) Start() {
	s.setStatus(domain.SandboxStarting)
	go func() {
		err := s.doStart()
		if err != nil {
			s.getLogger("doStart Error").Error(err)
			s.Kill()
			return
		}
	}()
}

// envkv stores the key and value pairs for environment variables
type envkv struct{ key, val string }

// forCmd returns a string suitable for exec.Command.Env array
func (e *envkv) forCmd() string {
	return fmt.Sprintf("%s=%s", e.key, e.val)
}

func (s *Sandbox) doStart() error {
	logger := s.getLogger("doStart()").AddNote(fmt.Sprintf("Sandbox %v", s.id))
	logger.Debug("starting...")

	s.setStatus(domain.SandboxStarting) // should already be set, but just in case

	// Mark beginning of start here
	tStart := time.Now()
	tRef := time.Now()

	memHighMb := sbStartMb
	if s.Config.Sandbox.MemoryHighMb < memHighMb {
		memHighMb = s.Config.Sandbox.MemoryHighMb
	}
	limits := domain.CGroupLimits{
		MemoryHigh: memHighMb * 1024 * 1024,
	}

	if s.Config.Sandbox.UseCGroups {
		cGroup, err := s.CGroups.CreateCGroup(limits)
		if err != nil {
			return err
		}
		s.cGroup = cGroup
	}

	tStr := fmt.Sprintf("Create Cgroups: %s", time.Since(tRef))

	logString := "Sandbox starting"
	if s.inspect {
		logString += " with --inspect-brk"
	}
	s.log(logString)

	socketsDir, err := os.MkdirTemp(s.Config.Sandbox.SocketsDir, "sock")
	if err != nil {
		logger.AddNote(fmt.Sprintf("os.MkdirTemp() dir: %v", s.Config.Sandbox.SocketsDir)).Error(err)
		return err
	}
	s.paths.sockets = socketsDir

	s.createPaths()

	err = s.writeImportMap()
	if err != nil {
		return err
	}

	// here write a file that imports the sandbox code and the app code
	// This should be done at install time of the app
	err = s.writeBootstrapFile()
	if err != nil {
		return err
	}

	err = os.MkdirAll(s.paths.hostPath("deno-dir"), 0755)
	if err != nil {
		return err
	}

	tRef = time.Now()

	twineServer, err := twine.NewUnixServer(path.Join(socketsDir, "rev.sock"))
	if err != nil {
		logger.AddNote("twine.NewUnixServer").Error(err)
		return err // maybe return a user-centered error
	}
	s.twine = twineServer

	tStr += fmt.Sprintf(" Start Twine: %s", time.Since(tRef))

	denoEnvs := []envkv{{"NO_COLOR", "true"}}
	denoArgs := []string{"run", "--unstable-kv"}

	if s.inspect {
		denoArgs = append(denoArgs, "--inspect-brk")
	}

	// Deno run by default does not type check. We'd like to type check on app init:
	if s.operation == opAppInit {
		denoArgs = append(denoArgs, "--check=all")
	}

	tRef = time.Now()
	err = s.startOutProxy()
	if err != nil {
		return fmt.Errorf("error starting outproxy: %w", err)
	}
	tStr += fmt.Sprintf(" Start OutProxy: %s", time.Since(tRef))

	denoEnvs = append(denoEnvs,
		envkv{"HTTP_PROXY", fmt.Sprintf("localhost:%d", s.outProxy.Port())},
		envkv{"HTTPS_PROXY", fmt.Sprintf("localhost:%d", s.outProxy.Port())})
	if s.outProxyMITM {
		denoArgs = append(denoArgs, "--cert="+s.paths.sandboxPath("goproxy-cert"))
	}

	// We have to pass an appspace data dir to the sandbox runner even in app-only mode
	// due to a quirk in how we read args there
	appspaceData := "/dev/null"
	if s.appspace != nil {
		appspaceData = s.paths.sandboxPath("appspace-data")
	}

	cwd := ""
	if s.appspace != nil {
		cwd = s.paths.sandboxPath("appspace-files")
	}

	denoArgs = append(denoArgs,
		"--import-map="+s.paths.sandboxPath("import-map"),
		"--allow-read="+s.paths.denoAllowRead(),
		"--allow-write="+s.paths.denoAllowWrite(),
		s.paths.sandboxPath("bootstrap"),
		s.paths.sandboxPath("sockets"),
		s.paths.sandboxPath("app-files"),
		appspaceData)

	var bwrapStatus BwrapStatusJsonI
	if s.Config.Sandbox.UseBubblewrap {
		bwrapStatus, err = NewBwrapJsonStatus(s.Config.Sandbox.SocketsDir)
		if err != nil {
			logger.AddNote("NewBwrapJsonStatus()").Error(err)
			return err // user centered error
		}

		bwrapArgs := []string{"--clearenv",
			"--setenv", "DENO_DIR", s.paths.sandboxPath("deno-dir"),
		}
		for _, e := range denoEnvs {
			bwrapArgs = append(bwrapArgs, "--setenv", e.key, e.val)
		}

		bwrapArgs = append(bwrapArgs,
			//"--unshare-all", "--share-net", // temporary
			"--unshare-user-try",
			"--unshare-ipc",
			//"--unshare-pid",	// can't unshare pid for now
			"--unshare-uts",
			"--unshare-cgroup-try",
			"--proc", "/proc",
			"--die-with-parent",
			"--new-session", // to protect against TIOCSTI
			"--json-status-fd", "3")

		if cwd != "" {
			bwrapArgs = append(bwrapArgs, "--chdir", cwd)
		}
		for _, p := range s.Config.Sandbox.BwrapMapPaths {
			bwrapArgs = append(bwrapArgs, "--ro-bind", p, p)
		}
		bwrapArgs = append(bwrapArgs, s.paths.getBwrapPathMaps()...)
		bwrapArgs = append(bwrapArgs, s.paths.sandboxPath("deno"))
		bwrapArgs = append(bwrapArgs, denoArgs...)

		s.cmd = exec.Command("bwrap", bwrapArgs...)
		s.cmd.ExtraFiles = []*os.File{bwrapStatus.GetFile()}
	} else {
		s.cmd = exec.Command(s.paths.sandboxPath("deno"), denoArgs...)
		s.cmd.Dir = cwd

		cmdEnvs := []string{}
		for _, e := range denoEnvs {
			cmdEnvs = append(cmdEnvs, e.forCmd())
		}
		s.cmd.Env = cmdEnvs
	}

	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		logger.AddNote("cmd.StdoutPipe()").Error(err)
		return err // user centered error
	}

	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		logger.AddNote("cmd.StderrPipe()").Error(err)
		return err
	}

	runDbIDCh := s.createRun()

	tStr += fmt.Sprintf(" Total pre-start: %s", time.Since(tStart))

	err = s.cmd.Start() // returns right away
	if err != nil {
		logger.AddNote("cmd.Start()").Error(err)
		return err
	}

	tRef = time.Now()

	if s.Config.Sandbox.UseCGroups {
		err = s.CGroups.AddPid(s.cGroup, s.cmd.Process.Pid)
		if err != nil {
			return err
		}
		if s.Config.Sandbox.UseBubblewrap {
			pid, ok := bwrapStatus.WaitPid() //TODO I really don't like waiting here. This could delay the sandbox starting up.
			if !ok {
				return errors.New("failed to get pid from bwrap")
			}
			err = s.CGroups.AddPid(s.cGroup, pid)
			if err != nil {
				return err
			}
		}
	}

	tStr += fmt.Sprintf(" Pid to CGroup: %s", time.Since(tRef))

	go s.monitor(stdout, stderr, runDbIDCh)

	_, ok := <-s.twine.ReadyChan
	if !ok {
		logger.Log("Apparent failed start. ReadyChan closed")
		time.Sleep(killDelay) // delay kill to allow error messages to come out
		s.Kill()
		return errors.New("failed to start sandbox")
	}

	go s.listenMessages()

	tStr += fmt.Sprintf(" Total to Twine ready: %s", time.Since(tStart))

	go func() {
		s.WaitFor(domain.SandboxReady)
		tStr += fmt.Sprintf(" Total to Sandbox ready: %s", time.Since(tStart))
		logger.Debug(tStr)

		if s.Config.Sandbox.UseCGroups {
			err = s.CGroups.SetLimits(s.cGroup, domain.CGroupLimits{MemoryHigh: sbRunMb * 1024 * 1024})
			if err != nil {
				logger.AddNote("s.CGroups.SetLimits").Error(err)
			}
		}
	}()

	return nil
}

func (s *Sandbox) startOutProxy() error {
	outProxy := &OutProxy{
		Log: func(logString string) {
			if s.Logger != nil && !reflect.ValueOf(s.Logger).IsNil() {
				go s.Logger.Log(fmt.Sprintf("sandbox-%v-outproxy", s.id), logString)
			}
		},
	}
	s.outProxy = outProxy
	return outProxy.Start(*s.Config, s.outProxyMITM)
}

// monitor waits for cmd to end or an error gets sent
// It also collects output for use somewhere.
func (s *Sandbox) monitor(stdout io.ReadCloser, stderr io.ReadCloser, runDBIDCh chan runDBIDData) {

	go func() {
		for { // you need to be in a loop to keep the channel "flowing"
			err := <-s.twine.ErrorChan
			if err != nil {
				s.getLogger("Twine ErrorChan").Error(err)
				time.Sleep(killDelay) // Delay kill to allow error messages to come out.
				s.Kill()
			} else {
				break // errorChan was closed, so exit loop
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		s.handleLog(stdout, "stdout")
		wg.Done()
	}()

	go func() {
		s.handleLog(stderr, "stderr")
		wg.Done()
	}()

	err := s.cmd.Wait()
	if err != nil {
		l := s.getLogger("monitor(), s.cmd.Wait()")
		if exiterr, ok := err.(*exec.ExitError); ok {
			l.AddNote(fmt.Sprintf("Exit Status: %d", exiterr.ExitCode())).Error(exiterr)
		} else {
			l.AddNote("cmd.Wait").Error(err)
		}
		s.Kill()
		// Here we should kill off reverse listener?
		// This is where we want to log things for the benefit of the dropapp user.
		// Also somehow whoever started the sandbox needs to know it exited with error
	}

	wg.Wait()

	s.setStatus(domain.SandboxDead)

	s.cleanup(runDBIDCh)

	s.setStatus(domain.SandboxCleanedUp)

	s.log("Sandbox terminated")
}

func (s *Sandbox) handleLog(rc io.ReadCloser, source string) {
	logSource := fmt.Sprintf("sandbox-%v-%v", s.id, source)
	buf := make([]byte, 1000)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			logString := string(buf[0:n])
			if s.Logger != nil && !reflect.ValueOf(s.Logger).IsNil() {
				go s.Logger.Log(logSource, logString)
			}
		}
		if err != nil {
			break
		}
	}
}

// Need to change this to be realtime
// And to be processed externally.
// Who are the consumers?
// - a log recorder, writes the messages to a log file somewhere
// - potential realtime log viewers, on the frontend.
// Q: should sandbox logs be independent of other appspace logs?
// web rquests, db logs, other stuff that is bound to get added?
// -> probably.
// ..I think one single text log with leaders: SANDBOX34: ..., so different parts can log to the same log
// For more fine-grained stuff, user a set of realtime events?
// This, as we know, will eventually tie-in to logging for the sake of billing. (or not, really.)

func (s *Sandbox) listenMessages() {
	for message := range s.twine.MessageChan {
		switch message.ServiceID() {
		case sandboxService:
			go s.handleMessage(message)
		default:
			if s.Services != nil {
				go s.Services.HandleMessage(message)
			} else {
				s.getLogger("listenMessages()").Error(fmt.Errorf("sandboxed code trying to access service but no service attached to sandbox, service: %v", message.ServiceID()))
				message.SendError("No services attached to sandbox")
			}
		}
	}
}

// incoming commands on sandboxService
const sandboxReady = 11

func (s *Sandbox) handleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case sandboxReady:
		s.setStatus(domain.SandboxReady)
		m.SendOK()
	default:
		s.getLogger("handleMessage()").Log(fmt.Sprintf("Command not recognized: %v", m.CommandID()))
		m.SendError("command not recognized")
	}
}

// Graceful politely asks the sandbox to shut itself down
func (s *Sandbox) Graceful() {
	s.getLogger("Graceful()").Log(fmt.Sprintf("Sandbox %v starting shutdown", s.id))

	s.setStatus(domain.SandboxKilling)

	go func() {
		reply, err := s.twine.SendBlock(sandboxService, 13, nil)
		if err != nil {
			// ???
			s.getLogger("Graceful() twine.SendBlock()").Error(err)
		} else if !reply.OK() {
			s.getLogger("Graceful() twine.SendBlock()").Log("response not OK")
		}

		// Then tell twine to shut down nicely:
		s.twine.Graceful()

		s.getLogger("Graceful()").Log(fmt.Sprintf("Sandbox %v graceful complete", s.id))
	}()
}

// Kill the sandbox. No mercy.
func (s *Sandbox) Kill() {
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

	s.setStatus(domain.SandboxKilling)

	err := s.kill(false)
	if err != nil {
		// force kill
		err = s.kill(true)
		if err != nil {
			// ???
		}
	}

	// the shutdown command sent to via twine should automatically call graceful shutdown on this side.
	// But still ned to handle the case that the Twine server never got going because it never got "hi" because client died prior to sending it.

	// after you kill, whether successful or not,
	// sandbox manager ought to remove the sandbox from sandboxes.
}

// Cleanup socket paths, and listeners, etc...
// After the sandbox has terminated
func (s *Sandbox) cleanup(runDBIDCh chan runDBIDData) {
	tiedUpDuration := s.taskTracker.getCumulDuration()

	// maybe twine needs a checkConn()? or something?
	// or maybe graceful itself should do it, so it can act accordingly
	//s.twine.Graceful() // not sure this will work because we probably have messages in limbo?
	// sandbox is already dead. Who are you sendign graceful to?
	s.twine.Stop()

	// Not sure if we can do this now, or have to wait til dead...
	err := os.RemoveAll(s.paths.hostPath("sockets"))
	if err != nil {
		// might ignore err?
		s.getLogger("Stop(), os.RemoveAll(sockets dir)").Error(err)
	}

	cGroupData := s.collectRunData()
	memByteSec := calcMemByteSec(cGroupData.MemoryBytes, tiedUpDuration)
	dbIDData := <-runDBIDCh
	if dbIDData.ok {
		s.SandboxRuns.End(dbIDData.id, time.Now(), domain.SandboxRunData{
			TiedUpMs:      int(tiedUpDuration.Milliseconds()),
			CpuUsec:       cGroupData.CpuUsec,
			MemoryByteSec: memByteSec,
			IOBytes:       cGroupData.IOBytes,
			IOs:           cGroupData.IOs})
	}

	if s.Config.Sandbox.UseCGroups {
		err = s.CGroups.RemoveCGroup(s.cGroup)
		if err != nil {
			s.getLogger("Stop(), s.CGroups.RemoveCGroup").Error(err)
		}
	}

	if s.outProxy != nil {
		s.outProxy.Stop()
	}

	s.getLogger("cleanup()").Log(fmt.Sprintf("Sandbox %v cleanup complete", s.id))
}

func (s *Sandbox) pidAlive() bool {
	if s.cmd == nil {
		return false
	}
	process := s.cmd.Process

	if process == nil {
		return false
	}

	// what does proces look like after the underlying process has dies?

	err := process.Signal(syscall.Signal(0))
	return err == nil
}

// kill sandbox, which means send it the kill sig
// This should get picked up internally and it should shut itself down.
func (s *Sandbox) kill(force bool) error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	process := s.cmd.Process

	sig := unix.SIGTERM
	if force {
		sig = unix.SIGKILL
	}
	err := process.Signal(sig)
	if err != nil {
		s.getLogger("kill()").AddNote(fmt.Sprintf("process.Signal(sig). Force: %t", force)).Error(err)
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
		return errors.New("sandbox failed to terminate") // is this a sentinel error?
	}
	return nil
}

func (s *Sandbox) createRun() chan runDBIDData {
	ch := make(chan runDBIDData)
	aID := domain.NewNullAppspaceID()
	if s.appspace != nil {
		aID.Set(s.appspace.AppspaceID)
	}
	go func() {
		dbID, err := s.SandboxRuns.Create(domain.SandboxRunIDs{
			Instance:   "ds-host", //this should come from elsewhere? Config?
			LocalID:    s.id,
			OwnerID:    s.ownerID,
			AppID:      s.appVersion.AppID,
			Version:    s.appVersion.Version,
			AppspaceID: aID,
			Operation:  s.operation,
			CGroup:     s.cGroup,
		}, time.Now())
		if err != nil {
			ch <- runDBIDData{0, false}
			close(ch)
			s.getLogger("createRun, Create").Error(err)
			return // for now we just bail.
		}
		ch <- runDBIDData{dbID, true}
		close(ch)
	}()

	return ch
}

func (s *Sandbox) collectRunData() domain.CGroupData {
	var metrics domain.CGroupData
	var err error
	if s.Config.Sandbox.UseCGroups {
		metrics, err = s.CGroups.GetMetrics(s.cGroup)
		if err != nil {
			s.getLogger("collectRunData").Error(err)
		}
	}
	return metrics
}

func calcMemByteSec(memBytes int, tiedUp time.Duration) int {
	return memBytes * int(tiedUp.Milliseconds()) / 1000
}

// Status returns the status of the Sandbox
func (s *Sandbox) Status() domain.SandboxStatus {
	s.statusMux.Lock()
	defer s.statusMux.Unlock()
	return s.status
}

// ExecFn executes a function in athesandbox, based on the params of AppspaceRouteHandler
func (s *Sandbox) ExecFn(handler domain.AppspaceRouteHandler) error {
	// check status of sandbox first?
	// taskCh := s.	// need to rethink how tasks are managed when calling ExecFn.
	// If folliwng the pattern of http requests, then it's up to the caller to signal task start and end?
	// but that seems unnecessary here.
	// defer func() {
	// 	taskCh <- true
	// }()

	payload, err := json.Marshal(handler)
	if err != nil {
		// this is an input error. The caller is at fault probably. Don't log.
		return err //return a "bad input" error, and wrap it?
	}

	sent, err := s.twine.Send(executeService, execFnCommand, payload)
	if err != nil {
		s.getLogger("ExecFn(), s.twine.Send()").Error(err)
		return errors.New("error sending to sandbox")
	}

	// maybe receive interim progress messages?

	reply, err := sent.WaitReply()
	if err != nil {
		s.getLogger("ExecFn(), sent.WaitReply()").Error(err)
		return err
	}

	if !reply.OK() {
		return reply.Error()
	}

	return nil
}

// SendMessage sends to sandbox via twine
func (s *Sandbox) SendMessage(serviceID int, commandID int, payload []byte) (twine.SentMessageI, error) {
	sent, err := s.twine.Send(serviceID, commandID, payload)
	if err != nil {
		return nil, err
	}
	return sent, nil
}

// GetTransport gets the http transport of the sandbox
func (s *Sandbox) GetTransport() http.RoundTripper {
	if s.transport == nil {
		s.transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", filepath.Join(s.paths.hostPath("sockets"), "server.sock"))
			},
		}
	}
	return s.transport
}

// getLogger retuns the Logging client
func (s *Sandbox) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("Sandbox")
	if s.appVersion != nil {
		l.AppID(s.appVersion.AppID).AppVersion(s.appVersion.Version)
	}
	if s.appspace != nil {
		l.AppspaceID(s.appspace.AppspaceID)
	}
	if note != "" {
		l.AddNote(note)
	}
	return l
}

// TiedUp indicates whether the sandbox has unfinished tasks
func (s *Sandbox) TiedUp() bool {
	return s.taskTracker.isTiedUp()
}

// LastActive returns the last used time
func (s *Sandbox) LastActive() time.Time {
	return s.taskTracker.lastActive
}

// NewTask returns a channel used to signal the beginning and end of a task.
// Pass a struct to indicate the task has started, close to indicate it has ended.
func (s *Sandbox) NewTask() chan struct{} {
	return s.taskTracker.newTask()
}

// setStatus sets the status
func (s *Sandbox) setStatus(status domain.SandboxStatus) {
	s.statusMux.Lock()
	defer s.statusMux.Unlock()

	if status > s.status {

		s.status = status
		for stat, subs := range s.waitStatusSub {
			if stat <= s.status {
				for _, sub := range subs {
					sub <- s.status
				}
				delete(s.waitStatusSub, stat)
			}
		}
		for _, sub := range s.statusSub {
			sub <- s.status
		}
		if status == domain.SandboxCleanedUp {
			for _, sub := range s.statusSub {
				close(sub)
			}
			s.statusSub = make([]chan domain.SandboxStatus, 0)
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

	if _, ok := s.waitStatusSub[status]; !ok {
		s.waitStatusSub[status] = []chan domain.SandboxStatus{}
	}
	statusMet := make(chan domain.SandboxStatus)
	s.waitStatusSub[status] = append(s.waitStatusSub[status], statusMet)

	s.statusMux.Unlock()

	<-statusMet
}

// SubscribeStatus returns a channel that will receive status for this sandbox
// It automatically closes the channel after sending sandbox dead
func (s *Sandbox) SubscribeStatus() chan domain.SandboxStatus {
	s.statusMux.Lock()
	defer s.statusMux.Unlock()
	sub := make(chan domain.SandboxStatus)
	s.statusSub = append(s.statusSub, sub)
	return sub
}

func (s *Sandbox) log(logString string) {
	if s.Logger != nil && !reflect.ValueOf(s.Logger).IsNil() {
		go s.Logger.Log("sandbox-"+strconv.Itoa(s.id), logString)
	}
}

func (s *Sandbox) createPaths() {
	s.paths.appLoc = s.appVersion.LocationKey
	if s.appspace != nil {
		s.paths.appspaceLoc = s.appspace.LocationKey
	}
	s.paths.Config = s.Config
	s.paths.AppLocation2Path = s.AppLocation2Path
	s.paths.AppspaceLocation2Path = s.AppspaceLocation2Path
	s.paths.init()
}

// ImportPaths defines a type for creating imopsts.json for Deno
type ImportPaths struct {
	Imports map[string]string `json:"imports"`
}

func (s *Sandbox) writeImportMap() error {
	// I'd like to only write the import map if it's not written.
	// BUT! this means that if anything changes in ds-host
	// or the admin turns on bubblewrap for ex, all the import maps are broken
	// So can't do that until we're smarter about it.

	im := s.paths.denoImportMap()

	data, err := json.Marshal(im)
	if err != nil {
		s.getLogger("writeImportMap()").Error(err)
		return err
	}

	p := s.paths.hostPath("import-map")
	err = os.WriteFile(p, data, 0600)
	if err != nil {
		s.getLogger("writeImportMap()").AddNote("os.WriteFile file: " + p).Error(err)
		return err
	}

	return nil
}

func (s *Sandbox) writeBootstrapFile() error { // divide this into two: getting the contenst o fo the file and writing it to disk
	p := s.paths.hostPath("bootstrap")

	// Same as import map, we have to write on every sb invocation until we have a mechanism for
	// removing this file on config change.

	str := "import '" + path.Join(s.paths.sandboxPath("sandbox-runner"), "index.ts") + "';\n"
	str += "import '" + path.Join(s.paths.sandboxPath("app-files"), s.appVersion.Entrypoint) + "';"

	err := os.WriteFile(p, []byte(str), 0600)
	if err != nil {
		s.getLogger("writeBootstrapFile()").AddNote("os.WriteFile file: " + p).Error(err)
		return err
	}
	return nil
}
