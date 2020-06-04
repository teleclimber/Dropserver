package appspacemetadb

import (
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

// AppspaceRouteModels can return a routes model for a given appspace id
type AppspaceRouteModels struct {
	Config         *domain.RuntimeConfig
	Validator      domain.Validator
	AppspaceMetaDB domain.AppspaceMetaDB

	modelsMux sync.Mutex
	modelsV0  map[domain.AppspaceID]*RouteModelV0 // maybe make that an interface for testing purposes.
}

// GetV0 returns the route model for the appspace
// There i a single RouteModel per appspaceID so that caching can be implemented in it.
// There will be different route model versions!
func (g *AppspaceRouteModels) GetV0(appspaceID domain.AppspaceID) domain.RouteModelV0 {
	g.modelsMux.Lock()
	defer g.modelsMux.Unlock()

	var rm *RouteModelV0

	rm, ok := g.modelsV0[appspaceID]
	if !ok {
		// make it and add it
		rm = &RouteModelV0{
			Validator:      g.Validator,
			AppspaceMetaDB: g.AppspaceMetaDB,
			appspaceID:     appspaceID,
		}
		g.modelsV0[appspaceID] = rm
	}

	return rm
}

// Command can process a command from the reverse listener
// It determines the right version to use based on the passed appspace
func (g *AppspaceRouteModels) Command(appspace *domain.Appspace, message twine.ReceivedMessageI) {
	// from appspace you should get the right API version
	// ..but for now just use V0
	v0 := g.GetV0(appspace.AppspaceID)
	v0.ReverseCommand(message)

}
