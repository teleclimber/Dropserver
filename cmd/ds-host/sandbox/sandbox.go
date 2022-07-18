package sandbox

// Keep this module but make significant changes to
// have it just manage deno processes?

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
	"golang.org/x/sys/unix"
)

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
		GetMetrics(string) (domain.CGroupData, error)
		RemoveCGroup(string) error
	}
	Logger        interface{ Log(string, string) }
	socketsDir    string
	cmd           *exec.Cmd
	twine         *twine.Twine
	Services      domain.ReverseServiceI
	statusMux     sync.Mutex
	status        domain.SandboxStatus
	statusSub     []chan domain.SandboxStatus
	waitStatusSub map[domain.SandboxStatus][]chan domain.SandboxStatus
	transport     http.RoundTripper
	taskTracker   taskTracker
	inspect       bool
	Location2Path interface {
		AppMeta(string) string
		AppFiles(string) string
	}
	Config *domain.RuntimeConfig

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
		waitStatusSub: make(map[domain.SandboxStatus][]chan domain.SandboxStatus)}

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

// Start sets the status to starting
// and begins the start process in a goroutine.
func (s *Sandbox) Start() {
	s.setStatus(domain.SandboxStarting)
	go func() {
		err := s.doStart()
		if err != nil {
			s.Kill()
			return
		}
	}()
}

func (s *Sandbox) doStart() error {
	logger := s.getLogger("doStart()")
	logger.Debug("starting...")

	s.setStatus(domain.SandboxStarting) // should already be set, but just in case

	// Mark beginning of start here
	tStart := time.Now()
	tRef := time.Now()

	memHigh := 128 * 1024 * 1024
	if s.operation == opAppInit {
		memHigh = 256 * 1024 * 1024 // allow more memory for type-checking
	}
	limits := domain.CGroupLimits{
		MemoryHigh: memHigh,
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

	socketsDir, err := ioutil.TempDir(s.Config.Sandbox.SocketsDir, "sock")
	if err != nil {
		logger.AddNote(fmt.Sprintf("ioutil.TempDir() dir: %v", s.Config.Sandbox.SocketsDir)).Error(err)
		return err
	}
	s.socketsDir = socketsDir

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

	tRef = time.Now()

	twineServer, err := twine.NewUnixServer(path.Join(socketsDir, "rev.sock"))
	if err != nil {
		logger.AddNote("twine.NewUnixServer").Error(err)
		return err // maybe return a user-centered error
	}
	s.twine = twineServer

	tStr += fmt.Sprintf(" Start Twine: %s", time.Since(tRef))

	err = os.Setenv("NO_COLOR", "true")
	if err != nil {
		logger.AddNote("os.Setenv").Error(err)
		return err // return user-centered error
	}

	appspacePath := "/dev/null"
	if s.appspace != nil {
		appspacePath = s.getAppspaceDataPath()
	}

	// Probably need to think more about flags we pass, such as --no-remote?
	denoArgs := make([]string, 0, 10)
	denoArgs = append(denoArgs, "run", "--unstable")
	if s.inspect {
		denoArgs = append(denoArgs, "--inspect-brk")
	}

	readFiles, readWriteFiles := s.populateFilePerms()
	if len(readFiles) == 0 || len(readWriteFiles) == 0 {
		// "--allow-read=" could be interpreted by Deno as allowing all reads!
		panic("sandbox readFiles or readWriteFiles are empty")
	}

	typeCheck := "--no-check"
	if s.operation == opAppInit {
		typeCheck = "--check"
	}

	runArgs := []string{
		typeCheck,
		"--importmap=" + s.getImportPathFile(),
		"--allow-read=" + strings.Join(append(readFiles, readWriteFiles...), ","),
		"--allow-write=" + strings.Join(readWriteFiles, ","),
		//"--allow-net=...",
		s.getBootstrapFilename(),
		s.socketsDir,
		s.getAppFilesPath(), // while we have an import-map, these are still needed to read files without importing
		appspacePath,
	}

	denoArgs = append(denoArgs, runArgs...)
	s.cmd = exec.Command("deno", denoArgs...)

	// Note that ultimately we need to stick this in a Cgroup

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
	}

	tStr += fmt.Sprintf(" Pid to CGroup: %s", time.Since(tRef))

	go s.monitor(stdout, stderr, runDbIDCh)

	_, ok := <-s.twine.ReadyChan
	if !ok {
		logger.Log("Apparent failed start. ReadyChan closed")
		s.Kill()
		return errors.New("failed to start sandbox")
	}

	go s.listenMessages()

	tStr += fmt.Sprintf(" Total to Twine ready: %s", time.Since(tStart))

	go func() {
		s.WaitFor(domain.SandboxReady)
		tStr += fmt.Sprintf(" Total to Sandbox ready: %s", time.Since(tStart))
		logger.Debug(tStr)
	}()

	return nil
}

