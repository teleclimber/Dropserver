package appspacemetadb

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

var asID = domain.AppspaceID(7)

func TestToDomainStructPermissions(t *testing.T) {
	u := &UserModel{}
	user := u.toDomainUser(domain.AppspaceID(7), appspaceUser{
		Permissions: "",
	}, []domain.AppspaceUserAuth{})

	if len(user.Permissions) != 0 {
		t.Errorf("expected 0 permissions %v", len(user.Permissions))
	}
}

func TestUpdateAuthsSP(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	db, _ := u.AppspaceMetaDB.GetHandle(asID)

	proxyID := domain.ProxyID("abc")
	err := updateAuthsSP(db, proxyID, []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
	}}, true)
	if err == nil || !strings.Contains(err.Error(), "unknown operation") {
		t.Errorf("expected unkown operation error, go %v", err)
	}

	err = updateAuthsSP(db, proxyID, []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
		Operation:  domain.EditOperationAdd,
	}}, false)
	if err != nil {
		t.Error(err)
	}

	err = updateAuthsSP(db, proxyID, []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
		Operation:  domain.EditOperationRemove,
	}}, false)
	if err == nil || !strings.Contains(err.Error(), "got a remove op with allowRemove false") {
		t.Errorf("expected error about allowRemove, got %v", err)
	}

	err = updateAuthsSP(db, proxyID, []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
		Operation:  domain.EditOperationRemove,
	}}, true)
	if err != nil {
		t.Error(err)
	}
}

func TestCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	displayName := "display-name"
	avatar := "mememe.jpg"

	proxyID, err := u.Create(asID, displayName, avatar, []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
		ExtraName:  "extra",
	}})
	if err != nil {
		t.Error(err)
	}

	user, err := u.Get(asID, proxyID)
	if err != nil {
		t.Error(err)
	}
	if user.DisplayName != "display-name" {
		t.Errorf("expected display name to be display-name, got %s", user.DisplayName)
	}
	if len(user.Auths) != 1 {
		t.Error("expected 1 auth for user")
	}
	if user.Auths[0].Identifier != "me.com/me" {
		t.Error("no identifier in user auth")
	}
	if user.Auths[0].ExtraName != "extra" {
		t.Error("expected extra in user auth")
	}

	_, err = u.Create(asID, "someone", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
	}})
	if err != ErrAuthIDExists {
		t.Error("Expect ErrAuthIDExists error")
	}
}

func TestUpdate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	proxyID, err := u.Create(asID, "Some Name", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
	}})
	if err != nil {
		t.Error(err)
	}

	displayName := "ME me me"
	avatar := "mememe.jpg"
	err = u.Update(asID, proxyID, displayName, avatar, []domain.EditAppspaceUserAuth{}) // no auth edits
	if err != nil {
		t.Error(err)
	}

	appspaceUser, err := u.Get(asID, proxyID)
	if err != nil {
		t.Error(err)
	}

	if appspaceUser.DisplayName != displayName {
		t.Errorf("display name is different: %v - %v", appspaceUser.DisplayName, displayName)
	}
	if appspaceUser.Avatar != avatar {
		t.Errorf("avatar is different: %v - %v", appspaceUser.Avatar, avatar)
	}
}

func TestGetAll(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	_, err := u.Create(asID, "ME", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
	}})
	if err != nil {
		t.Error(err)
	}
	_, err = u.Create(asID, "YOU", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "you.com/you",
	}})
	if err != nil {
		t.Error(err)
	}

	appspaceUsers, err := u.GetAll(asID)
	if err != nil {
		t.Error(err)
	}

	if len(appspaceUsers) != 2 {
		t.Error("expected 2 appspace users")
	}
}

func TestGetByAuth(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	_, err := u.Create(asID, "ME", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
	}})
	if err != nil {
		t.Error(err)
	}

	user, err := u.GetByAuth(asID, "dropid", "me.com/me")
	if err != nil {
		t.Error(err)
	}
	if user.Auths[0].Identifier != "me.com/me" {
		t.Error("expected drop id to match")
	}
}

func TestDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	dropID := "me.com/me"
	proxyID, err := u.Create(asID, "ME", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: dropID,
	}})
	if err != nil {
		t.Error(err)
	}

	_, err = u.Get(asID, proxyID)
	if err != nil {
		t.Error(err)
	}
	err = u.Delete(asID, proxyID)
	if err != nil {
		t.Error(err)
	}
	_, err = u.Get(asID, proxyID)
	if err != domain.ErrNoRowsInResultSet {
		t.Error(err)
	}
}

