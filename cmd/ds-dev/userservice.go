package main

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

// TODO: how does this handle changes in DS API versions?

// What do we need ?
// app files events and app files so we can load user permissions (later)
//

// UserService is a twine service that sets the desired user params
// and keeps the frontend up to date with app's declared permissions
type UserService struct {
	DevAuthenticator     *DevAuthenticator `checkinject:"required"`
	AppspaceUsersModelV0 interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
		Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
		UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, permissions []string) error
		Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error
	} `checkinject:"required"`
	Avatars interface {
		Save(locationKey string, proxyID domain.ProxyID, img io.Reader) (string, error)
		Remove(locationKey string, fn string) error
	} `checkinject:"required"`
	AppspaceFilesEvents interface {
		Subscribe(chan<- domain.AppspaceID)
		Unsubscribe(chan<- domain.AppspaceID)
	} `checkinject:"required"`

	dummyDropidNum int
}

// Start creates listeners and then shuts everything down when twine exits
func (u *UserService) Start(t *twine.Twine) {
	asFilesCh := make(chan domain.AppspaceID)
	go func() {
		for range asFilesCh {
			u.sendUsers(t)
			// presumably appspace files contain owner info, so it gets reset in that way somehow
			// ..just have to send it back down
			// Also for selected user, we can maybe get away with doing nothing? They either still exist or not.
		}
	}()
	u.AppspaceFilesEvents.Subscribe(asFilesCh)

	// send initial users down
	u.sendUsers(t)

	// TODO [later] subscripe to app files changes and resend user permissions when changed

	// Wait for twine to close and shut it all down.
	t.WaitClose()

	fmt.Println("closing user service")

	u.AppspaceFilesEvents.Unsubscribe(asFilesCh)
	close(asFilesCh)

	u.dummyDropidNum = 0
}

// outgoing commnds:
const (
	loadAllUsersCmd = 11
)

func (u *UserService) sendUsers(twine *twine.Twine) {
	users, err := u.AppspaceUsersModelV0.GetAll(appspaceID)
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

type IncomingUser struct {
	ProxyID     domain.ProxyID `json:"proxy_id"`
	DisplayName string         `json:"display_name"`
	Avatar      string         `json:"avatar"`
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
// - SelectUser (proxy) .. sets that in DevAuth

// incoming commands
const (
	userCreateCmd     = 11
	userUpdateCmd     = 12
	userDeleteCmd     = 13
	userSelectUserCmd = 15
)

func (u *UserService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case userCreateCmd:
		u.handleUserCreateMessage(m)
	case userUpdateCmd:
		u.handleUserUpdateMessage(m)
	case userDeleteCmd:
		u.handleUserDeleteMessage(m)
	case userSelectUserCmd:
		u.handleUserSelectUserMessage(m)
	default:
		m.SendError(fmt.Sprintf("command not recognized %v", m.CommandID()))
	}
}

func (u *UserService) handleUserCreateMessage(m twine.ReceivedMessageI) {
	var incomingUser IncomingUser
	err := json.Unmarshal(m.Payload(), &incomingUser)
	if err != nil {
		panic(err)
	}

	u.dummyDropidNum++
	proxyID, err := u.AppspaceUsersModelV0.Create(appspaceID, "dropid", fmt.Sprintf("dropid.dummy.develop/%v", u.dummyDropidNum))
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	}

	avatar := ""
	if incomingUser.Avatar != "" {
		// for now we assume avatar is from baked-in avatars
		f, err := avatarsFS.Open(filepath.Join("avatars", incomingUser.Avatar))
		if err != nil {
			panic(err)
		}
		avatar, err = u.Avatars.Save("", proxyID, f)
		if err != nil {
			panic(err)
		}
	}

	err = u.AppspaceUsersModelV0.UpdateMeta(appspaceID, proxyID, incomingUser.DisplayName, avatar, incomingUser.Permissions)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	}

	// send the full user as a reply? would make sense.
	user, err := u.AppspaceUsersModelV0.Get(appspaceID, proxyID)
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
		m.SendError(err.Error())
		panic(err)
	}

	avatar := ""
	user, err := u.AppspaceUsersModelV0.Get(appspaceID, incomingUser.ProxyID)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	}
	// create new avatar:
	if incomingUser.Avatar != "" && user.Avatar != incomingUser.Avatar {
		// for now we assume avatar is from baked-in avatars
		f, err := avatarsFS.Open(filepath.Join("avatars", incomingUser.Avatar))
		if err != nil {
			m.SendError(err.Error())
			panic(err)
		}
		avatar, err = u.Avatars.Save("", incomingUser.ProxyID, f)
		if err != nil {
			m.SendError(err.Error())
			panic(err)
		}
	}
	// remove old avatar
	if user.Avatar != "" && user.Avatar != incomingUser.Avatar {
		err = u.Avatars.Remove("", user.Avatar)
		if err != nil {
			m.SendError(err.Error())
			panic(err)
		}
	}

	err = u.AppspaceUsersModelV0.UpdateMeta(appspaceID, incomingUser.ProxyID, incomingUser.DisplayName, avatar, incomingUser.Permissions)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	}

	// send the full user as a reply? would make sense.
	user, err = u.AppspaceUsersModelV0.Get(appspaceID, incomingUser.ProxyID)
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

func (u *UserService) handleUserDeleteMessage(m twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(m.Payload()))

	user, err := u.AppspaceUsersModelV0.Get(appspaceID, proxyID)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	}

	if user.Avatar != "" {
		err = u.Avatars.Remove("", user.Avatar)
		if err != nil {
			m.SendError(err.Error())
			panic(err)
		}
	}

	err = u.AppspaceUsersModelV0.Delete(appspaceID, proxyID)
	if err != nil {
		m.SendError(err.Error())
		panic(err)
	} else {
		m.SendOK()
	}
}

func (u *UserService) handleUserSelectUserMessage(m twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(m.Payload()))

	if proxyID == "" {
		u.DevAuthenticator.SetNoAuth()
	} else {
		u.DevAuthenticator.Set(domain.Authentication{
			Authenticated: true,
			AppspaceID:    appspaceID,
			ProxyID:       proxyID,
			UserAccount:   false,
		})
	}

	m.SendOK()
}
