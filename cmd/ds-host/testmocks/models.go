package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

//go:generate mockgen -destination=models_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks AppModel,AppspaceModel,AppspaceInfoModels,MigrationJobModel

// AppModel is the interface for the app model
type AppModel interface {
	GetFromID(domain.AppID) (*domain.App, domain.Error)
	GetForOwner(domain.UserID) ([]*domain.App, domain.Error)
	Create(domain.UserID, string) (*domain.App, domain.Error)
	GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, domain.Error)
	GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, domain.Error)
	CreateVersion(domain.AppID, domain.Version, int, domain.APIVersion, string) (*domain.AppVersion, domain.Error)
	DeleteVersion(domain.AppID, domain.Version) domain.Error
}

// AppspaceModel is the interface for the appspace model
type AppspaceModel interface {
	GetFromID(domain.AppspaceID) (*domain.Appspace, domain.Error)
	GetFromSubdomain(string) (*domain.Appspace, domain.Error)
	GetForOwner(domain.UserID) ([]*domain.Appspace, domain.Error)
	GetForApp(domain.AppID) ([]*domain.Appspace, domain.Error)
	Create(domain.UserID, domain.AppID, domain.Version, string, string) (*domain.Appspace, domain.Error)
	Pause(domain.AppspaceID, bool) domain.Error
	SetVersion(domain.AppspaceID, domain.Version) domain.Error
}

// AppspaceInfoModels caches and dishes AppspaceInfoModels
type AppspaceInfoModels interface {
	Init()
	Get(domain.AppspaceID) domain.AppspaceInfoModel
	GetSchema(domain.AppspaceID) (int, error)
}

// MigrationJobModel handles writing jobs to the db
type MigrationJobModel interface {
	Create(domain.UserID, domain.AppspaceID, domain.Version, bool) (*domain.MigrationJob, error)
	GetJob(domain.JobID) (*domain.MigrationJob, error)
	GetPending() ([]*domain.MigrationJob, error)
	SetStarted(domain.JobID) (bool, error)
	SetFinished(domain.JobID, nulltypes.NullString) error
	//GetForAppspace(AppspaceID) (*MigrationJob, Error)
	// Delete(AppspaceID) Error
}