func TestGetProxyIDsFromAuths(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	// Create first user with dropid auth
	proxyID1, err := u.Create(asID, "User One", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "user1.com/user1",
	}})
	if err != nil {
		t.Error(err)
	}

	// Create second user with email auth
	proxyID2, err := u.Create(asID, "User Two", "", []domain.EditAppspaceUserAuth{{
		Type:       "email",
		Identifier: "user2@example.com",
	}})
	if err != nil {
		t.Error(err)
	}

	// Create third user with multiple auths
	proxyID3, err := u.Create(asID, "User Three", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "user3.com/user3",
	}})
	if err != nil {
		t.Error(err)
	}
	// Add another auth to user 3
	err = u.UpdateAuth(asID, proxyID3, domain.EditAppspaceUserAuth{
		Type:       "email",
		Identifier: "user3@example.com",
		Operation:  domain.EditOperationAdd,
	})
	if err != nil {
		t.Error(err)
	}

	// Test 1: Single query that matches one user
	proxyIDs, err := u.GetProxyIDsFromAuths(asID, []domain.AppspaceUserAuthBare{{
		Type:       "dropid",
		Identifier: "user1.com/user1",
	}})
	if err != nil {
		t.Error(err)
	}
	if len(proxyIDs) != 1 {
		t.Errorf("expected 1 proxy ID, got %d", len(proxyIDs))
	}
	if proxyIDs[0] != proxyID1 {
		t.Errorf("expected proxy ID %s, got %s", proxyID1, proxyIDs[0])
	}

	// Test 2: Multiple queries matching different users
	proxyIDs, err = u.GetProxyIDsFromAuths(asID, []domain.AppspaceUserAuthBare{
		{
			Type:       "dropid",
			Identifier: "user1.com/user1",
		},
		{
			Type:       "email",
			Identifier: "user2@example.com",
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(proxyIDs) != 2 {
		t.Errorf("expected 2 proxy IDs, got %d", len(proxyIDs))
	}
	// Check both IDs are present (order doesn't matter)
	found1, found2 := false, false
	for _, id := range proxyIDs {
		if id == proxyID1 {
			found1 = true
		}
		if id == proxyID2 {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("expected both proxyID1 and proxyID2 in results")
	}

	// Test 3: Multiple queries matching the same user (should return unique proxy ID)
	proxyIDs, err = u.GetProxyIDsFromAuths(asID, []domain.AppspaceUserAuthBare{
		{
			Type:       "dropid",
			Identifier: "user3.com/user3",
		},
		{
			Type:       "email",
			Identifier: "user3@example.com",
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(proxyIDs) != 1 {
		t.Errorf("expected 1 unique proxy ID, got %d", len(proxyIDs))
	}
	if proxyIDs[0] != proxyID3 {
		t.Errorf("expected proxy ID %s, got %s", proxyID3, proxyIDs[0])
	}

	// Test 4: Query with no matches (should return empty slice)
	proxyIDs, err = u.GetProxyIDsFromAuths(asID, []domain.AppspaceUserAuthBare{{
		Type:       "dropid",
		Identifier: "nonexistent.com/user",
	}})
	if err != nil {
		t.Error(err)
	}
	if len(proxyIDs) != 0 {
		t.Errorf("expected 0 proxy IDs for nonexistent auth, got %d", len(proxyIDs))
	}

	// Test 5: Mix of matching and non-matching queries
	proxyIDs, err = u.GetProxyIDsFromAuths(asID, []domain.AppspaceUserAuthBare{
		{
			Type:       "dropid",
			Identifier: "user1.com/user1",
		},
		{
			Type:       "dropid",
			Identifier: "nonexistent.com/user",
		},
		{
			Type:       "email",
			Identifier: "user2@example.com",
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(proxyIDs) != 2 {
		t.Errorf("expected 2 proxy IDs, got %d", len(proxyIDs))
	}

	// Test 6: Empty query list
	proxyIDs, err = u.GetProxyIDsFromAuths(asID, []domain.AppspaceUserAuthBare{})
	if err != nil {
		t.Error(err)
	}
	if len(proxyIDs) != 0 {
		t.Errorf("expected 0 proxy IDs for empty query list, got %d", len(proxyIDs))
	}
}

func makeUserModel(mockCtrl *gomock.Controller) *UserModel {
	// Beware of in-memory DBs: they vanish as soon as the connection closes!
	// We may be able to start a sqlx transaction to avoid problems with that?
	// See: https://github.com/jmoiron/sqlx/issues/164
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	dbc := &dbConn{
		handle: handle,
	}
	err = dbc.migrateTo(curSchema)
	if err != nil {
		panic("Failed to migrate")
	}

	appspaceMetaDB := testmocks.NewMockAppspaceMetaDB(mockCtrl)
	appspaceMetaDB.EXPECT().GetHandle(asID).Return(dbc.handle, nil).AnyTimes()

	return &UserModel{
		AppspaceMetaDB: appspaceMetaDB}
}
