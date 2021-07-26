package vxservices

import (
	"encoding/json"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/twine"
)

const (
	getUserCmd     = 12
	getAllUsersCmd = 13
)

// UsersV0 responds to requests about appspace users for the appspace
type UsersV0 struct {
	AppspaceUsersV0 interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	}
	appspaceID domain.AppspaceID
}

// service HandleMessage is for sandboxed code.
// Not frontend or anything. ...I think?
// host frontend and other systems will use controllers that will call regular model methods.

// HandleMessage processes a command and payload from the reverse listener
func (u *UsersV0) HandleMessage(message twine.ReceivedMessageI) {
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

func (u *UsersV0) handleGetUserCommand(message twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(message.Payload()))
	user, err := u.AppspaceUsersV0.Get(u.appspaceID, proxyID)
	if err != nil {
		message.SendError(err.Error())
		return
	}
	if user.ProxyID == "" {
		message.Reply(13, nil)
	} else {
		bytes, err := json.Marshal(user)
		if err != nil {
			u.getLogger("handleGetUserCommand(), json Marshal error").Error(err)
			message.SendError("Error on host")
			return
		}
		message.Reply(14, bytes)
	}
}

func (u *UsersV0) handleGetAllUsersCommand(message twine.ReceivedMessageI) {
	users, err := u.AppspaceUsersV0.GetAll(u.appspaceID)
	if err != nil {
		message.SendError(err.Error())
		return
	}

	bytes, err := json.Marshal(users)
	if err != nil {
		u.getLogger("handleGetAllUsersCommand(), json Marshal error").Error(err)
		message.SendError("Error on host")
	}
	message.Reply(14, bytes)
}

func (u *UsersV0) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("V0User")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
