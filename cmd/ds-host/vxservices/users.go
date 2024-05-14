package sandboxservices

import (
	"encoding/json"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
)

const (
	getUserCmd     = 12
	getAllUsersCmd = 13
)

type SandboxUser struct {
	ProxyID     domain.ProxyID `json:"proxyId"`
	DisplayName string         `json:"displayName"`
	Avatar      string         `json:"avatar"`
}

// UsersService responds to requests about appspace users for the appspace
type UsersService struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	}
	appspaceID domain.AppspaceID
}

// HandleMessage processes a command and payload from the reverse listener
func (u *UsersService) HandleMessage(message twine.ReceivedMessageI) {
	switch message.CommandID() {
	case getUserCmd:
		// from proxy id fetch user's name and permissions
		// and figure out if they are owner or not.
		u.handleGetUserCommand(message)
	case getAllUsersCmd:
		// get all users for the appspace
		u.handleGetAllUsersCommand(message)
	default:
		message.SendError("Command not recognized")
	}
}

func (u *UsersService) handleGetUserCommand(message twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(message.Payload()))
	user, err := u.AppspaceUserModel.Get(u.appspaceID, proxyID)
	if err != nil {
		message.SendError(err.Error())
		return
	}
	if user.ProxyID == "" {
		message.Reply(13, nil)
	} else {
		bytes, err := json.Marshal(getUserForSandbox(user))
		if err != nil {
			u.getLogger("handleGetUserCommand(), json Marshal error").Error(err)
			message.SendError("Error on host")
			return
		}
		message.Reply(14, bytes)
	}
}

func (u *UsersService) handleGetAllUsersCommand(message twine.ReceivedMessageI) {
	users, err := u.AppspaceUserModel.GetAll(u.appspaceID)
	if err != nil {
		message.SendError(err.Error())
		return
	}
	sUsers := make([]SandboxUser, len(users))
	for i, u := range users {
		sUsers[i] = getUserForSandbox(u)
	}
	bytes, err := json.Marshal(sUsers)
	if err != nil {
		u.getLogger("handleGetAllUsersCommand(), json Marshal error").Error(err)
		message.SendError("Error on host")
	}
	message.Reply(14, bytes)
}

func (u *UsersService) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("UsersService")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

func getUserForSandbox(user domain.AppspaceUser) SandboxUser {
	return SandboxUser{
		ProxyID:     user.ProxyID,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
	}
}
