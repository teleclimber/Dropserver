package appspaceusermodel

import (
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

func TestUpdateAuth(t *testing.T) {
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
	err = model.UpdateMeta(appspaceID, proxyID, displayName, []string{"read", "write"})
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
