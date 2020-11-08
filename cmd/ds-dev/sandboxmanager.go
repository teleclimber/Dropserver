package main

import (
	"os"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
)

var sandboxID = 0

// DevSandboxManager manages a single sandbox that can be resatrted to recompile app code
type DevSandboxManager struct {
	AppspaceLogger interface {
		Log(domain.AppspaceID, string, string)
	}
	Services interface {
		Get(appspace *domain.Appspace, api domain.APIVersion) domain.ReverseServiceI
	}
	AppVersionEvents interface {
		Subscribe(chan<- domain.AppID)
	}
	Config *domain.RuntimeConfig

	sb      domain.SandboxI
	inspect bool
}

// Init sets up app version events loop
func (sM *DevSandboxManager) Init() {
	err := os.MkdirAll(sM.Config.Sandbox.SocketsDir, 0700)
	if err != nil {
		panic(err)
	}
	appVersionEvent := make(chan domain.AppID)
	sM.AppVersionEvents.Subscribe(appVersionEvent)
	go func() {
		for range appVersionEvent {
			go sM.StopAppspace(appspaceID)
		}
	}()
}

// need Start/Stop/Restart functions

// GetForAppSpace always returns the crrent sandbox
func (sM *DevSandboxManager) GetForAppSpace(appVersion *domain.AppVersion, appspace *domain.Appspace) chan domain.SandboxI {
	ch := make(chan domain.SandboxI)

	if sM.sb != nil {
		go func() {
			sM.sb.WaitFor(domain.SandboxReady)
			ch <- sM.sb
		}()
	} else {
		sM.startSandbox(appVersion, appspace, ch)
	}

	return ch

}

func (sM *DevSandboxManager) startSandbox(appVersion *domain.AppVersion, appspace *domain.Appspace, ch chan domain.SandboxI) {
	newSandbox := sandbox.NewSandbox(sandboxID, appVersion, appspace, sM.Services.Get(appspace, 0), sM.Config)
	newSandbox.AppspaceLogger = sM.AppspaceLogger
	newSandbox.SetInspect(sM.inspect)
	sM.sb = newSandbox

	go func() {
		err := newSandbox.Start()
		if err != nil {
			close(ch)
			newSandbox.Kill()
			return
		}
		newSandbox.WaitFor(domain.SandboxReady)
		// sandbox may not be ready if it failed to start.
		// check status? Or maybe status ought to be checked by proxy for each request anyways?
		ch <- newSandbox
	}()

	go func() {
		newSandbox.WaitFor(domain.SandboxKilling)
		sM.sb = nil
	}()

	sandboxID++
}

// StopAppspace is used to stop an appspace sandbox from running if there is one
// it returns if/when no sanboxes are running for that appspace
func (sM *DevSandboxManager) StopAppspace(appspaceID domain.AppspaceID) {
	if sM.sb != nil {
		sM.sb.Graceful()
	}
}

//SetInspect sets the inspect flag which makes the sandbox wait for a debugger to attach
func (sM *DevSandboxManager) SetInspect(inspect bool) {
	sM.inspect = inspect
}

////////////////////////////////////////////////////

// DevSandboxMaker holds data necessary to create a new migration sandbox
type DevSandboxMaker struct {
	AppspaceLogger interface {
		Log(domain.AppspaceID, string, string)
	}
	Services interface {
		Get(appspace *domain.Appspace, api domain.APIVersion) domain.ReverseServiceI
	}
	Config  *domain.RuntimeConfig
	inspect bool
}

// here we can potentially add setDebug mode to pass to NewSandbox,
// ..which should manifest as --debug or --inspect-brk

// SetInspect sets the inspect flag on the sandbox
func (m *DevSandboxMaker) SetInspect(inspect bool) {
	m.inspect = inspect
}

// Make a new migration sandbox
func (m *DevSandboxMaker) Make(appVersion *domain.AppVersion, appspace *domain.Appspace) domain.SandboxI {
	s := sandbox.NewSandbox(sandboxID, appVersion, appspace, m.Services.Get(appspace, appVersion.APIVersion), m.Config)
	sandboxID++
	s.AppspaceLogger = m.AppspaceLogger
	s.SetInspect(m.inspect)
	return s
}
