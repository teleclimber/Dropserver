package main

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
)

// DevSandboxManager manages a single sandbox that can be resatrted to recompile app code
type DevSandboxManager struct {
	AppspaceLogger interface {
		Log(domain.AppspaceID, string, string)
	}
	Services *domain.ReverseServices
	Config   *domain.RuntimeConfig

	nextID int
	sb     domain.SandboxI
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
	sandboxID := sM.nextID
	sM.nextID++

	newSandbox := sandbox.NewSandbox(sandboxID, sM.Services, sM.Config)
	newSandbox.AppspaceLogger = sM.AppspaceLogger
	sM.sb = newSandbox

	go func() {
		err := newSandbox.Start(appVersion, appspace)
		if err != nil {
			close(ch)
			newSandbox.Stop()
			return
		}
		newSandbox.WaitFor(domain.SandboxReady)
		// sandbox may not be ready if it failed to start.
		// check status? Or maybe status ought to be checked by proxy for each request anyways?
		ch <- newSandbox
	}()
}

// StopAppspace is used to stop an appspace sandbox from running if there is one
// it returns if/when no sanboxes are running for that appspace
func (sM *DevSandboxManager) StopAppspace(appspaceID domain.AppspaceID) {
	if sM.sb != nil {
		sM.sb.Stop()
	}
}

////////////////////////////////////////////////////

// DevSandboxMaker holds data necessary to create a new migration sandbox
type DevSandboxMaker struct {
	AppspaceLogger interface {
		Log(domain.AppspaceID, string, string)
	}
	ReverseServices *domain.ReverseServices
	Config          *domain.RuntimeConfig
	inspect         bool
}

// here we can potentially add setDebug mode to pass to NewSandbox,
// ..which should manifest as --debug or --inspect-brk

// SetInspect sets the inspect flag on the sandbox
func (m *DevSandboxMaker) SetInspect(inspect bool) {
	m.inspect = inspect
}

// Make a new migration sandbox
func (m *DevSandboxMaker) Make() domain.SandboxI {
	s := sandbox.NewSandbox(0, m.ReverseServices, m.Config)
	s.AppspaceLogger = m.AppspaceLogger
	s.SetInspect(m.inspect)
	return s
}
