package server

// not clear if "server" is the right package for this. Proba need its own pacakge.

import (
	"net/http"
	"os"
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
	AppspaceTSNetModel interface {
		GetAllConnect() ([]domain.AppspaceTSNet, error)
		Get(domain.AppspaceID) (domain.AppspaceTSNet, error)
	} `checkinject:"required"`
	AppspaceTSNetModelEvents interface {
		Subscribe() <-chan domain.AppspaceTSNetModelEvent
		Unsubscribe(ch <-chan domain.AppspaceTSNetModelEvent)
	} `checkinject:"required"`
	AppspaceTSNetStatusEvents interface {
		Send(data domain.TSNetAppspaceStatus)
	} `checkinject:"required"`
	AppspaceTSNetPeersEvents interface {
		Send(data domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceLocation2Path interface {
		TailscaleNodeStore(locationKey string) string
	} `checkinject:"required"`

	tsnetModelEventsChan <-chan domain.AppspaceTSNetModelEvent

	serversMux sync.Mutex
	servers    map[domain.AppspaceID]*AppspaceTSNode
}

func (a *AppspaceTSNet) StopAll() { // maybe rename to Shutdown to convey we're de-initializing
	a.AppspaceTSNetModelEvents.Unsubscribe(a.tsnetModelEventsChan)
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
	a.tsnetModelEventsChan = a.AppspaceTSNetModelEvents.Subscribe()

	go func() {
		for modelEvent := range a.tsnetModelEventsChan {
			go a.updateAppspace(modelEvent)
		}
	}()
}

func (a *AppspaceTSNet) updateAppspace(modelEvent domain.AppspaceTSNetModelEvent) {
	config := tsNodeConfig{
		controlURL: modelEvent.BackendURL,
		hostname:   modelEvent.Hostname,
		connect:    !modelEvent.Deleted && modelEvent.Connect,
	}

	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	node, exists := a.servers[modelEvent.AppspaceID]
	if exists {
		if node.deleteNode {
			// no-op: if node is deleting, let it delete. We call StartAppspcae at the end to re-create node if necessary
		} else if modelEvent.Deleted {
			a.getLogger("updateAppspace").AppspaceID(modelEvent.AppspaceID).Debug("model data deleted, deleting node and files")
			node.deleteNode = true
			go func(n *AppspaceTSNode) {
				n.stop()
				err := os.RemoveAll(n.tsnetDir)
				if err != nil {
					a.getLogger("updateAppspace").AppspaceID(modelEvent.AppspaceID).AddNote("deleting node files: os.RemoveAll()").Error(err)
				}
				a.serversMux.Lock()
				defer a.serversMux.Unlock()
				delete(a.servers, n.appspaceID)
				go a.StartAppspace(n.appspaceID) // Try to start it again in case user re-created data during delete
			}(node)
		} else {
			go node.setConfig(config)
		}
	} else if !exists && config.connect {
		appspace, err := a.AppspaceModel.GetFromID(modelEvent.AppspaceID)
		if err != nil {
			a.getLogger("updateAppspace AppspaceModel.GetFromID()").AppspaceID(modelEvent.AppspaceID).Error(err)
			return
		}
		node = a.makeNodeStruct(*appspace)
		a.servers[modelEvent.AppspaceID] = node
		go node.setConfig(config)
	}
}

func (a *AppspaceTSNet) StartAll() {
	tsnets, err := a.AppspaceTSNetModel.GetAllConnect()
	if err != nil {
		return
	}

	for _, n := range tsnets {
		a.start(n)
	}
}

func (a *AppspaceTSNet) StartAppspace(appspaceID domain.AppspaceID) error {
	tsnetData, err := a.AppspaceTSNetModel.Get(appspaceID)
	if err != nil {
		return err
	}
	return a.start(tsnetData)
}

func (a *AppspaceTSNet) start(tsnetData domain.AppspaceTSNet) error {
	appspace, err := a.AppspaceModel.GetFromID(tsnetData.AppspaceID)
	if err != nil {
		a.getLogger("start() GetFromID()").AppspaceID(tsnetData.AppspaceID).Error(err)
		return err
	}

	a.serversMux.Lock()
	defer a.serversMux.Unlock()

	node := a.makeNodeStruct(*appspace)
	a.servers[appspace.AppspaceID] = node
	go node.setConfig(tsNodeConfig{
		controlURL: tsnetData.BackendURL,
		hostname:   tsnetData.Hostname,
		connect:    tsnetData.Connect,
	})
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

func (a *AppspaceTSNet) addGet(appspace domain.Appspace) *AppspaceTSNode {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	if node, exists := a.servers[appspace.AppspaceID]; exists {
		return node
	}
	a.servers[appspace.AppspaceID] = a.makeNodeStruct(appspace)
	return a.servers[appspace.AppspaceID]
}

func (a *AppspaceTSNet) makeNodeStruct(appspace domain.Appspace) *AppspaceTSNode {
	return &AppspaceTSNode{
		Config:                    a.Config,
		AppspaceLocation2Path:     a.AppspaceLocation2Path,
		AppspaceModel:             a.AppspaceModel,
		AppspaceRouter:            a.AppspaceRouter,
		AppspaceTSNetStatusEvents: a.AppspaceTSNetStatusEvents,
		AppspaceTSNetPeersEvents:  a.AppspaceTSNetPeersEvents,
		appspaceID:                appspace.AppspaceID,
		tsnetDir:                  a.AppspaceLocation2Path.TailscaleNodeStore(appspace.LocationKey),
	}
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
		State:      "Off",
	}
}

func (a *AppspaceTSNet) GetPeerUsers(appspaceID domain.AppspaceID) []domain.TSNetPeerUser {
	node := a.get(appspaceID)
	if node != nil {
		return node.getPeerUsers()
	}
	return []domain.TSNetPeerUser{}
}

func (m *AppspaceTSNet) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceTSNet")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
