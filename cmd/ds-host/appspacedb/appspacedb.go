package appspacedb

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

//AppspaceDB holds various api versions of AppspaceDB
type AppspaceDB struct {
	Config *domain.RuntimeConfig
	V0     *V0

	connManager *ConnManager
}

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
