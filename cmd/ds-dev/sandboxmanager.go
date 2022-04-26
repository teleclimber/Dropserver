package main

import (
	"errors"
	"os"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
)

var opAppInit = "app-init"
var opAppspaceRun = "appspace-run"
var opAppspaceMigration = "appspace-migration"

// DevSandboxManager manages a single sandbox that can be resatrted to recompile app code
type DevSandboxManager struct {
	SandboxRuns interface {
		Create(run domain.SandboxRunIDs, start time.Time) (int, error)
		End(sandboxID int, end time.Time, data domain.SandboxRunData) error
	} `checkinject:"required"`
	AppLogger interface {
		Get(string) domain.LoggerI
	} `checkinject:"required"`
	AppspaceLogger interface {
		Get(domain.AppspaceID) domain.LoggerI
	} `checkinject:"required"`
	Services interface {
		Get(appspace *domain.Appspace, api domain.APIVersion) domain.ReverseServiceI
	} `checkinject:"required"`
	AppVersionEvents interface {
		Subscribe(chan<- string)
	} `checkinject:"required"`
	SandboxStatusEvents interface {
		Send(SandboxStatus)
	} `checkinject:"required"`
	Location2Path interface {
		AppMeta(string) string
		AppFiles(string) string
	} `checkinject:"required"`
	Config *domain.RuntimeConfig `checkinject:"required"`

	appSb       domain.SandboxI
	migrationSb domain.SandboxI
	appspaceSb  domain.SandboxI

	nextID int

	inspect bool
}

// Init sets up app version events loop
func (m *DevSandboxManager) Init() {
	err := os.MkdirAll(m.Config.Sandbox.SocketsDir, 0700)
	if err != nil {
		panic(err)
	}
	appVersionEvent := make(chan string)
	m.AppVersionEvents.Subscribe(appVersionEvent) // this should probably come from app watcher
	go func() {
		for e := range appVersionEvent {
			if e == "loading" {
				go m.StopAppspace(appspaceID)
			}
		}
	}()
}

// need Start/Stop/Restart functions

// GetForAppspace always returns the crrent sandbox
func (m *DevSandboxManager) GetForAppspace(appVersion *domain.AppVersion, appspace *domain.Appspace) (domain.SandboxI, chan struct{}) {
	if m.appspaceSb == nil {
		m.startSandbox(appVersion, appspace)
	}
	return m.appspaceSb, m.appspaceSb.NewTask()
}

func (m *DevSandboxManager) startSandbox(appVersion *domain.AppVersion, appspace *domain.Appspace) {
	s := sandbox.NewSandbox(m.getNextID(), opAppspaceRun, ownerID, appVersion, appspace)
	s.SandboxRuns = m.SandboxRuns
	s.Services = m.Services.Get(appspace, 0)
	s.Logger = m.AppspaceLogger.Get(appspace.AppspaceID)
	s.Location2Path = m.Location2Path
	s.Config = m.Config
	s.SetInspect(m.inspect)
	m.appspaceSb = s

	statCh := s.SubscribeStatus()
	go func() {
		for stat := range statCh {
			m.SandboxStatusEvents.Send(SandboxStatus{
				Type:   "appspace",
				Status: stat,
			})
		}
	}()

	s.Start()

	go func() {
		s.WaitFor(domain.SandboxKilling)
		m.appspaceSb = nil
	}()
}

// StopAppspace is used to stop an appspace sandbox from running if there is one
// it returns if/when no sanboxes are running for that appspace
func (m *DevSandboxManager) StopAppspace(appspaceID domain.AppspaceID) {
	if m.appspaceSb != nil {
		m.appspaceSb.Graceful()
	}
}

func (m *DevSandboxManager) ForApp(appVersion *domain.AppVersion) (domain.SandboxI, error) {
	s := sandbox.NewSandbox(m.getNextID(), opAppInit, ownerID, appVersion, nil)
	s.SandboxRuns = m.SandboxRuns
	s.Logger = m.AppLogger.Get("")
	s.Location2Path = m.Location2Path
	s.Config = m.Config
	s.SetInspect(m.inspect)

	m.appSb = s

	statCh := s.SubscribeStatus()
	go func() {
		for stat := range statCh {
			m.SandboxStatusEvents.Send(SandboxStatus{
				Type:   "app",
				Status: stat,
			})
		}
	}()

	s.Start()
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

// Make a new migration sandbox
func (m *DevSandboxManager) ForMigration(appVersion *domain.AppVersion, appspace *domain.Appspace) (domain.SandboxI, error) {
	s := sandbox.NewSandbox(m.getNextID(), opAppspaceMigration, ownerID, appVersion, appspace)
	s.SandboxRuns = m.SandboxRuns
	s.Services = m.Services.Get(appspace, appVersion.APIVersion)
	s.Location2Path = m.Location2Path
	s.Logger = m.AppspaceLogger.Get(appspaceID)
	s.Config = m.Config
	s.SetInspect(m.inspect)

	m.migrationSb = s

	statCh := s.SubscribeStatus()
	go func() {
		for stat := range statCh {
			m.SandboxStatusEvents.Send(SandboxStatus{
				Type:   "migration",
				Status: stat,
			})
		}
	}()

	s.Start()
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

//SetInspect sets the inspect flag which makes the sandbox wait for a debugger to attach
func (m *DevSandboxManager) SetInspect(inspect bool) {
	m.inspect = inspect
}

func (m *DevSandboxManager) StopSandboxes() {
	if m.appspaceSb != nil {
		m.appspaceSb.Kill()
	}
	if m.appSb != nil {
		m.appSb.Kill()
	}
	if m.migrationSb != nil {
		m.migrationSb.Kill()
	}
}

func (m *DevSandboxManager) getNextID() int {
	m.nextID++
	return m.nextID
}
