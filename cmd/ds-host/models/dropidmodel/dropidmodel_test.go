package dropidmodel

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

	model := &DropIDModel{
		DB: db}

	model.PrepareStatements()
}

func TestGetFromNonExistentID(t *testing.T) {
	dropIDModel := makeDropIDModel()
	defer dropIDModel.DB.Handle.Close()

	// There should be an error, but no panics
	_, err := dropIDModel.Get("nobody", "yadayada.com")
	if err == nil {
		t.Error("Expected an error")
	}
	if err != domain.ErrNoRowsInResultSet {
		t.Error(err)
	}
}

func TestCreate(t *testing.T) {
	dropIDModel := makeDropIDModel()
	defer dropIDModel.DB.Handle.Close()

	cmp := domain.DropID{
		UserID:      domain.UserID(7),
		Handle:      "me",
		Domain:      "example.com",
		DisplayName: "ollie"}

	d, err := dropIDModel.Create(cmp.UserID, cmp.Handle, cmp.Domain, cmp.DisplayName)
	if err != nil {
		t.Error(err)
	}

	cmp.Created = d.Created // make created match so we can compare structs directly.

	if d != cmp {
		t.Error("expected the same struct")
	}

	// Test fetch with different case:
	d, err = dropIDModel.Get("Me", cmp.Domain)
	if err != nil {
		t.Error(err)
	}
	if d != cmp {
		t.Error("expected the same struct")
	}
}

func TestUpdate(t *testing.T) {
	dropIDModel := makeDropIDModel()
	defer dropIDModel.DB.Handle.Close()

	cmp := domain.DropID{
		UserID:      domain.UserID(7),
		Handle:      "me",
		Domain:      "example.com",
		DisplayName: "ollie"}

	d, err := dropIDModel.Create(cmp.UserID, cmp.Handle, cmp.Domain, cmp.DisplayName)
	if err != nil {
		t.Error(err)
	}

	cmp.DisplayName = "Oscar"

	err = dropIDModel.Update(cmp.UserID, cmp.Handle, cmp.Domain, cmp.DisplayName)
	if err != nil {
		t.Error(err)
	}

	d, err = dropIDModel.Get(cmp.Handle, cmp.Domain)
	if err != nil {
		t.Error(err)
	}

	cmp.Created = d.Created // make created match so we can compare structs directly.

	if d != cmp {
		t.Error("expected the same struct")
	}
}

func TestGetForUser(t *testing.T) {
	dropIDModel := makeDropIDModel()
	defer dropIDModel.DB.Handle.Close()

	userID := domain.UserID(7)

	dropIDModel.Create(userID, "one", "domain1", "1")
	dropIDModel.Create(domain.UserID(13), "two", "domain2", "2")
	dropIDModel.Create(userID, "three", "domain3", "3")

	dropIDs, err := dropIDModel.GetForUser(userID)
	if err != nil {
		t.Error(err)
	}
	if len(dropIDs) != 2 {
		t.Error("expected 2 dropids")
	}
	if dropIDs[0].Handle != "one" && dropIDs[1].Handle != "one" {
		t.Error("didn't get the drop ids I was expecting.")
	}
	if dropIDs[0].Handle != "three" && dropIDs[1].Handle != "three" {
		t.Error("didn't get the drop ids I was expecting.")
	}
}

func TestDelete(t *testing.T) {
	dropIDModel := makeDropIDModel()
	defer dropIDModel.DB.Handle.Close()

	userID := domain.UserID(7)

	_, err := dropIDModel.Create(userID, "one", "domain1", "1")
	if err != nil {
		t.Error(err)
	}
	err = dropIDModel.Delete(userID, "one", "domain1")
	if err != nil {
		t.Error(err)
	}

	_, err = dropIDModel.Get("one", "domain1")
	if err == nil {
		t.Error("expected error")
	}
	if err != domain.ErrNoRowsInResultSet {
		t.Error("expected error to be no rows")
	}
}

func makeDropIDModel() *DropIDModel {
	dropIDModel := &DropIDModel{
		DB: &domain.DB{
			Handle: migrate.MakeSqliteDummyDB()}}

	dropIDModel.PrepareStatements()

	return dropIDModel
}
