package sandbox

import (
	"errors"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

var opAppInit = "app-init"
var opAppspaceRun = "appspace-run"
var opAppspaceMigration = "appspace-migration"

// Manager manages sandboxes
type Manager struct {
	Config      *domain.RuntimeConfig `checkinject:"required"`
	SandboxRuns interface {
		Create(run domain.SandboxRunIDs, start time.Time) (int, error)
		End(sandboxID int, end time.Time, data domain.SandboxRunData) error
	} `checkinject:"required"`
	CGroups interface {
		CreateCGroup(domain.CGroupLimits) (string, error)
		AddPid(string, int) error
		SetLimits(string, domain.CGroupLimits) error
		GetMetrics(string) (domain.CGroupData, error)
		RemoveCGroup(string) error
	} `checkinject:"optional"`
	AppLogger interface {
		Get(string) domain.LoggerI
	} `checkinject:"required"`
	AppspaceLogger interface {
		Open(domain.AppspaceID) domain.LoggerI
	} `checkinject:"required"`
	Services interface {
		Get(appspace *domain.Appspace, api domain.APIVersion) domain.ReverseServiceI
	} `checkinject:"required"`
	AppLocation2Path interface {
		Meta(string) string
		Files(string) string
	} `checkinject:"required"`
	AppspaceLocation2Path interface {
		Base(string) string
		Data(string) string
		Files(string) string
		Avatars(string) string
	} `checkinject:"required"`

	idMux  sync.Mutex
	nextID int

	sandboxesMux sync.Mutex
	sandboxes    []domain.SandboxI

	ticker *time.Ticker
}

// Init creates maps
func (m *Manager) Init() {
	m.sandboxes = make([]domain.SandboxI, 0)

	err := os.MkdirAll(m.Config.Sandbox.SocketsDir, 0700)
	if err != nil {
		panic(err)
	}

	m.ticker = time.NewTicker(2 * time.Minute)
	go func() {
		for range m.ticker.C {
			m.startStopSandboxes()
		}
	}()
}

// StopAll takes all known sandboxes and stops them.
// It also stops the ticker
func (m *Manager) StopAll() {
	m.ticker.Stop()

	// lock Mutex?
	var stopWg sync.WaitGroup
	for _, sb := range m.sandboxes {
		// Should migration and app init sandboxes be treated differently?
		// Like, just wait for them to end naturally?
		stopWg.Add(1)
		go func(sb1 domain.SandboxI) {
			if sb1.Operation() == opAppspaceRun {
				sb1.Graceful()
			}
			sb1.WaitFor(domain.SandboxCleanedUp)
			stopWg.Done()
		}(sb)
	}

	stopWg.Wait()

	m.getLogger("StopAll").Log("all sandboxes stopped")
}

// GetForAppspace records the need for a sandbox and returns a channel
// I wonder if this should essentially tie up the sandbox
// such that it doesn't get cleaned out before the request gets passed to it.
func (m *Manager) GetForAppspace(appVersion *domain.AppVersion, appspace *domain.Appspace) (domain.SandboxI, chan struct{}) {
	m.sandboxesMux.Lock()
	defer m.sandboxesMux.Unlock()

	s, found := m.findAppspaceSandbox(appVersion, appspace.AppspaceID)
	if !found {
		newS := NewSandbox(m.getNextID(), opAppspaceRun, appspace.OwnerID, appVersion, appspace)
		newS.AppLocation2Path = m.AppLocation2Path
		newS.AppspaceLocation2Path = m.AppspaceLocation2Path
		newS.Services = m.Services.Get(appspace, appVersion.APIVersion)
		newS.Logger = m.AppspaceLogger.Open(appspace.AppspaceID)
		newS.taskTracker.lastActive = time.Now()

		m.startSandbox(newS)

		s = newS
	}

	return s, s.NewTask()
}

// findAppspaceSandbox returns the first viable appspace-run sandbox for the appspace
// If appVersion is passed, the sandbox will match the app id and verion.
// Sandboxes that are dying or dead are not considered.
func (m *Manager) findAppspaceSandbox(appVersion *domain.AppVersion, appspaceID domain.AppspaceID) (domain.SandboxI, bool) {
	for _, s := range m.sandboxes {
		nullAID := s.AppspaceID()
		aID, hasAppspace := nullAID.Get()
		if hasAppspace && aID == appspaceID && s.Operation() == opAppspaceRun && s.Status() <= domain.SandboxReady {
			av := s.AppVersion()
			if appVersion == nil || av.AppID == appVersion.AppID && av.Version == appVersion.Version {
				return s, true
			}
		}
	}
	return nil, false
}

// StopAppspace is used to stop an appspace sandbox from running if there is one
// it returns if/when no sanboxes are running for that appspace
func (m *Manager) StopAppspace(appspaceID domain.AppspaceID) {
	m.sandboxesMux.Lock()
	s, found := m.findAppspaceSandbox(nil, appspaceID)
	if !found {
		m.sandboxesMux.Unlock()
		return
	}
	m.sandboxesMux.Unlock()

	s.Graceful()

	s.WaitFor(domain.SandboxDead)
}

func (m *Manager) ForApp(appVersion *domain.AppVersion) (domain.SandboxI, error) {
	// TODO need to add owner id

	m.sandboxesMux.Lock()

	s := NewSandbox(m.getNextID(), opAppInit, domain.UserID(0), appVersion, nil)
	s.AppLocation2Path = m.AppLocation2Path
	s.Logger = m.AppLogger.Get(appVersion.LocationKey)

	m.startSandbox(s)
	m.sandboxesMux.Unlock()

	s.WaitFor(domain.SandboxReady)

	if s.Status() != domain.SandboxReady {
		return nil, errors.New("failed to start sandbox")
	}

	taskCh := s.NewTask()
	taskCh <- struct{}{}
	go func() {
		s.WaitFor(domain.SandboxDead)
		close(taskCh)
	}()

	return s, nil
}

func (m *Manager) ForMigration(appVersion *domain.AppVersion, appspace *domain.Appspace) (domain.SandboxI, error) {
	m.sandboxesMux.Lock()

	s := NewSandbox(m.getNextID(), opAppspaceMigration, appspace.OwnerID, appVersion, appspace)
	s.AppLocation2Path = m.AppLocation2Path
	s.AppspaceLocation2Path = m.AppspaceLocation2Path
	s.Services = m.Services.Get(appspace, appVersion.APIVersion)
	s.Logger = m.AppspaceLogger.Open(appspace.AppspaceID)

	m.startSandbox(s)
	m.sandboxesMux.Unlock()

	s.WaitFor(domain.SandboxReady) // what if it never gets there?

	if s.Status() != domain.SandboxReady {
		return nil, errors.New("failed to start sandbox")
	}

	taskCh := s.NewTask()
	taskCh <- struct{}{}
	go func() {
		s.WaitFor(domain.SandboxDead)
		close(taskCh)
	}()

	return s, nil
}

// startSandbox launches a new Deno instance for a specific sandbox
// not sure if it should return a channel or just a started sb.
// Problem is if it takes too long, would like to independently send timout as response to request.
func (m *Manager) startSandbox(s *Sandbox) {
	// expects poolmux to be locked

	s.CGroups = m.CGroups
	s.Config = m.Config
	s.SandboxRuns = m.SandboxRuns

	m.sandboxes = append(m.sandboxes, s)

	go m.startStopSandboxes()

	go func() {
		s.WaitFor(domain.SandboxDead)
		m.removeSandbox(s)
		m.startStopSandboxes()
	}()
}
func (m *Manager) getNextID() int {
	m.idMux.Lock()
	defer m.idMux.Unlock()
	m.nextID++
	return m.nextID
}

func (m *Manager) removeSandbox(sandbox *Sandbox) {
	m.sandboxesMux.Lock()
	defer m.sandboxesMux.Unlock()
	for i, s := range m.sandboxes {
		if s == sandbox {
			m.sandboxes = append(m.sandboxes[:i], m.sandboxes[i+1:]...)
			return
		}
	}
	// is it an error if you don't find the sandbox oyou want to remove? Probably?
	m.getLogger("removeSandbox").Log("failed to find the sandbox to remove")
}

// startStopSandboxes determines which sandboxes should be stopped and which
// should can be started based on availabel resources.
func (m *Manager) startStopSandboxes() {
	m.sandboxesMux.Lock()
	defer m.sandboxesMux.Unlock()
	m.startStopSandboxesInner(m.Config, m.sandboxes)
}

func (m *Manager) recordSandboxStatusMetric() {
	// var s = &record.SandboxStatuses{ //TODO nope do not use imported record. inject instead
	// 	Starting:   0,
	// 	Ready:      0,
	// 	Committing: 0,
	// 	Committed:  0,
	// 	Recycling:  0}
	// for _, c := range sM.sandboxes {
	// 	switch c.Status {
	// 	case "starting":
	// 		s.Starting++
	// 	case "ready":
	// 		s.Ready++
	// 	case "committing":
	// 		s.Committing++
	// 	case "committed":
	// 		s.Committed++
	// 	case "recycling":
	// 		s.Recycling++
	// 	}
	// }
	// record.SandboxStatusCounts(s) //TODO nope
}

// PrintSandboxes outputs containersa and status
func (m *Manager) PrintSandboxes() {
	// var readys []string
	// for rc := sM.readySandboxes.Front(); rc != nil; rc = rc.Next() {
	// 	readys = append(readys, rc.Value.(*Sandbox).Name)
	// }

	// fmt.Println("Ready Sandboxes", readys)

	// fmt.Println("Committed sandboxes", sM.committedSandboxes)
	// for _, c := range sM.sandboxes {
	// 	tiedUp := "not-tied"
	// 	if c.appSpaceSession.tiedUp {
	// 		tiedUp = "tied-up"
	// 	}
	// 	fmt.Println(c.Name, c.Status, c.appSpaceID, tiedUp, c.recycleScore)
	// }
}

func (m *Manager) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("SandboxManager")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

type scored struct {
	sandbox domain.SandboxI
	score   float64
}

type startStopStatus struct {
	startables []scored
	stoppables []scored
	numRunning int
	numOld     int
	numDying   int
}

func (m *Manager) startStopSandboxesInner(config *domain.RuntimeConfig, sandboxes []domain.SandboxI) {
	status := getStartStoppables(sandboxes)
	doStartStop(config, status)
}

// doStartStop determines which sandboxes should be stopped and which
// should can be started based on available resources.
func doStartStop(config *domain.RuntimeConfig, status startStopStatus) {
	// if there are any in the start queue
	// first see if there are enough resources to start
	numMax := config.Sandbox.Num // For now just going to use num sandboxes from config as a resource max.
	numStartable := len(status.startables)
	if numStartable > 0 && status.numRunning < numMax {
		// if we have room to start any sandboxes, go do it.
		numStartNow := numMax - status.numRunning
		if numStartable < numStartNow {
			numStartNow = numStartable
		}
		for i := 0; i < numStartNow; i++ {
			status.startables[i].sandbox.Start()
		}
	}
	// try to shut off enough sandboxes to start as needed, or just to give us overhead
	// (running - dying) - (max - startable - margin) = num_to_kill
	numStopNow := status.numRunning - status.numDying - numMax + numStartable + 1 // right now hard-coding one sandbox of overhaed margin.
	// stop the old unused sandboxes even if we don't need the resources:
	if status.numOld > numStopNow {
		numStopNow = status.numOld
	}
	for i := 0; i < numStopNow && i < len(status.stoppables); i++ {
		status.stoppables[i].sandbox.Graceful()
	}
}

func getStartStoppables(sandboxes []domain.SandboxI) startStopStatus {
	s := startStopStatus{
		startables: make([]scored, 0),
		stoppables: make([]scored, 0),
		numRunning: 0,
		numOld:     0,
		numDying:   0}

	for _, sb := range sandboxes {
		sbStatus := sb.Status()
		if sbStatus == domain.SandboxPrepared {
			score := 0.0
			if sb.Operation() == opAppspaceRun {
				score = 10.0
			}
			s.startables = append(s.startables, scored{sandbox: sb, score: score})
		} else {
			s.numRunning++ // includes those starting up and shutting down
			if sb.Operation() == opAppspaceRun && sbStatus == domain.SandboxReady && !sb.TiedUp() {
				score := time.Since(sb.LastActive()).Seconds()
				s.stoppables = append(s.stoppables, scored{sandbox: sb, score: score})
				if score > 10*60 { // hard coded 10 minute max unused time for sandbox.
					s.numOld++
				}
			}
			if sbStatus == domain.SandboxKilling {
				s.numDying++
			}
		}
	}

	// sort the startablles and killables
	sort.Slice(s.startables, func(i, j int) bool {
		return s.startables[i].score > s.startables[j].score
	})
	sort.Slice(s.stoppables, func(i, j int) bool {
		return s.stoppables[i].score > s.stoppables[j].score
	})

	return s
}
