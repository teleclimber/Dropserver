package server

import (
	"errors"
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
	AppspaceTSNetModel interface {
		GetAllConnect() ([]domain.AppspaceTSNet, error)
		Get(domain.AppspaceID) (domain.AppspaceTSNet, error)
	} `checkinject:"required"`
	AppspaceTSNetStatusEvents interface {
		Send(data domain.TSNetAppspaceStatus)
	} `checkinject:"required"`
	AppspaceTSNetPeersEvents interface {
		Send(data domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceLocation2Path interface {
		TailnetNodeStore(locationKey string) string
	} `checkinject:"required"`

	serversMux sync.Mutex
	servers    map[domain.AppspaceID]*TSNetNode
}

func (a *AppspaceTSNet) StopAll() { // maybe rename to Shutdown to convey we're de-initializing
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	wg := sync.WaitGroup{}
	for _, s := range a.servers {
		wg.Add(1)
		go func(appspaceNode *TSNetNode) {
			appspaceNode.stop()
			wg.Done()
		}(s)
	}
	wg.Wait()
}

func (a *AppspaceTSNet) Init() {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	a.servers = make(map[domain.AppspaceID]*TSNetNode)
}

func (a *AppspaceTSNet) Create(appspaceID domain.AppspaceID, config domain.TSNetCreateConfig) error {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	_, exists := a.servers[appspaceID]
	if exists {
		err := errors.New("unable to create server as it exists already")
		a.getLogger("Create").AppspaceID(appspaceID).Error(err)
		return err
	}

	appspace, err := a.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		a.getLogger("Create AppspaceModel.GetFromID()").AppspaceID(appspaceID).Error(err)
		return err
	}
	node := a.makeNodeStruct(*appspace)
	a.servers[appspaceID] = node
	return node.createTailnetNode(config)
}

func (a *AppspaceTSNet) Connect(appspaceID domain.AppspaceID) error {
	tsnetData, err := a.AppspaceTSNetModel.Get(appspaceID)
	if err == domain.ErrNoRowsInResultSet {
		a.getLogger("Connect").AppspaceID(appspaceID).Error(errors.New("no tsnet data in db"))
	}
	if err != nil {
		return err
	}
	err = a.start(tsnetData)
	if err != nil {
		a.getLogger("Connect start").AppspaceID(appspaceID).Error(err)
	}
	return err
}

func (a *AppspaceTSNet) Disconnect(appspaceID domain.AppspaceID) {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	node, exists := a.servers[appspaceID]
	if exists {
		node.stop()
	}
}

func (a *AppspaceTSNet) Delete(appspaceID domain.AppspaceID) error {
	a.serversMux.Lock()
	node, exists := a.servers[appspaceID]
	if exists {
		delete(a.servers, appspaceID)
	}
	a.serversMux.Unlock()

	if !exists {
		appspace, err := a.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			a.getLogger("Create AppspaceModel.GetFromID()").AppspaceID(appspaceID).Error(err)
			return err
		}
		node = a.makeNodeStruct(*appspace)
	}
	return node.delete()
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

func (a *AppspaceTSNet) start(tsnetData domain.AppspaceTSNet) error {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	node, exists := a.servers[tsnetData.AppspaceID]
	if !exists {
		appspace, err := a.AppspaceModel.GetFromID(tsnetData.AppspaceID)
		if err != nil {
			a.getLogger("Create AppspaceModel.GetFromID()").AppspaceID(tsnetData.AppspaceID).Error(err)
			return err
		}
		node = a.makeNodeStruct(*appspace)
		a.servers[tsnetData.AppspaceID] = node
	}
	return node.connect(tsnetData.TSNetCommon)
}

func (a *AppspaceTSNet) get(appspaceID domain.AppspaceID) *TSNetNode {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	if node, exists := a.servers[appspaceID]; exists {
		return node
	}
	return nil
}

func (a *AppspaceTSNet) addGet(appspace domain.Appspace) *TSNetNode {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	if node, exists := a.servers[appspace.AppspaceID]; exists {
		return node
	}
	a.servers[appspace.AppspaceID] = a.makeNodeStruct(appspace)
	return a.servers[appspace.AppspaceID]
}

type statusEventsAdapter struct {
	appspaceTSNetStatusEvents interface {
		Send(data domain.TSNetAppspaceStatus)
	}
	appspaceID domain.AppspaceID
}

func (t *statusEventsAdapter) Send(tsnetStatus domain.TSNetStatus) {
	t.appspaceTSNetStatusEvents.Send(domain.TSNetAppspaceStatus{
		TSNetStatus: tsnetStatus,
		AppspaceID:  t.appspaceID})
}

type peersEventsAdapter struct {
	appspaceTSNetPeersEvents interface {
		Send(data domain.AppspaceID)
	}
	appspaceID domain.AppspaceID
}

func (t *peersEventsAdapter) Send() {
	t.appspaceTSNetPeersEvents.Send(t.appspaceID)
}

func (a *AppspaceTSNet) makeNodeStruct(appspace domain.Appspace) *TSNetNode {
	return &TSNetNode{
		Config: a.Config,
		Router: a.AppspaceRouter,
		TSNetStatusEvents: &statusEventsAdapter{
			appspaceTSNetStatusEvents: a.AppspaceTSNetStatusEvents,
			appspaceID:                appspace.AppspaceID},
		TSNetPeersEvents: &peersEventsAdapter{
			appspaceTSNetPeersEvents: a.AppspaceTSNetPeersEvents,
			appspaceID:               appspace.AppspaceID},
		hasAppspaceID: true,
		appspaceID:    appspace.AppspaceID,
		tsnetDir:      a.AppspaceLocation2Path.TailnetNodeStore(appspace.LocationKey),
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
		status := node.getStatus()
		return domain.TSNetAppspaceStatus{
			TSNetStatus: status,
			AppspaceID:  appspaceID,
		}
	}
	return domain.TSNetAppspaceStatus{
		TSNetStatus: domain.TSNetStatus{
			State: "Off",
		},
		AppspaceID: appspaceID,
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
