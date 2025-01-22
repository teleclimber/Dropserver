package appspaceops

import (
	"fmt"
	"io"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type ManageUsers struct {
	AppspaceModel interface {
	} `checkinject:"required"`
	AppspaceUserModel interface {
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
		Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
		UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, permissions []string) error
	} `checkinject:"required"`
	Avatars interface {
		Save(locationKey string, proxyID domain.ProxyID, img io.Reader) (string, error)
	} `checkinject:"required"`
	AppspaceTSNet interface {
		GetPeerUsers(appspaceID domain.AppspaceID) []domain.TSNetPeerUser
	} `checkinject:"required"`
	AppspaceTSNetPeersEvents interface {
		Subscribe() <-chan domain.AppspaceID
		Unsubscribe(ch <-chan domain.AppspaceID)
	} `checkinject:"required"`

	appspaceIdChan <-chan domain.AppspaceID
}

func (m *ManageUsers) Init() { // context would be great here.
	m.appspaceIdChan = m.AppspaceTSNetPeersEvents.Subscribe()
	go func() {
		for appspaceID := range m.appspaceIdChan {
			m.fromTSNet(appspaceID)
		}
	}()
}

func (m *ManageUsers) fromTSNet(appspaceID domain.AppspaceID) {
	tsnetUsers := m.AppspaceTSNet.GetPeerUsers(appspaceID)
	curUsers, err := m.AppspaceUserModel.GetAll(appspaceID)
	if err != nil {
		// the error is logged in user model, so just abandon? Or log it here too?
		return
	}

	for _, tsnetU := range tsnetUsers {
		// check control url is not ""?
		if tsnetU.ControlURL == "" {
			continue
		}
		identifier := identifierFromTSNet(tsnetU)
		if tsnetU.Sharee {
			if _, found := findByAuth("tsnetid", identifier, curUsers); !found {
				// before adding, check that there isn't a similar user by comparing login name and match names?
				m.addUserFromTSNet(appspaceID, tsnetU)
				// after adding user, should we not include that user in tsnetUsers?
				// ..in case of possible dupe (two tsnet accounts for one user?)
				// -> seems really edge-casey
			}
		}
	}
}

func identifierFromTSNet(user domain.TSNetPeerUser) string {
	return fmt.Sprintf("%s@%s", user.ID, user.ControlURL)
}

func findByAuth(authType string, authID string, curUsers []domain.AppspaceUser) (domain.AppspaceUser, bool) {
	for _, curU := range curUsers {
		for _, a := range curU.Auths {
			if a.Type == authType && a.Identifier == authID {
				return curU, true
			}
		}
	}
	return domain.AppspaceUser{}, false
}

// do we even have email anywhere.?
// func findByEmail(email string, curUsers []domain.AppspaceUser) (domain.AppspaceUser, bool) {

// }

func (m *ManageUsers) addUserFromTSNet(appspaceID domain.AppspaceID, tsnetU domain.TSNetPeerUser) {
	identifier := identifierFromTSNet(tsnetU)
	proxyID, err := m.AppspaceUserModel.Create(appspaceID, "tsnetid", identifier)
	if err != nil {
		//log it
		return
	}

	// sort out avatar: fetch it, save it, pass it.

	displayName := getDisplayNameFromTSNetUser(tsnetU)
	if displayName == "" {
		// log it
		displayName = "(invalid name)" // or not? Jut leav blank?
	}

	err = m.AppspaceUserModel.UpdateMeta(appspaceID, proxyID, displayName, "", []string{})
	if err != nil {
		// log it
		return
	}

}

func getDisplayNameFromTSNetUser(tsnetU domain.TSNetPeerUser) string {
	displayName := validator.NormalizeDisplayName(tsnetU.DisplayName)
	if displayName == "" {
		displayName = validator.NormalizeDisplayName(tsnetU.LoginName)
	}
	if len(displayName) >= 30 {
		displayName = displayName[:30]
	}
	if err := validator.DisplayName(displayName); err != nil {
		displayName = ""
	}
	return displayName
}
