package appspaceops

import (
	"io"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

type ManageUsers struct {
	AppspaceModel interface {
		GetAll() ([]domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceUserModel interface {
		GetProxyIDsFromAuths(domain.AppspaceID, []domain.AppspaceUserAuthBare) ([]domain.ProxyID, error)
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
		Create(appspaceID domain.AppspaceID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) (domain.ProxyID, error)
		Update(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) error
	} `checkinject:"required"`
	DropIDModel interface {
		Get(handle string, dom string) (domain.DropID, error)
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

// User getting.
// There are two gotchas with users:
// - an instance user matches more than one appspace user
// - an appspace user matches more than one instance user.
// - an instance user linked with a proxy id that doesn't exist (via appspacae_instance_user db)

// AppspaceUsers returns appspace proxyIDs that match a user id from the instance,
// including their matching auths and how they match with instance users if any.
// Any conflicts are also fully detailed
func (m *ManageUsers) AppspaceUsers(appspaceID domain.AppspaceID) (map[domain.ProxyID]domain.UserIDProxyIDConflicts, error) {
	appspaceUsers, err := m.AppspaceUserModel.GetAll(appspaceID)
	if err != nil {
		return nil, err
	}

	ids := idConflicts{}
	ids.init()

	for _, au := range appspaceUsers {
		for _, auth := range au.Auths {
			userID, err := m.getUserFromAuth(auth)
			if err == domain.ErrNoRowsInResultSet {
				continue
			} else if err != nil {
				return nil, err
			} else {
				ids.addAuth(au.ProxyID, userID, bareAuth(auth))
			}
		}
	}

	// TODO get instance_appspace_user for appspaceid and add using ids.addInstance(...)

	ret := ids.getProxyIDsConflicts() // TODO why is this by proxy ID?

	return ret, nil
}

// GetProxyID returns the appspace user's proxy id
// by comparing the userID's identifiers with
// identifiers in the appspace's auths data.
// The ProxyID is free of conflicts.
// If there are conflicts it returns ErrNoRowsInResultSet
func (m *ManageUsers) GetProxyID(appspaceID domain.AppspaceID, userID domain.UserID) (domain.ProxyID, error) {
	auths, err := m.getUserAuths(userID)
	if err != nil {
		return domain.ProxyID(""), err
	}
	uConflicts, err := m.UserInAppspace(userID, auths, appspaceID)
	if err != nil {
		return domain.ProxyID(""), err
	}
	if uConflicts.Conflict {
		return domain.ProxyID(""), domain.ErrNoRowsInResultSet
	}
	return uConflicts.ProxyID, nil
}

// AppspacesForUser returns the appspaces and proxy ids
// that the instance user is a user in.
// If a conflict exists it is returned and the Conflict detailed
func (m *ManageUsers) AppspacesForUser(userID domain.UserID) (map[domain.AppspaceID]domain.UserIDProxyIDConflicts, error) {
	ret := make(map[domain.AppspaceID]domain.UserIDProxyIDConflicts)

	auths, err := m.getUserAuths(userID)
	if err != nil {
		return ret, err
	}

	appspaces, err := m.AppspaceModel.GetAll()
	if err != nil {
		return ret, err
	}

	for _, a := range appspaces {
		uia, err := m.UserInAppspace(userID, auths, a.AppspaceID)
		if err == domain.ErrNoRowsInResultSet {
			// no-op
		} else if err != nil {
			// ignore. Maybe the appspace is borked.
			// maybe log it.
		} else {
			ret[a.AppspaceID] = uia
		}
	}
	return ret, nil
}

// UserInAppspace returns an instance user's participation in an appspace
// If there are conflicts, these are noted. Conflicts are:
// - the user matches more than one proxyID of that appspcae
// - Multiple users match the proxy ID
// TODO this has a lot of duplication over UsersForAppspace
func (m *ManageUsers) UserInAppspace(userID domain.UserID, auths []domain.AppspaceUserAuthBare, appspaceID domain.AppspaceID) (domain.UserIDProxyIDConflicts, error) {
	//ret := domain.UserIDProxyIDConflicts{}

	ids := idConflicts{}
	ids.init()

	// Maybe loop over each auth so we know what auth matched what?
	// Also, would be interesting to think of whether we can guarantee that a single auth returns a single proxy ID
	// It might simplify things.
	proxyIDs, err := m.AppspaceUserModel.GetProxyIDsFromAuths(appspaceID, auths)
	if err != nil {
		return domain.UserIDProxyIDConflicts{}, err
	}
	for _, p := range proxyIDs {
		ids.add(p, userID) //maybe do addAuth...
	}

	// TODO get proxy IDS from instance users DB table

	conflicts := ids.getUserIDsConflicts()
	userConflicts, ok := conflicts[userID]
	if !ok {
		return domain.UserIDProxyIDConflicts{}, domain.ErrNoRowsInResultSet // not a great error but gets the point across.
	} else if userConflicts.Conflict {
		return userConflicts, nil // we already have a conflict, bail here.
	}

	// from here we're dealing with one proxy ID
	proxyID := userConflicts.ProxyID

	// reverse the query to ensure our proxy ID is only matched by one userID
	appspaceUser, err := m.AppspaceUserModel.Get(appspaceID, proxyID)
	if err != nil {
		return userConflicts, err
	}
	for _, auth := range appspaceUser.Auths {
		userID, err := m.getUserFromAuth(auth)
		if err == domain.ErrNoRowsInResultSet {
			continue
		} else if err != nil {
			return userConflicts, err
		} else {
			ids.addAuth(proxyID, userID, bareAuth(auth))
		}
	}

	userConflicts = ids.getUserIDsConflicts()[userID]

	return userConflicts, nil
}

func (m *ManageUsers) getUserFromAuth(auth domain.AppspaceUserAuth) (domain.UserID, error) {
	userID := domain.UserID(0)
	err := domain.ErrNoRowsInResultSet
	switch auth.Type {
	case "dropid":
		userID, err = m.getUserIDForDropID(auth.Identifier)
	case "tsnetid":
		// TODO
	}
	return userID, err
}

// UserIDsForAppspace returns all instance user ids that are users
// of the appspace. Conflicts are included.
func (m *ManageUsers) UsersForAppspace(appspaceID domain.AppspaceID) (map[domain.UserID]domain.UserIDProxyIDConflicts, error) {
	appspaceUsers, err := m.AppspaceUserModel.GetAll(appspaceID)
	if err != nil {
		return nil, err
	}

	ids := idConflicts{}
	ids.init()

	for _, au := range appspaceUsers {
		for _, auth := range au.Auths {
			userID, err := m.getUserFromAuth(auth)
			if err == domain.ErrNoRowsInResultSet {
				continue
			} else if err != nil {
				return nil, err
			} else {
				ids.add(au.ProxyID, userID)
			}
		}
	}

	// TODO get instance_appspace_user for appspaceid and add the user ids.

	ret := ids.getUserIDsConflicts()

	return ret, nil
}

func (m *ManageUsers) getUserIDForDropID(dropID string) (domain.UserID, error) {
	h, d := validator.SplitDropID(dropID)
	did, err := m.DropIDModel.Get(h, d)
	return did.UserID, err
}

func (m *ManageUsers) getUserAuths(userID domain.UserID) ([]domain.AppspaceUserAuthBare, error) {
	auths := make([]domain.AppspaceUserAuthBare, 0)

	dropIDs, err := m.DropIDModel.GetForUser(userID)
	if err != nil {
		return auths, err
	}
	for _, d := range dropIDs {
		auths = append(auths, domain.AppspaceUserAuthBare{
			Type:       "dropid",
			Identifier: validator.JoinDropID(d.Handle, d.Domain)})
	}

	// TODO also add user's tsnetid if there is one.

	return auths, nil
}

func (m *ManageUsers) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("ManageUsers")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

func bareAuth(auth domain.AppspaceUserAuth) domain.AppspaceUserAuthBare {
	return domain.AppspaceUserAuthBare{
		Type:       auth.Type,
		Identifier: auth.Identifier}
}

type idConflicts struct {
	userIDs   pairs[domain.UserID, domain.ProxyID]
	proxyIDs  pairs[domain.ProxyID, domain.UserID]
	pairAuths pairAuths
}

func (i *idConflicts) init() {
	i.userIDs.init()
	i.proxyIDs.init()
	i.pairAuths = makePairAuths()
}

func (i *idConflicts) add(p domain.ProxyID, u domain.UserID) {
	i.userIDs.add(u, p)
	i.proxyIDs.add(p, u)
}

func (i *idConflicts) addAuth(p domain.ProxyID, u domain.UserID, auth domain.AppspaceUserAuthBare) {
	i.add(p, u)
	i.pairAuths.addAuth(p, u, auth)
}

func (i *idConflicts) addInstance(p domain.ProxyID, u domain.UserID) {
	i.add(p, u)
	i.pairAuths.addInstanceRelation(p, u)
}

func (i *idConflicts) getUserIDsConflicts() map[domain.UserID]domain.UserIDProxyIDConflicts {
	ret := make(map[domain.UserID]domain.UserIDProxyIDConflicts)
	for t, c := range i.getConflicts() {
		ret[t.UserID] = c
	}
	return ret
}

func (i *idConflicts) getProxyIDsConflicts() map[domain.ProxyID]domain.UserIDProxyIDConflicts {
	ret := make(map[domain.ProxyID]domain.UserIDProxyIDConflicts)
	for t, c := range i.getConflicts() {
		ret[t.ProxyID] = c
	}
	return ret
}

// Here instead of a ProxyID and a userID based function
// make a function that returns map[UserProxyTuple]domain.UserIDProxyIDConflicts
// then have u or p=specific function to make to turn those into map[UserID|ProxyID]domain.UserIDProxyIDConflicts
func (i *idConflicts) getConflicts() map[UserProxyTuple]domain.UserIDProxyIDConflicts {
	ret := make(map[UserProxyTuple]domain.UserIDProxyIDConflicts)
	for _, t := range i.pairAuths.getAllPairs() {
		r := domain.UserIDProxyIDConflicts{}

		userIDs := i.proxyIDs.getAllV(t.ProxyID)
		r.UserIDMatches = make(map[domain.UserID]domain.UserIDProxyIDMatches)
		for _, userID := range userIDs {
			r.UserIDMatches[userID] = i.pairAuths.getMatchedOn(t.ProxyID, userID)
		}

		proxyIDs := i.userIDs.getAllV(t.UserID)
		r.ProxyIDMatches = make(map[domain.ProxyID]domain.UserIDProxyIDMatches)
		for _, proxyID := range proxyIDs {
			r.ProxyIDMatches[proxyID] = i.pairAuths.getMatchedOn(proxyID, t.UserID)
		}

		if len(userIDs) == 1 && len(proxyIDs) == 1 {
			r.UserID = userIDs[0]
			r.ProxyID = proxyIDs[0]
		} else {
			r.Conflict = true
		}
		ret[t] = r
	}
	return ret
}

// pairs tracks conflicts between ids.
type pairs[K domain.UserID | domain.ProxyID, V domain.UserID | domain.ProxyID] struct {
	associations map[K]map[V]struct{}
}

func (p *pairs[K, V]) init() {
	p.associations = make(map[K]map[V]struct{})
}

func (p *pairs[K, V]) add(k K, v V) {
	if _, ok := p.associations[k]; !ok {
		p.associations[k] = make(map[V]struct{})
	}
	p.associations[k][v] = struct{}{}
}

func (p *pairs[K, V]) getSingleV(k K) (v V) { // apparently unused. Maybe can it?
	assoc, ok := p.associations[k]
	if !ok {
		return
	}
	for v = range assoc {
		return
	}
	return
}

func (p *pairs[K, V]) getAllV(k K) []V {
	assoc, ok := p.associations[k]
	if !ok {
		return []V{}
	}
	ret := make([]V, len(assoc))
	i := 0
	for v := range assoc {
		ret[i] = v
		i++
	}
	return ret
}

type UserProxyTuple struct {
	UserID  domain.UserID
	ProxyID domain.ProxyID
}

type pairAuths struct {
	auths    map[UserProxyTuple][]domain.AppspaceUserAuthBare
	instance map[UserProxyTuple]struct{}
}

func makePairAuths() pairAuths {
	return pairAuths{
		auths:    make(map[UserProxyTuple][]domain.AppspaceUserAuthBare),
		instance: make(map[UserProxyTuple]struct{}),
	}
}
func (a *pairAuths) addAuth(p domain.ProxyID, u domain.UserID, auth domain.AppspaceUserAuthBare) {
	k := UserProxyTuple{
		UserID:  u,
		ProxyID: p}
	if _, ok := a.auths[k]; !ok {
		a.auths[k] = []domain.AppspaceUserAuthBare{}
	}
	a.auths[k] = append(a.auths[k], auth)
}

func (a *pairAuths) addInstanceRelation(p domain.ProxyID, u domain.UserID) {
	a.instance[UserProxyTuple{
		UserID:  u,
		ProxyID: p}] = struct{}{}
}

func (a *pairAuths) getAllPairs() []UserProxyTuple {
	m := make(map[UserProxyTuple]struct{})
	for t := range a.auths {
		m[t] = struct{}{}
	}
	for t := range a.instance {
		m[t] = struct{}{}
	}
	ret := make([]UserProxyTuple, len(m))
	i := 0
	for t := range m {
		ret[i] = t
		i++
	}
	return ret
}
func (a *pairAuths) getAuths(p domain.ProxyID, u domain.UserID) []domain.AppspaceUserAuthBare {
	return a.auths[UserProxyTuple{
		UserID:  u,
		ProxyID: p}]
}

func (a *pairAuths) getInstanceRelation(p domain.ProxyID, u domain.UserID) bool {
	_, ok := a.instance[UserProxyTuple{
		UserID:  u,
		ProxyID: p}]
	return ok
}

func (a *pairAuths) getMatchedOn(p domain.ProxyID, u domain.UserID) domain.UserIDProxyIDMatches {
	return domain.UserIDProxyIDMatches{
		Instance: a.getInstanceRelation(p, u),
		Auths:    a.getAuths(p, u)}
}
