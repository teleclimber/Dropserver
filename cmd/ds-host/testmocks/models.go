package testmocks

import (
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

//go:generate mockgen -destination=models_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks CookieModel,UserModel,SettingsModel,UserInvitationModel,AppFilesModel,AppModel,AppspaceModel,RemoteAppspaceModel,AppspaceFilesModel,AppspaceInfoModels,AppspaceUserModel,ContactModel,DropIDModel,MigrationJobModel

type CookieModel interface {
	Get(cookieID string) (domain.Cookie, error)
	Create(domain.Cookie) (string, error)
	UpdateExpires(cookieID string, exp time.Time) error
	Delete(cookieID string) error
}

type UserModel interface {
	Create(email, password string) (domain.User, error)
	UpdatePassword(userID domain.UserID, password string) error
	GetFromID(userID domain.UserID) (domain.User, error)
	GetFromEmail(email string) (domain.User, error)
	GetFromEmailPassword(email, password string) (domain.User, error)
	GetAll() ([]domain.User, error)
	IsAdmin(userID domain.UserID) bool
	GetAllAdmins() ([]domain.UserID, error)
	MakeAdmin(userID domain.UserID) error
	DeleteAdmin(userID domain.UserID) error
}

// SettingsModel is used to get and set settings
type SettingsModel interface {
	Get() (domain.Settings, error)
	Set(domain.Settings) error
	SetRegistrationOpen(bool) error
}

// UserInvitationModel is the interface to the UserInvitation model
type UserInvitationModel interface {
	PrepareStatements()
	GetAll() ([]domain.UserInvitation, error)
	Get(email string) (domain.UserInvitation, error)
	Create(email string) error
	Delete(email string) error
}

// AppFilesModel represents the application's files saved to disk
type AppFilesModel interface {
	Save(*map[string][]byte) (string, error)
	ReadMeta(string) (*domain.AppFilesMetadata, error)
	WriteRoutes(locationKey string, routesData []byte) error
	ReadRoutes(locationKey string) ([]byte, error)
	Delete(string) error
}

// AppModel is the interface for the app model
type AppModel interface {
	GetFromID(domain.AppID) (*domain.App, error)
	GetForOwner(domain.UserID) ([]*domain.App, error)
	Create(domain.UserID, string) (*domain.App, error)
	Delete(appID domain.AppID) error
	GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
	GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	CreateVersion(domain.AppID, domain.Version, int, domain.APIVersion, string) (*domain.AppVersion, error)
	DeleteVersion(domain.AppID, domain.Version) error
}

// AppspaceModel is the interface for the appspace model
type AppspaceModel interface {
	GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	GetFromDomain(string) (*domain.Appspace, error)
	GetForOwner(domain.UserID) ([]*domain.Appspace, error)
	GetForApp(domain.AppID) ([]*domain.Appspace, error)
	GetForAppVersion(appID domain.AppID, version domain.Version) ([]*domain.Appspace, error)
	Create(domain.Appspace) (*domain.Appspace, error)
	Pause(domain.AppspaceID, bool) error
	SetVersion(domain.AppspaceID, domain.Version) error
	Delete(domain.AppspaceID) error
}

// RemoteAppspaceModel is the inrweface for remote appspace model
type RemoteAppspaceModel interface {
	Get(userID domain.UserID, domainName string) (domain.RemoteAppspace, error)
	GetForUser(userID domain.UserID) ([]domain.RemoteAppspace, error)
	Create(userID domain.UserID, domainName string, ownerDropID string, dropID string) error
	Delete(userID domain.UserID, domainName string) error
}

// AppspaceFilesModel manipulates data directories for appspaces
type AppspaceFilesModel interface {
	CreateLocation() (string, error)
	//DeleteLocation(string) error
}

// AppspaceInfoModels caches and dishes AppspaceInfoModels
type AppspaceInfoModels interface {
	Init()
	Get(domain.AppspaceID) domain.AppspaceInfoModel
	GetSchema(domain.AppspaceID) (int, error)
}

// AppspaceUserModel stores the users of an appspace
type AppspaceUserModel interface {
	Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
	GetByDropID(appspaceID domain.AppspaceID, dropID string) (domain.AppspaceUser, error)
	GetForAppspace(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
	UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, permissions []string) error
	Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error
	DeleteForAppspace(appspaceID domain.AppspaceID) error
}

// ContactModel stores a user's contacts
type ContactModel interface {
	Create(userID domain.UserID, name string, displayName string) (domain.Contact, error)
	Update(userID domain.UserID, contactID domain.ContactID, name string, displayName string) error
	Delete(userID domain.UserID, contactID domain.ContactID) error
	Get(contactID domain.ContactID) (domain.Contact, error)
	GetForUser(userID domain.UserID) ([]domain.Contact, error)
	// InsertAppspaceContact(appspaceID domain.AppspaceID, contactID domain.ContactID, proxyID domain.ProxyID) error
	// DeleteAppspaceContact(appspaceID domain.AppspaceID, contactID domain.ContactID) error
	// GetContactProxy(appspaceID domain.AppspaceID, contactID domain.ContactID) (domain.ProxyID, error)
	// GetByProxy(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.ContactID, error)
	// GetContactAppspaces(contactID domain.ContactID) ([]domain.AppspaceContact, error)
	// GetAppspaceContacts(appspaceID domain.AppspaceID) ([]domain.AppspaceContact, error)
}

// DropIDModel CRUD ops for a user's DropIDs.
type DropIDModel interface {
	Create(userID domain.UserID, handle string, dom string, displayName string) (domain.DropID, error)
	Update(userID domain.UserID, handle string, dom string, displayName string) (domain.DropID, error)
	Get(handle string, dom string) (domain.DropID, error)
	GetForUser(userID domain.UserID) ([]domain.DropID, error)
	Delete(userID domain.UserID, handle string, dom string) error
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
	DeleteForAppspace(appspaceID domain.AppspaceID) error
}
