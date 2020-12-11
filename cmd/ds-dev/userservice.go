package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

// What do we need ?
// app files events and app files so we can load user permissions (later)
//

// UserService is a twine service that sets the desired user params
// and keeps the frontend up to date with app's declared permissions
type UserService struct {
	DevAuthenticator   *DevAuthenticator
	AppspaceUserModels interface {
		GetV0(domain.AppspaceID) domain.V0UserModel
	}
	DevAppspaceContactModel *DevAppspaceContactModel
}

// Start creates listeners and then shuts everything down when twine exits
func (u *UserService) Start(t *twine.Twine) {
	// send initial users down
	u.sendUsers(t)

	// send owner down if set
	u.sendOwner(t)

	// TODO [later] subscripe to app files changes and resend user permissions when changed

	// Wait for twine to close and shut it all down.
	t.WaitClose()

	fmt.Println("closing user service")

}

// outgoing commnds:
const (
	loadAllUsersCmd = 11
	loadOwnerCmd    = 12
)

func (u *UserService) sendUsers(twine *twine.Twine) {
	v0userModel := u.AppspaceUserModels.GetV0(appspaceID)
	users, err := v0userModel.GetAll()
	if err != nil {
		fmt.Println("sendUsers error getting users: " + err.Error())
	}

	bytes, err := json.Marshal(users)
	if err != nil {
		fmt.Println("sendUsers json Marshal Error: " + err.Error())
	}

	_, err = twine.SendBlock(userControlService, loadAllUsersCmd, bytes)
	if err != nil {
		fmt.Println("sendUsers SendBlock Error: " + err.Error())
	}
}

func (u *UserService) sendOwner(twine *twine.Twine) {
	proxyID := u.DevAppspaceContactModel.GetOwnerProxyID()

	_, err := twine.SendBlock(userControlService, loadOwnerCmd, []byte(proxyID))
	if err != nil {
		fmt.Println("sendUsers SendBlock Error: " + err.Error())
	}
}

type IncomingUser struct {
	ProxyID     domain.ProxyID `json:"proxy_id"`
	DisplayName string         `json:"display_name"`
	Permissions []string       `json:"permissions"`
}

// THis whole interaction must be rethought:
// - we now have a db of users, which ds-dev users should be able to CRUD against.
// - they can select which user they want to "be"
// - They may need to be able to set which user is the owner.
// Note that the users can not be changed by app code, so ds-dev is in full control here.
// ..no need for events, etc... just send all the users on load then CRUD on it.
// Kind of a bummer that the appspace doesn't know who the owner is?
// -> it's clear the appspace will have to include a dump of various data including who the owner is

// Commands:
// - CreateUser (displa name, permissions) ..implies generating proxy id
// - UpdateUser (proxy , display name, permissions)
// - DeleteUser (proxy)
// - SelectOwner (proxy) .. sets that in DevAppspaceContactsModel
// - SelectUser (proxy) .. sets that in DevAuth

// incoming commands
const (
	userCreateCmd      = 11
	userUpdateCmd      = 12
	userDeleteCmd      = 13
	userSelectOwnerCmd = 14
	userSelectUserCmd  = 15
)

func (u *UserService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case userCreateCmd:
		u.handleUserCreateMessage(m)
	case userUpdateCmd:
		u.handleUserUpdateMessage(m)
	case userDeleteCmd:
		u.handleUserDeleteMessage(m)
	case userSelectOwnerCmd:
		u.handleUserSelectOwnerMessage(m)
	case userSelectUserCmd:
		u.handleUserSelectUserMessage(m)
	default:
		m.SendError("command not recognized")
	}
}

func (u *UserService) handleUserCreateMessage(m twine.ReceivedMessageI) {
	var incomingUser IncomingUser
	err := json.Unmarshal(m.Payload(), &incomingUser)
	if err != nil {
		panic(err)
	}

	proxyID := randomProxyID()
	v0userModel := u.AppspaceUserModels.GetV0(appspaceID)
	err = v0userModel.Create(proxyID, incomingUser.DisplayName, incomingUser.Permissions)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	}
	// send the full user as a reply? would make sense.
	user, err := v0userModel.Get(proxyID)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	}

	payload, err := json.Marshal(user)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	}

	err = m.Reply(11, payload)
	if err != nil {
		panic(err)
	}
}

func (u *UserService) handleUserUpdateMessage(m twine.ReceivedMessageI) {
	var incomingUser IncomingUser
	err := json.Unmarshal(m.Payload(), &incomingUser)
	if err != nil {
		panic(err)
	}

	v0userModel := u.AppspaceUserModels.GetV0(appspaceID)
	err = v0userModel.Update(incomingUser.ProxyID, incomingUser.DisplayName, incomingUser.Permissions)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	} else {
		m.SendOK()
	}
}

func (u *UserService) handleUserDeleteMessage(m twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(m.Payload()))

	v0userModel := u.AppspaceUserModels.GetV0(appspaceID)
	err := v0userModel.Delete(proxyID)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	} else {
		m.SendOK()
	}
}

func (u *UserService) handleUserSelectOwnerMessage(m twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(m.Payload()))

	u.DevAppspaceContactModel.SetOwnerProxyID(proxyID)

	m.SendOK()
}

func (u *UserService) handleUserSelectUserMessage(m twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(m.Payload()))

	if proxyID == "" {
		u.DevAuthenticator.SetNoAuth()
	} else {
		u.DevAuthenticator.Set(domain.Authentication{
			AppspaceID:  appspaceID,
			ProxyID:     proxyID,
			UserAccount: false,
		})
	}

	m.SendOK()
}

////////////
// random string
const chars36 = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand2 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomProxyID() domain.ProxyID {
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars36[seededRand2.Intn(len(chars36))]
	}
	return domain.ProxyID(string(b))
}
