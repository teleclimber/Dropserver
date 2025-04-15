package server

import (
	"errors"
	"net/http"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type UserTSNet struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	SettingsModel interface {
		GetTSNet() (domain.TSNetCommon, error)
	} `checkinject:"required"`
	UserRoutes        http.Handler `checkinject:"required"`
	TSNetStatusEvents interface {
		Send(data domain.TSNetStatus)
	} `checkinject:"required"`
	TSNetPeersEvents interface {
		Send()
	} `checkinject:"required"`

	serverMux sync.Mutex
	server    *TSNetNode
}

func (u *UserTSNet) Create(config domain.TSNetCreateConfig) error {
	u.serverMux.Lock()
	defer u.serverMux.Unlock()
	if u.server != nil {
		err := errors.New("unable to create server as it exists already")
		u.getLogger("Create").Error(err)
		return err
	}
	u.server = u.makeNodeStruct()
	return u.server.createTailnetNode(config)
}

// Connect attempts to start the tsnet node.
// if expectConnect is true: a missing config or Connect of false are errors
// if expectConnect is false: missing or false just means don't connect
func (u *UserTSNet) Connect(expectConnect bool) error {
	settingsTSNet, err := u.SettingsModel.GetTSNet()
	if err != nil {
		return err
	}
	if settingsTSNet.Hostname == "" {
		if expectConnect {
			err = errors.New("no config set for tsnet")
			u.getLogger("Connect").Error(err)
			return err
		}
		return nil
	}
	if !settingsTSNet.Connect {
		if expectConnect {
			err = errors.New("called connect while config says don't connect")
			u.getLogger("Connect").Error(err)
			return err
		}
		return nil
	}
	u.serverMux.Lock()
	defer u.serverMux.Unlock()
	if u.server == nil {
		u.server = u.makeNodeStruct()
	}
	err = u.server.connect(settingsTSNet)
	if err != nil {
		u.getLogger("Connect server.connect").Error(err)
	}
	return err
}

func (u *UserTSNet) Disconnect() {
	u.serverMux.Lock()
	defer u.serverMux.Unlock()
	if u.server != nil {
		go u.server.stop() // TODO not coverd by lock! Need a sync call to a stop function
	}
}

func (u *UserTSNet) Delete() error {
	u.serverMux.Lock()
	defer u.serverMux.Unlock()
	serv := u.server
	u.server = nil
	if serv == nil {
		serv = u.makeNodeStruct()
	}
	return serv.delete()
}

func (u *UserTSNet) GetStatus() domain.TSNetStatus {
	u.serverMux.Lock()
	defer u.serverMux.Unlock()
	if u.server != nil {
		return u.server.getStatus()
	}
	return domain.TSNetStatus{
		State: "Off",
	}
}

func (u *UserTSNet) GetPeerUsers() []domain.TSNetPeerUser {
	u.serverMux.Lock()
	defer u.serverMux.Unlock()
	if u.server != nil {
		return u.server.getPeerUsers()
	}
	return []domain.TSNetPeerUser{}
}

func (u *UserTSNet) makeNodeStruct() *TSNetNode {
	return &TSNetNode{
		Config:            u.Config,
		Router:            u.UserRoutes,
		TSNetStatusEvents: u.TSNetStatusEvents,
		TSNetPeersEvents:  u.TSNetPeersEvents,
		tsnetDir:          u.Config.Exec.UserTSNetPath,
	}
}

func (u *UserTSNet) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("UserTSNet")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
