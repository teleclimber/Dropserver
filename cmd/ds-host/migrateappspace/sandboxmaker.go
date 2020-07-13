package migrateappspace

//go:generate mockgen -destination=mocks.go -package=migrateappspace -self_package=github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace SandboxMakerI

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
)

// SandboxMakerI interface decouples sandbox implementation from its use for migration
type SandboxMakerI interface {
	Make() domain.SandboxI
}

// SandboxMaker holds data necessary to create a new migration sandbox
type SandboxMaker struct {
	ReverseServices *domain.ReverseServices
	Config          *domain.RuntimeConfig
}

// Make a new migration sandbox
func (m *SandboxMaker) Make() domain.SandboxI {
	return sandbox.NewSandbox(0, m.ReverseServices, m.Config)
}
