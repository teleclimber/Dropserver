package appspacemetadb

import (
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// AppspaceRouteModels can return a routes model for a given appspace id
type AppspaceRouteModels struct {
	Config         *domain.RuntimeConfig
	AppspaceMetaDB interface {
		GetConn(domain.AppspaceID) (domain.DbConn, error)
	}

	RouteEvents interface {
		Send(domain.AppspaceID, domain.AppspaceRouteEvent)
	}

	modelsMux sync.Mutex
	modelsV0  map[domain.AppspaceID]*V0RouteModel // maybe make that an interface for testing purposes.
}

// Init the data structures as necessary
func (g *AppspaceRouteModels) Init() {
	g.modelsV0 = make(map[domain.AppspaceID]*V0RouteModel)
}

// GetV0 returns the route model for the appspace
// There i a single RouteModel per appspaceID so that caching can be implemented in it.
// There will be different route model versions!
func (g *AppspaceRouteModels) GetV0(appspaceID domain.AppspaceID) domain.V0RouteModel {
	g.modelsMux.Lock()
	defer g.modelsMux.Unlock()

	var rm *V0RouteModel

	rm, ok := g.modelsV0[appspaceID]
	if !ok {
		rm = &V0RouteModel{
			AppspaceMetaDB: g.AppspaceMetaDB,
			RouteEvents:    g.RouteEvents,
			appspaceID:     appspaceID,
		}
		g.modelsV0[appspaceID] = rm
	}

	return rm
}
