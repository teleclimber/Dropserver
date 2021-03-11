package appspacemetadb

import (
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// TODO is this deprecated in favor of appspace user model on host?

// AppspaceUserModels can return a user model for a given appspace id
type AppspaceUserModels struct {
	Config         *domain.RuntimeConfig //unused?
	AppspaceMetaDB interface {
		GetConn(domain.AppspaceID) (domain.DbConn, error)
	}

	modelsMux sync.Mutex
	modelsV0  map[domain.AppspaceID]*V0UserModel // maybe make that an interface for testing purposes.
}

// Init the data structures as necessary
func (g *AppspaceUserModels) Init() {
	g.modelsV0 = make(map[domain.AppspaceID]*V0UserModel)
}

// GetV0 returns the route model for the appspace
// There i a single RouteModel per appspaceID so that caching can be implemented in it.
// There will be different route model versions!
func (g *AppspaceUserModels) GetV0(appspaceID domain.AppspaceID) domain.V0UserModel {
	g.modelsMux.Lock()
	defer g.modelsMux.Unlock()

	var rm *V0UserModel

	rm, ok := g.modelsV0[appspaceID]
	if !ok {
		rm = &V0UserModel{
			AppspaceMetaDB: g.AppspaceMetaDB,
			appspaceID:     appspaceID,
		}
		g.modelsV0[appspaceID] = rm
	}

	return rm
}