// monitor waits for cmd to end or an error gets sent
// It also collects output for use somewhere.
func (s *Sandbox) monitor(stdout io.ReadCloser, stderr io.ReadCloser, runDBIDCh chan runDBIDData) {

	go func() {
		for { // you need to be in a loop to keep the channel "flowing"
			err := <-s.twine.ErrorChan
			if err != nil {
				s.getLogger("ErrorChan").Error(err)
				// We may want to stash a message on s. to enlighten user as to what happened?
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
		// TODO check error type (see Wait comment)
		s.getLogger("monitor(), s.cmd.Wait()").Error(err)
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
			if s.Logger != nil {
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
	s.getLogger("Graceful()").Log("starting shutdown")

	s.setStatus(domain.SandboxKilling)

	go func() {
		// it seems this SendBlock does not return. Bug in twine?
		reply, err := s.twine.SendBlock(sandboxService, 13, nil)
		if err != nil {
			// ???
			s.getLogger("Graceful() twine.SendBlock()").Error(err)
		}
		if !reply.OK() {
			s.getLogger("Graceful() twine.SendBlock()").Log("response not OK")
		}

		s.getLogger("Graceful()").Log("sending twine.Graceful()")

		// Then tell twine to shut down nicely:
		s.twine.Graceful()

		s.getLogger("Graceful()").Log("graceful complete")
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
	err := os.RemoveAll(s.socketsDir)
	if err != nil {
		// might ignore err?
		s.getLogger("Stop(), os.RemoveAll(s.socketsDir)").Error(err)
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

	s.getLogger("cleanup()").Log("cleanup complete")
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
				return net.Dial("unix", filepath.Join(s.socketsDir, "server.sock"))
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

	s.getLogger("setStatus()").Log(fmt.Sprintf("Sandbox %v set status from %v to %v\n", s.id, s.status, status))

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
	fmt.Println(s.id, "waiting for sandbox status", status)

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
	if s.Logger != nil {
		go s.Logger.Log("sandbox-"+strconv.Itoa(s.id), logString)
	}
}

// ImportPaths defines a type for creating imopsts.json for Deno
type ImportPaths struct {
	Imports map[string]string `json:"imports"`
}

func (s *Sandbox) makeImportMap() ([]byte, error) {
	bootstrapPath := s.getBootstrapFilename()
	appPath := trailingSlash(s.getAppFilesPath())
	dropserverPath := trailingSlash(s.Config.Exec.SandboxCodePath)
	// TODO: check that none of these paths are "/" as this can defeat protection against forbidden imports.

	im := ImportPaths{
		Imports: map[string]string{
			"/":            "/dev/null/", // Defeat imports from outside the app dir. See:
			"./":           "./",         // https://github.com/denoland/deno/issues/6294#issuecomment-663256029
			bootstrapPath:  bootstrapPath,
			appPath:        appPath,
			dropserverPath: dropserverPath,

			// // TODO DELETE EXTERMELY TEMPORARY
			// "https://deno.land/x/dropserver_lib_support/":         "/Users/ollie/Documents/Code/dropserver_lib_support/",
			// "/Users/ollie/Documents/Code/dropserver_lib_support/": "/Users/ollie/Documents/Code/dropserver_lib_support/",
			// "https://deno.land/x/dropserver_app/":                 "/Users/ollie/Documents/Code/dropserver_app/",
			// "/Users/ollie/Documents/Code/dropserver_app/":         "/Users/ollie/Documents/Code/dropserver_app/",
			// "https://deno.land/x/twine@0.1.0/":        "/Users/ollie/Documents/Code/twine-deno/",
			// "/Users/ollie/Documents/Code/twine-deno/": "/Users/ollie/Documents/Code/twine-deno/",
		}}

	if s.appspace != nil {
		appspacePath := trailingSlash(s.getAppspaceFilesPath())
		im = ImportPaths{
			Imports: map[string]string{
				"/":            "/dev/null/", // Defeat imports from outside the app dir. See:
				"./":           "./",         // https://github.com/denoland/deno/issues/6294#issuecomment-663256029
				bootstrapPath:  bootstrapPath,
				appPath:        appPath,
				appspacePath:   appspacePath,
				dropserverPath: dropserverPath,

				// TODO DELETE EXTERMELY TEMPORARY
				// "https://deno.land/x/dropserver_lib_support/":         "/Users/ollie/Documents/Code/dropserver_lib_support/",
				// "/Users/ollie/Documents/Code/dropserver_lib_support/": "/Users/ollie/Documents/Code/dropserver_lib_support/",
				// "https://deno.land/x/dropserver_app/":                 "/Users/ollie/Documents/Code/dropserver_app/",
				// "/Users/ollie/Documents/Code/dropserver_app/":         "/Users/ollie/Documents/Code/dropserver_app/",
			}}
	}

	j, err := json.Marshal(im)
	if err != nil {
		s.getLogger("makeImportMap()").Error(err)
		return nil, err
	}

	return j, nil
}
func trailingSlash(p string) string {
	if strings.HasSuffix(p, string(os.PathSeparator)) {
		return p
	}
	return p + string(os.PathSeparator)
}

func (s *Sandbox) writeImportMap() error {
	data, err := s.makeImportMap()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(s.getImportPathFile(), data, 0600)
	if err != nil {
		s.getLogger("writeImportMap()").AddNote("ioutil.WriteFile file: " + s.getImportPathFile()).Error(err)
		return err
	}

	return nil
}

func (s *Sandbox) getBootstrapFilename() string {
	return filepath.Join(s.Location2Path.AppMeta(s.appVersion.LocationKey), "bootstrap.js")
}
func (s *Sandbox) writeBootstrapFile() error {
	p := s.getBootstrapFilename()
	if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
		s.getLogger("writeBootstrapFile()").Debug("Skipping writing bootstrap js file")
		return nil
	}

	str := "import '" + path.Join(s.Config.Exec.SandboxCodePath, "index.ts") + "';\n"
	str += "import '" + path.Join(s.Location2Path.AppFiles(s.appVersion.LocationKey), "app.ts") + "';"

	err := ioutil.WriteFile(p, []byte(str), 0600)
	if err != nil {
		s.getLogger("writeBootstrapFile()").AddNote("ioutil.WriteFile file: " + p).Error(err)
		return err
	}
	return nil
}

func (s *Sandbox) populateFilePerms() (readFiles, readWriteFiles []string) {
	// sandbox runner and socket:
	readWriteFiles = append(readWriteFiles, s.socketsDir)

	// read app files:
	readFiles = append(readFiles, s.Location2Path.AppFiles(s.appVersion.LocationKey))

	if s.appspace == nil {
		return
	}

	// readonly avatars dir:
	readFiles = append(readFiles, filepath.Join(s.getAppspaceDataPath(), "avatars"))

	// read-write appspace files:
	readWriteFiles = append(readWriteFiles, filepath.Join(s.getAppspaceFilesPath()))

	// TODO probably need to process / quote / check / escape these strings for problems?

	return
}

// this should be taken care of by location2path
func (s *Sandbox) getAppFilesPath() string {
	return s.Location2Path.AppFiles(s.appVersion.LocationKey)
}
func (s *Sandbox) getAppspaceDataPath() string {
	return filepath.Join(s.Config.Exec.AppspacesPath, s.appspace.LocationKey, "data")
}
func (s *Sandbox) getAppspaceFilesPath() string {
	return filepath.Join(s.Config.Exec.AppspacesPath, s.appspace.LocationKey, "data", "files")
}
func (s *Sandbox) getImportPathFile() string {
	if s.appspace != nil {
		return filepath.Join(s.Config.Exec.AppspacesPath, s.appspace.LocationKey, "import-paths.json")
	} else {
		return filepath.Join(s.Location2Path.AppMeta(s.appVersion.LocationKey), "import-paths.json")
	}
}
