package appspacemetadb

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

var asID = domain.AppspaceID(7)

func TestToDomainStructPermissions(t *testing.T) {
	u := &UserModel{}
	user := u.toDomainUser(domain.AppspaceID(7), appspaceUser{
		Permissions: "",
	})

	if len(user.Permissions) != 0 {
		t.Errorf("expected 0 permissions %v", len(user.Permissions))
	}
}

func TestCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	_, err := u.Create(asID, "email", "me@me.com")
	if err != nil {
		t.Error(err)
	}

	_, err = u.Create(asID, "email", "me@me.com")
	if err != ErrAuthIDExists {
		t.Error("Expect ErrAuthIDExists error")
	}
}

// add test for update auth
func TestUpdateAuth(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	proxyID, err := u.Create(asID, "email", "me@me.com")
	if err != nil {
		t.Error(err)
	}
	_, err = u.Create(asID, "email", "me@me2.com")
	if err != nil {
		t.Error(err)
	}

	err = u.UpdateAuth(asID, proxyID, "dropid", "moi.com/moi")
	if err != nil {
		t.Error(err)
	}

	user, _ := u.Get(asID, proxyID)
	if user.AuthType != "dropid" || user.AuthID != "moi.com/moi" {
		t.Errorf("did not get the auth data we expected: %v, %v", user.AuthType, user.AuthID)
	}

	err = u.UpdateAuth(asID, proxyID, "email", "me@me2.com")
	if err != ErrAuthIDExists {
		t.Errorf("Expected auth id exists error. %v", err)
	}
}

func TestUpdateMeta(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	proxyID, err := u.Create(asID, "email", "me@me.com")
	if err != nil {
		t.Error(err)
	}

	displayName := "ME me me"
	avatar := "mememe.jpg"
	permissions := []string{"read", "write"}
	err = u.UpdateMeta(asID, proxyID, displayName, avatar, permissions)
	if err != nil {
		t.Error(err)
	}

	appspaceUser, err := u.Get(asID, proxyID)
	if err != nil {
		t.Error(err)
	}

	if appspaceUser.DisplayName != displayName {
		t.Errorf("display name is diefferent: %v - %v", appspaceUser.DisplayName, displayName)
	}
	if appspaceUser.Avatar != avatar {
		t.Errorf("avatar is diefferent: %v - %v", appspaceUser.Avatar, avatar)
	}
	if !reflect.DeepEqual(appspaceUser.Permissions, permissions) {
		t.Errorf("permissions different: %v - %v ", appspaceUser.Permissions, permissions)
	}
}

func TestGetAll(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	_, err := u.Create(asID, "email", "me@me.com")
	if err != nil {
		t.Error(err)
	}
	_, err = u.Create(asID, "dropid", "me.com/me")
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

func TestGetByDropID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	dropID := "me.com/me"
	_, err := u.Create(asID, "dropid", dropID)
	if err != nil {
		t.Error(err)
	}

	user, err := u.GetByDropID(asID, dropID)
	if err != nil {
		t.Error(err)
	}
	if user.AuthID != dropID {
		t.Error("expected drop id to match")
	}
}

func TestDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	u := makeUserModel(mockCtrl)

	dropID := "me.com/me"
	proxyID, err := u.Create(asID, "dropid", dropID)
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
	if err != sql.ErrNoRows {
		t.Error(err)
	}
}

func makeUserModel(mockCtrl *gomock.Controller) *UserModel {
	db := getV0TestDBHandle()
	appspaceMetaDB := testmocks.NewMockAppspaceMetaDB(mockCtrl)
	appspaceMetaDB.EXPECT().GetHandle(asID).Return(db, nil).AnyTimes()

	return &UserModel{
		AppspaceMetaDB: appspaceMetaDB}
}
