package appspaceusermodel

import (
	"reflect"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceUserModel{
		DB: db}

	model.PrepareStatements()
}

func TestToDomainStructPermissions(t *testing.T) {
	user := toDomainStruct(appspaceUser{
		Permissions: "",
	})

	if len(user.Permissions) != 0 {
		t.Errorf("expected 0 permissions %v", len(user.Permissions))
	}
}

func TestCreate(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &AppspaceUserModel{
		DB: &domain.DB{
			Handle: h}}
	model.PrepareStatements()

	_, err := model.Create(domain.AppspaceID(7), "email", "me@me.com")
	if err != nil {
		t.Error(err)
	}

	// should we use sentinel errors here?
	_, err = model.Create(domain.AppspaceID(7), "email", "me@me.com")
	if err != ErrAuthIDExists {
		t.Error("Expect ErrAuthIDExists error")
	}

}

func TestUpdateMeta(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &AppspaceUserModel{
		DB: &domain.DB{
			Handle: h}}
	model.PrepareStatements()

	appspaceID := domain.AppspaceID(7)
	proxyID, err := model.Create(appspaceID, "email", "me@me.com")
	if err != nil {
		t.Error(err)
	}

	displayName := "ME me me"
	avatar := "mememe.jpg"
	permissions := []string{"read", "write"}
	err = model.UpdateMeta(appspaceID, proxyID, displayName, avatar, permissions)
	if err != nil {
		t.Error(err)
	}

	appspaceUser, err := model.Get(appspaceID, proxyID)
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

func TestGetForAppspace(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &AppspaceUserModel{
		DB: &domain.DB{
			Handle: h}}
	model.PrepareStatements()

	appspaceID := domain.AppspaceID(7)
	_, err := model.Create(appspaceID, "email", "me@me.com")
	if err != nil {
		t.Error(err)
	}
	_, err = model.Create(appspaceID, "dropid", "me.com/me")
	if err != nil {
		t.Error(err)
	}

	appspaceUsers, err := model.GetForAppspace(appspaceID)
	if err != nil {
		t.Error(err)
	}

	if len(appspaceUsers) != 2 {
		t.Error("expected 2 appspace users")
	}

}

func TestGetByDropID(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &AppspaceUserModel{
		DB: &domain.DB{
			Handle: h}}
	model.PrepareStatements()

	appspaceID := domain.AppspaceID(7)
	dropID := "me.com/me"
	_, err := model.Create(appspaceID, "dropid", dropID)
	if err != nil {
		t.Error(err)
	}

	user, err := model.GetByDropID(appspaceID, dropID)
	if err != nil {
		t.Error(err)
	}
	if user.AuthID != dropID {
		t.Error("expected drop id to match")
	}
}
