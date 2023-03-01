package userinvitationmodel

import (
	"database/sql"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &UserInvitationModel{
		DB: db}

	model.PrepareStatements()
}

func TestGetAllEmpty(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &UserInvitationModel{
		DB: db}

	model.PrepareStatements()

	invites, err := model.GetAll()
	if err != nil {
		t.Error(err)
	}
	if invites == nil {
		t.Error("invites is nil, should be empty")
	}
	if len(invites) != 0 {
		t.Error("invites not empty")
	}
}

func TestGetNone(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &UserInvitationModel{
		DB: db}

	model.PrepareStatements()

	_, err := model.Get("foo@bar")
	if err == nil {
		t.Error("should have gotten error")
	} else if err != sql.ErrNoRows {
		t.Error("got error but not no-rows", err)
	}
}

func TestInsert(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &UserInvitationModel{
		DB: db}

	model.PrepareStatements()

	//create
	err := model.Create("fOo@bAr")
	if err != nil {
		t.Fatal(err)
	}

	//create another
	err = model.Create("bax@zoink")
	if err != nil {
		t.Fatal(err)
	}

	// check by getting
	invite, err := model.Get("Foo@baR")
	if err != nil {
		t.Fatal(err)
	}
	if invite.Email != "foo@bar" {
		t.Fatal("did not get right email back")
	}

	// double invite should be no-op
	err = model.Create("foO@Bar")
	if err != nil {
		t.Fatal(err)
	}

	// shoudl be just one row
	invites, err := model.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(invites) != 2 {
		t.Error("invites not empty")
	}

	// now delete
	err = model.Delete("foo@BAR")
	if err != nil {
		t.Error(err)
	}

	// shoudl be one rows
	invites, err = model.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(invites) != 1 {
		t.Error("invites not empty")
	}
}

func TestDeleteNoRows(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &UserInvitationModel{
		DB: db}

	model.PrepareStatements()

	err := model.Delete("abc@def.com")
	if err == nil {
		t.Error("expected error")
	}
	if err != domain.ErrNoRowsAffected {
		t.Error("expected error to be No rows Affected, got ", err)
	}
}
