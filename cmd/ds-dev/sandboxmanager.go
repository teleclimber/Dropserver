package main

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
)

// DevSandboxManager manages a single sandbox that can be resatrted to recompile app code
type DevSandboxManager struct {
	Services *domain.ReverseServices
}

// need Start/Stop/Restart functions

// GetForAppSpace always returns the crrent sandbox
func (m *DevSandboxManager) GetForAppSpace(*domain.AppVersion, *domain.Appspace) chan domain.SandboxI {
	ch := make(chan domain.SandboxI)

	return ch

}

// StopAppspace is used to stop an appspace sandbox from running if there is one
// it returns if/when no sanboxes are running for that appspace
func (m *DevSandboxManager) StopAppspace(appspaceID domain.AppspaceID) {
	// s, ok := sM.sandboxes[appspaceID]
	// if !ok {
	// 	return
	// }

	// s.Stop() // this should work but sandbox manager may not be updated because bugg
}

////////////////////////////////////////////////////

// DevSandboxMaker holds data necessary to create a new migration sandbox
type DevSandboxMaker struct {
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
	s.SetInspect(m.inspect)
	return s
}
