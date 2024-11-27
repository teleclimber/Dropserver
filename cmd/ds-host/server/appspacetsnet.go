package server

// not clear if "server" is the right package for this. Proba need its own pacakge.

import (
	"net/http"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type AppspaceTSNet struct {
	Config         *domain.RuntimeConfig `checkinject:"required"`
	AppspaceRouter http.Handler          `checkinject:"required"`
	AppspaceModel  interface {
		GetAll() ([]domain.Appspace, error)
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceTSNetStatusEvents interface {
		Send(data domain.TSNetAppspaceStatus)
	} `checkinject:"required"`
	AppspaceLocation2Path interface {
		TailscaleNodeStore(locationKey string) string
	} `checkinject:"required"`

	serversMux sync.Mutex
	servers    map[domain.AppspaceID]*AppspaceTSNode
}

func (a *AppspaceTSNet) StopAll() {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	wg := sync.WaitGroup{}
	for _, s := range a.servers {
		wg.Add(1)
		go func(appspaceNode *AppspaceTSNode) {
			appspaceNode.stop()
			wg.Done()
		}(s)
	}
	wg.Wait()
}

func (a *AppspaceTSNet) Init() {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	a.servers = make(map[domain.AppspaceID]*AppspaceTSNode)
}

func (a *AppspaceTSNet) StartAll() error {
	appspaces, err := a.AppspaceModel.GetAll()
	if err != nil {
		return err
	}
	for _, as := range appspaces {
		if !as.Paused { // this might be temporary? Should Pausing an appspace cause its TS server to stop?
			go func(appspace domain.Appspace) {
				node := a.addGet(appspace.AppspaceID)
				node.ownerID = appspace.OwnerID      // needed for notifications.
				node.createNode(appspace.DomainName) // TODO as.DomainName is tempoary.
			}(as)
		}
	}
	return nil
}

func (a *AppspaceTSNet) get(appspaceID domain.AppspaceID) *AppspaceTSNode {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	if node, exists := a.servers[appspaceID]; exists {
		return node
	}
	return nil
}

func (a *AppspaceTSNet) addGet(appspaceID domain.AppspaceID) *AppspaceTSNode {
	// probably need to panic or somehow handle if a server is already present for that appspace ID.
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	if node, exists := a.servers[appspaceID]; exists {
		//panic("server already exists for appspace id. Handle this better")
		return node
	}
	a.servers[appspaceID] = &AppspaceTSNode{
		Config:                    a.Config,
		AppspaceLocation2Path:     a.AppspaceLocation2Path,
		AppspaceModel:             a.AppspaceModel,
		AppspaceRouter:            a.AppspaceRouter,
		AppspaceTSNetStatusEvents: a.AppspaceTSNetStatusEvents,
		appspaceID:                appspaceID,
	}
	return a.servers[appspaceID]
}

func (a *AppspaceTSNet) rmServer(appspaceID domain.AppspaceID) {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	delete(a.servers, appspaceID)
}

func (a *AppspaceTSNet) GetStatus(appspaceID domain.AppspaceID) domain.TSNetAppspaceStatus {
	node := a.get(appspaceID)
	if node != nil {
		return node.getStatus()
	}
	return domain.TSNetAppspaceStatus{
		AppspaceID: appspaceID,
		// No need to populate owner since this is not sent via event system
		State: "Off",
	}
}

func (m *AppspaceTSNet) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceTSNet")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
