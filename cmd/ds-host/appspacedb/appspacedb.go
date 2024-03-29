package appspacedb

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

//AppspaceDB holds various api versions of AppspaceDB
type AppspaceDB struct {
	Config *domain.RuntimeConfig `checkinject:"required"`
	V0     *V0

	connManager *ConnManager
}

// shouldappspace db check with appspace status for IsLockedClosed?
// Or not necessary because appspace db is usually only opened by a route or whatever?
// BUT! What about if there is a db explorer in ds-host UI as part of appspace observabilitiy?
// this would no doubt case the appspace db to open.

// Init creates the versions of appspace db
func (a *AppspaceDB) Init() {
	// a connManager could and should be shared by all versions
	a.connManager = &ConnManager{}
	a.connManager.Init(a.Config.Exec.AppspacesPath)

	a.V0 = &V0{
		connManager: a.connManager,
	}
}

// CloseAppspace closes all appsace DBs
func (a *AppspaceDB) CloseAppspace(appspaceID domain.AppspaceID) {
	a.connManager.closeAppspaceDBs(appspaceID)
}
