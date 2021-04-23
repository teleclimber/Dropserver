package testmocks

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=controllers_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks MigrationJobController,AppspaceStatus,AppspaceRouter

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
	Track(appspaceID domain.AppspaceID) domain.AppspaceStatusEvent
	WaitTempPaused(appspaceID domain.AppspaceID, reason string) chan struct{}
	IsTempPaused(appspaceId domain.AppspaceID) bool
	WaitStopped(appspaceID domain.AppspaceID)
	LockClosed(appspaceID domain.AppspaceID) (chan struct{}, bool)
	IsLockedClosed(appspaceID domain.AppspaceID) bool
}

// AppspaceRouter is a route handler that also tracks ongoing requests
// for each appspace ID.
type AppspaceRouter interface {
	ServeHTTP(http.ResponseWriter, *http.Request, *domain.AppspaceRouteData)
	SubscribeLiveCount(domain.AppspaceID, chan<- int) int
	UnsubscribeLiveCount(domain.AppspaceID, chan<- int)
}
