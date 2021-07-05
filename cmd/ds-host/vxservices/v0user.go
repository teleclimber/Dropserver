package vxservices

import (
	"encoding/json"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/twine"
)

type VxUserModels struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetForAppspace(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	}
}

func (m *VxUserModels) GetV0(appspaceID domain.AppspaceID) *V0UserModel {
	return &V0UserModel{
		AppspaceUserModel: m.AppspaceUserModel,
		appspaceID:        appspaceID,
	}
}

const (
	getUserCmd     = 12
	getAllUsersCmd = 13
)

// V0UserModel responds to requests about appspace users for the appspace
type V0UserModel struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetForAppspace(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	}

	appspaceID domain.AppspaceID
}

// service HandleMessage is for sandboxed code.
// Not frontend or anything. ...I think?
// host frontend and other systems will use controllers that will call regular model methods.

// HandleMessage processes a command and payload from the reverse listener
func (m *V0UserModel) HandleMessage(message twine.ReceivedMessageI) {
	switch message.CommandID() {
	case getUserCmd:
		// from proxy id fetch user's name and permissions
		// and figure out if they are owner or not.
		m.handleGetUserCommand(message)
	case getAllUsersCmd:
		// get all users for the appspace
		m.handleGetAllUsersCommand(message)
	default:
		message.SendError("Command not recognized")
	}
}

func (m *V0UserModel) handleGetUserCommand(message twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(message.Payload()))
	user, err := m.AppspaceUserModel.Get(m.appspaceID, proxyID)
	if err != nil {
		message.SendError(err.Error())
		return
	}
	if user.ProxyID == "" {
		message.Reply(13, nil)
	} else {
		bytes, err := json.Marshal(user)
		if err != nil {
			m.getLogger("handleGetUserCommand(), json Marshal error").Error(err)
			message.SendError("Error on host")
		}
		message.Reply(14, bytes)
	}
}

func (m *V0UserModel) handleGetAllUsersCommand(message twine.ReceivedMessageI) {
	users, err := m.AppspaceUserModel.GetForAppspace(m.appspaceID)
	if err != nil {
		message.SendError(err.Error())
		return
	}

	bytes, err := json.Marshal(users)
	if err != nil {
		m.getLogger("handleGetAllUsersCommand(), json Marshal error").Error(err)
		message.SendError("Error on host")
	}
	message.Reply(14, bytes)
}

func (m *V0UserModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("V0User")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
