package main

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

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
