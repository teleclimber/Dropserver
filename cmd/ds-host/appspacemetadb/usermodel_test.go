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

	_, err = u.Create(asID, "someone", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
	}})
	if err != ErrAuthIDExists {
		t.Error("Expect ErrAuthIDExists error")
	}
}

func TestAddAuth(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	proxyID, err := u.Create(asID, "ME", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
	}})
	if err != nil {
		t.Error(err)
	}
	_, err = u.Create(asID, "you", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "you.com/you",
	}})
	if err != nil {
		t.Error(err)
	}

	err = u.AddAuth(asID, proxyID, "dropid", "moi.com/moi")
	if err != nil {
		t.Error(err)
	}

	user, err := u.GetByAuth(asID, "dropid", "moi.com/moi")
	if err != nil {
		t.Error(err)
	}
	if user.ProxyID != proxyID {
		t.Error("did not get the user we expected", user)
	}

	err = u.AddAuth(asID, proxyID, "dropid", "you.com/you")
	if err != ErrAuthIDExists {
		t.Errorf("Expected auth id exists error. %v", err)
	}
}

func TestDeleteAuth(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	proxyID, err := u.Create(asID, "someone", "", []domain.EditAppspaceUserAuth{{
		Type:       "dropid",
		Identifier: "me.com/me",
	}})
	if err != nil {
		t.Error(err)
	}

	err = u.DeleteAuth(asID, proxyID, "dropid", "me.com/me")
	if err != nil {
		t.Error(err)
	}

	user, err := u.Get(asID, proxyID)
	if err != nil {
		t.Error(err)
	}
	if len(user.Auths) != 0 {
		t.Error("expected zero auths on user", user)
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
