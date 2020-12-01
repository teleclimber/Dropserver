package main

import (
	"encoding/json"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

// What do we need ?
// app files events and app files so we can load user permissions (later)
//

// UserService is a twine service that sets the desired user params
// and keeps the frontend up to date with app's declared permissions
type UserService struct {
	DevAuthenticator     *DevAuthenticator
	DevAppspaceUserModel *DevAppspaceUserModel
}

// Start creates listeners and then shuts everything down when twine exits
func (u *UserService) Start(t *twine.Twine) {
	// [later] subscripe to app files changes and resend user permissions when changed

	// Wait for twine to close and shut it all down.
	t.WaitClose()

	fmt.Println("closing user service")

}

type IncomingUser struct {
	Type        string         `json:"type"`
	Permissions []string       `json:"permissions"`
	DisplayName string         `json:"display_name"`
	ProxyID     domain.ProxyID `json:"proxy_id"`
}

func (u *UserService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case 11:
		var incomingUser IncomingUser
		err := json.Unmarshal(m.Payload(), &incomingUser)
		if err != nil {
			panic(err)
		}

		switch incomingUser.Type {
		case "public":
			u.DevAuthenticator.SetNoAuth()
			u.DevAppspaceUserModel.SetNoUser()
		case "user":
			u.DevAuthenticator.Set(domain.Authentication{
				ProxyID:    incomingUser.ProxyID,
				AppspaceID: appspaceID,
			})
			u.DevAppspaceUserModel.SetUser(domain.AppspaceUser{
				AppspaceID:  appspaceID,
				ContactID:   7,
				DisplayName: incomingUser.DisplayName,
				Permissions: incomingUser.Permissions,
				ProxyID:     incomingUser.ProxyID,
			})
		case "owner":
			u.DevAuthenticator.Set(domain.Authentication{
				ProxyID:     incomingUser.ProxyID,
				AppspaceID:  appspaceID,
				UserAccount: false,
			})
			u.DevAppspaceUserModel.SetUser(domain.AppspaceUser{
				IsOwner:     true,
				AppspaceID:  appspaceID,
				ContactID:   1,
				DisplayName: "Ze Owner",
				ProxyID:     incomingUser.ProxyID,
			})
		default:
			panic("what is this incoming type? " + incomingUser.Type)
		}

		m.SendOK()

	default:
		m.SendError("command not recognized")
	}
}

// const loadPermissionsCmd = 11

// type userPermissions struct {	// use later
// 	Permissions []string
// }

// func (s *UserService) sendAppspaceRoutes(twine *twine.Twine) {
// 	v0routeModel := s.AppspaceRouteModels.GetV0(appspaceID)
// 	routes, err := v0routeModel.GetAll()
// 	if err != nil {
// 		fmt.Println("sendAppspaceRoutes error getting routes: " + err.Error())
// 	}

// 	data := appspaceRoutes{
// 		Path:   "",
// 		Routes: routes}

// 	bytes, err := json.Marshal(data)
// 	if err != nil {
// 		fmt.Println("sendAppspaceRoutes json Marshal Error: " + err.Error())
// 	}

// 	_, err = twine.SendBlock(appspaceRouteService, loadAllCmd, bytes)
// 	if err != nil {
// 		fmt.Println("sendAppspaceRoutes SendBlock Error: " + err.Error())
// 	}
// }
