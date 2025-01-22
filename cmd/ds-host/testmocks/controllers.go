package testmocks

import (
	"io"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=controllers_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks SetupKey,RemoteAppGetter,DeleteApp,BackupAppspace,RestoreAppspace,MigrationJobController,AppspaceStatus,AppspaceTSNet,AppspaceRouter

type SetupKey interface {
	Has() (bool, error)
	Get() (string, error)
	Delete() error
}

type RemoteAppGetter interface {
	FetchValidListing(url string) (domain.AppListingFetch, error)
	FetchPackageJob(url string) (string, error)
}

type DeleteApp interface {
	Delete(appID domain.AppID) error
	DeleteVersion(appID domain.AppID, version domain.Version) error
}

type BackupAppspace interface {
	CreateBackup(appspaceID domain.AppspaceID) (string, error)
	BackupNoPause(appspaceID domain.AppspaceID) (string, error)
}

type RestoreAppspace interface {
	Prepare(reader io.Reader) (string, error)
	PrepareBackup(appspaceID domain.AppspaceID, backupFile string) (string, error)
	CheckAppspaceDataValid(tok string) error
	GetMetaInfo(tok string) (domain.AppspaceMetaInfo, error)
	ReplaceData(tok string, appspaceID domain.AppspaceID) error
}

// MigrationJobController controls and tracks appspace migration jobs
type MigrationJobController interface {
	Start()
	Stop()
	WakeUp()
}

// AppspaceStatus for mocking
type AppspaceStatus interface {
	SetHostStop(stop bool)
	Ready(appspaceID domain.AppspaceID) bool
	Get(appspaceID domain.AppspaceID) domain.AppspaceStatusEvent
	WaitTempPaused(appspaceID domain.AppspaceID, reason string) chan struct{}
	IsTempPaused(appspaceId domain.AppspaceID) bool
	WaitStopped(appspaceID domain.AppspaceID)
	LockClosed(appspaceID domain.AppspaceID) (chan struct{}, bool)
	IsLockedClosed(appspaceID domain.AppspaceID) bool
}

type AppspaceTSNet interface {
	GetStatus(domain.AppspaceID) domain.TSNetAppspaceStatus
	GetPeerUsers(domain.AppspaceID) []domain.TSNetPeerUser
}

// AppspaceRouter is a route handler that also tracks ongoing requests
// for each appspace ID.
type AppspaceRouter interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	SubscribeLiveCount(domain.AppspaceID, chan<- int) int
	UnsubscribeLiveCount(domain.AppspaceID, chan<- int)
}
