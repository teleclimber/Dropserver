package appspacedb

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

//AppspaceDB holds various api versions of AppspaceDB
type AppspaceDB struct {
	Config *domain.RuntimeConfig
	V0     *V0
}

// Init creates the versions of appspace db
func (a *AppspaceDB) Init() {
	// a connManager could and should be shared by all versions
	connManager := &ConnManager{}
	connManager.Init(a.Config.Exec.AppspacesPath)

	a.V0 = &V0{
		connManager: connManager,
	}
}
