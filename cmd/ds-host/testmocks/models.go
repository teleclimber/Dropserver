package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

//go:generate mockgen -destination=models_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks AppFilesModel,AppModel,AppspaceModel,AppspaceFilesModel,AppspaceInfoModels,AppspaceContactModel,MigrationJobModel

// AppFilesModel represents the application's files saved to disk
type AppFilesModel interface {
	Save(*map[string][]byte) (string, error)
	ReadMeta(string) (*domain.AppFilesMetadata, error)
	Delete(string) error
}

// AppModel is the interface for the app model
type AppModel interface {
	GetFromID(domain.AppID) (*domain.App, error)
	GetForOwner(domain.UserID) ([]*domain.App, error)
	Create(domain.UserID, string) (*domain.App, error)
	GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
	GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	CreateVersion(domain.AppID, domain.Version, int, domain.APIVersion, string) (*domain.AppVersion, error)
	DeleteVersion(domain.AppID, domain.Version) error
}

// AppspaceModel is the interface for the appspace model
type AppspaceModel interface {
	GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	GetFromSubdomain(string) (*domain.Appspace, error)
	GetForOwner(domain.UserID) ([]*domain.Appspace, error)
	GetForApp(domain.AppID) ([]*domain.Appspace, error)
	GetForAppVersion(appID domain.AppID, version domain.Version) ([]*domain.Appspace, error)
	Create(domain.UserID, domain.AppID, domain.Version, string, string) (*domain.Appspace, error)
	Pause(domain.AppspaceID, bool) error
	SetVersion(domain.AppspaceID, domain.Version) error
}

// AppspaceFilesModel manipulates data directories for appspaces
type AppspaceFilesModel interface {
	CreateLocation() (string, error)
}

// AppspaceInfoModels caches and dishes AppspaceInfoModels
type AppspaceInfoModels interface {
	Init()
	Get(domain.AppspaceID) domain.AppspaceInfoModel
	GetSchema(domain.AppspaceID) (int, error)
}

// AppspaceContactModel stores users of appspaces
type AppspaceContactModel interface {
	Create(userID domain.UserID, name string, displayName string) (domain.Contact, error)
	Update(userID domain.UserID, contactID domain.ContactID, name string, displayName string) error
	Delete(userID domain.UserID, contactID domain.ContactID) error
	Get(contactID domain.ContactID) (domain.Contact, error)
	GetForUser(userID domain.UserID) ([]domain.Contact, error)
	InsertAppspaceContact(appspaceID domain.AppspaceID, contactID domain.ContactID, proxyID domain.ProxyID) error
	DeleteAppspaceContact(appspaceID domain.AppspaceID, contactID domain.ContactID) error
	GetContactProxy(appspaceID domain.AppspaceID, contactID domain.ContactID) (domain.ProxyID, error)
	GetByProxy(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.ContactID, error)
	GetContactAppspaces(contactID domain.ContactID) ([]domain.AppspaceContact, error)
	GetAppspaceContacts(appspaceID domain.AppspaceID) ([]domain.AppspaceContact, error)
}

// MigrationJobModel handles writing jobs to the db
type MigrationJobModel interface {
	Create(domain.UserID, domain.AppspaceID, domain.Version, bool) (*domain.MigrationJob, error)
	GetJob(domain.JobID) (*domain.MigrationJob, error)
	GetPending() ([]*domain.MigrationJob, error)
	GetRunning() ([]domain.MigrationJob, error)
	SetStarted(domain.JobID) (bool, error)
	SetFinished(domain.JobID, nulltypes.NullString) error
	GetForAppspace(appspaceID domain.AppspaceID) ([]*domain.MigrationJob, error)
	// Delete(AppspaceID) Error
}
