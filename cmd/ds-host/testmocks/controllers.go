package testmocks

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=controllers_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks MigrationJobController,AppspaceStatus,AppspaceRoutes

// MigrationJobController controls and tracks appspace migration jobs
type MigrationJobController interface {
	Start()
	Stop()
	WakeUp()
	GetRunningJobs() []domain.MigrationStatusData
}

// AppspaceStatus for mocking
type AppspaceStatus interface {
	SetHostStop(stop bool)
	Ready(appspaceID domain.AppspaceID) bool
	WaitStopped(appspaceID domain.AppspaceID)
}

// AppspaceRoutes is a route handler that also tracks ongoing requests
// for each appspace ID.
type AppspaceRoutes interface {
	ServeHTTP(http.ResponseWriter, *http.Request, *domain.AppspaceRouteData)
	SubscribeLiveCount(domain.AppspaceID, chan<- int) int
	UnsubscribeLiveCount(domain.AppspaceID, chan<- int)
}
