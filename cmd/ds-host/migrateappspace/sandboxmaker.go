package migrateappspace

//go:generate mockgen -destination=mocks.go -package=migrateappspace -self_package=github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace SandboxMakerI

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
)

// SandboxMakerI interface decouples sandbox implementation from its use for migration
type SandboxMakerI interface {
	Make(appVersion *domain.AppVersion, appspace *domain.Appspace) domain.SandboxI
}

// SandboxMaker holds data necessary to create a new migration sandbox
type SandboxMaker struct {
	AppspaceLogger interface {
		Log(domain.AppspaceID, string, string)
	}
	Services interface {
		Get(appspace *domain.Appspace, api domain.APIVersion) domain.ReverseServiceI
	}
	Config *domain.RuntimeConfig
}

// Make a new migration sandbox
func (m *SandboxMaker) Make(appVersion *domain.AppVersion, appspace *domain.Appspace) domain.SandboxI {
	s := sandbox.NewSandbox(0, appVersion, appspace, m.Services.Get(appspace, appVersion.APIVersion), m.Config)
	s.AppspaceLogger = m.AppspaceLogger
	return s
}
