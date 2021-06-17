package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=sandbox_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks SandboxMaker,SandboxManager

type SandboxMaker interface {
	ForApp(appVersion *domain.AppVersion) (domain.SandboxI, error)
	ForMigration(appVersion *domain.AppVersion, appspace *domain.Appspace) (domain.SandboxI, error)
}

// SandboxManager is an interface that describes sm
type SandboxManager interface {
	GetForAppspace(appVersion *domain.AppVersion, appspace *domain.Appspace) chan domain.SandboxI
	StopAppspace(domain.AppspaceID)
}
