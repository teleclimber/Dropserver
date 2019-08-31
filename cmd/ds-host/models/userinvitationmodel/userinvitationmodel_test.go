package userinvitationmodel

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/internal/dserror"
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

	invites, dsErr := model.GetAll()
	if dsErr != nil {
		t.Error(dsErr)
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

	invite, dsErr := model.Get("foo@bar")
	if dsErr == nil {
		t.Error("should have gotten error")
	} else if dsErr.Code() != dserror.NoRowsInResultSet {
		t.Error("got error but not no-rows", dsErr)
	}

	if invite != nil {
		t.Error("invite should be nil")
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
	dsErr := model.Create("fOo@bAr")
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	//create another
	dsErr = model.Create("bax@zoink")
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	// check by getting
	invite, dsErr := model.Get("Foo@baR")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if invite.Email != "foo@bar" {
		t.Fatal("did not get right email back")
	}

	// double invite should be no-op
	dsErr = model.Create("foO@Bar")
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	// shoudl be just one row
	invites, dsErr := model.GetAll()
	if dsErr != nil {
		t.Error(dsErr)
	}
	if len(invites) != 2 {
		t.Error("invites not empty")
	}

	// now delete
	dsErr = model.Delete("foo@BAR")
	if dsErr != nil {
		t.Error(dsErr)
	}

	// shoudl be one rows
	invites, dsErr = model.GetAll()
	if dsErr != nil {
		t.Error(dsErr)
	}
	if len(invites) != 1 {
		t.Error("invites not empty")
	}
}
