package appspaceops

import (
	"io"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

type ManageUsers struct {
	AppspaceModel interface {
	} `checkinject:"required"`
	AppspaceUserModel interface {
		GetProxyIDsFromAuths(domain.AppspaceID, []domain.AppspaceUSerAuthQuery) ([]domain.ProxyID, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
		Create(appspaceID domain.AppspaceID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) (domain.ProxyID, error)
		Update(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) error
	} `checkinject:"required"`
	DropIDModel interface {
		GetForUser(userID domain.UserID) ([]domain.DropID, error)
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

// Currently no-op, but some functionality would be nice.
// I think this is auto-add a user to an appspace
// ..when the node is *shared* with them.
// ..after checking to avoid duplicate users.
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
		if tsnetU.Sharee {
			if _, found := findByAuth("tsnetid", tsnetU.FullID, curUsers); !found {
				// before adding, check that there isn't a similar user by comparing login name and match names?
				// m.addUserFromTSNet(appspaceID, tsnetU)
				// Note: auto-add is disabled in favor of auto-adding based on presence in contacts.
			}
		}
	}
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
	displayName := getDisplayNameFromTSNetUser(tsnetU)
	if displayName == "" {
		// log it
		displayName = "(invalid name)" // or not? Jut leav blank?
	}

	// sort out avatar: fetch it, save it, pass it.

	_, err := m.AppspaceUserModel.Create(appspaceID, displayName, "", []domain.EditAppspaceUserAuth{{
		Type:       "tsnetid",
		Identifier: tsnetU.FullID,
		Operation:  domain.EditOperationAdd,
	}})
	if err != nil {
		//log it
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

// InstanceUser returns the appspace user's proxy id
// by comparing the userID's identifiers with
// identifiers in the appspace's auths data
func (m *ManageUsers) InstanceUser(appspaceID domain.AppspaceID, userID domain.UserID) (domain.ProxyID, error) {
	// get user's auths. (user's dropids at this point, but also tailnet ids)
	// For each auth, get proxy id from appspace meta db.
	// [get instance appspace user, compare proxy ids..]
	//
	// if one proxy id return that
	// if more than one error conflict.
	// if non erro no result.

	auths := make([]domain.AppspaceUSerAuthQuery, 0)

	dropIDs, err := m.DropIDModel.GetForUser(userID)
	if err != nil {
		return domain.ProxyID(""), err
	}
	for _, d := range dropIDs {
		auths = append(auths, domain.AppspaceUSerAuthQuery{
			Type:       "dropid",
			Identifier: validator.JoinDropID(d.Domain, d.Handle)})
	}

	// TODO also add user's tsnetid if there is one.

	proxyIDs, err := m.AppspaceUserModel.GetProxyIDsFromAuths(appspaceID, auths)
	if err != nil {
		return domain.ProxyID(""), err
	}

	// TODO a.AppspaceInstanceUsers.GetProxyID(appspaceID, userID)
	// if it returns a proxy ID then handle it.

	if len(proxyIDs) == 0 {
		return domain.ProxyID(""), domain.ErrNoRowsInResultSet
	}
	if len(proxyIDs) > 1 {
		return domain.ProxyID(""), domain.ErrNoRowsInResultSet //TODO need an error here for this situation?
	}
	return proxyIDs[0], nil
}

// other functions here could be used to return maps of appspace proxy ids
// to instance users, for display and conflict resolution.

func (m *ManageUsers) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("ManageUsers")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
