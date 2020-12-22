package contactmodel

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

	model := &ContactModel{
		DB: db}

	model.PrepareStatements()
}

func TestGetFromNonExistentID(t *testing.T) {
	contactModel := makeContactModel()
	defer contactModel.DB.Handle.Close()

	// There should be an error, but no panics
	_, err := contactModel.Get(domain.UserID(7), domain.ContactID(11))
	if err == nil {
		t.Error("Expected an error")
	}
	if err != sql.ErrNoRows {
		t.Error(err)
	}
}

func TestGetAllEmptyTable(t *testing.T) {
	contactModel := makeContactModel()
	defer contactModel.DB.Handle.Close()

	// There should be an error, but no panics
	ret, err := contactModel.GetForUser(domain.UserID(7))
	if err != nil {
		t.Error(err)
	}

	if len(ret) != 0 {
		t.Error("expected zero length")
	}
}

func TestCreateContact(t *testing.T) {
	contactModel := makeContactModel()
	defer contactModel.DB.Handle.Close()

	userID := domain.UserID(7)
	name := "Jim Bob"
	displayName := "jimmie"

	contact, err := contactModel.Create(userID, name, displayName)
	if err != nil {
		t.Error(err)
	}
	if contact.ContactID == domain.ContactID(0) {
		t.Error("got zero valud for contact")
	}
	if contact.Name != name || contact.DisplayName != displayName {
		t.Error("mismatched names")
	}
}

func TestUpdateContact(t *testing.T) {
	contactModel := makeContactModel()
	defer contactModel.DB.Handle.Close()

	userID := domain.UserID(7)
	name := "Jim Bob"
	displayName := "jimmie"

	contact, err := contactModel.Create(userID, "Jack", displayName)
	if err != nil {
		t.Error(err)
	}

	err = contactModel.Update(userID, contact.ContactID, name, contact.DisplayName)
	if err != nil {
		t.Error(err)
	}

	contact, err = contactModel.Get(userID, contact.ContactID)
	if err != nil {
		t.Error(err)
	}
	if contact.Name != name {
		t.Error("expected name")
	}
}

func TestMultiContact(t *testing.T) {
	contactModel := makeContactModel()
	defer contactModel.DB.Handle.Close()

	userID := domain.UserID(7)

	contactModel.Create(userID, "alice", "alice")
	c2, err := contactModel.Create(userID, "bob", "bob")
	contactModel.Create(userID, "charlie", "charlie")

	all, err := contactModel.GetForUser(userID)
	if err != nil {
		t.Error(err)
	}
	if len(all) != 3 {
		t.Error("expected 3")
	}

	err = contactModel.Delete(userID, c2.ContactID)
	if err != nil {
		t.Error(err)
	}

	all, err = contactModel.GetForUser(userID)
	if err != nil {
		t.Error(err)
	}
	if len(all) != 2 {
		t.Error("expected 2")
	}
}

func TestAppspaceContactNoRows(t *testing.T) {
	contactModel := makeContactModel()
	defer contactModel.DB.Handle.Close()

	appspaceID := domain.AppspaceID(11)
	contactID := domain.ContactID(7)
	_, err := contactModel.GetContactProxy(appspaceID, contactID)
	if err == nil {
		t.Error("expected error no rows")
	} else if err != sql.ErrNoRows {
		t.Error(err)
	}

	_, err = contactModel.GetByProxy(appspaceID, domain.ProxyID("abc"))
	if err == nil {
		t.Error("expected error no rows")
	} else if err != sql.ErrNoRows {
		t.Error(err)
	}
}

func TestAppspaceContact(t *testing.T) {
	contactModel := makeContactModel()
	defer contactModel.DB.Handle.Close()

	appspaceID := domain.AppspaceID(11)
	contactID := domain.ContactID(7)
	proxyID := domain.ProxyID("abc")
	err := contactModel.InsertAppspaceContact(appspaceID, contactID, proxyID)
	if err != nil {
		t.Error(err)
	}

	err = contactModel.InsertAppspaceContact(appspaceID, contactID, proxyID)
	if err == nil {
		t.Error("expected error because exact dupe")
	}

	contactID2 := domain.ContactID(77)
	proxyID2 := domain.ProxyID("def")
	err = contactModel.InsertAppspaceContact(appspaceID, contactID2, proxyID)
	if err == nil {
		t.Error("expected error because dupe proxy ID")
	}

	err = contactModel.InsertAppspaceContact(appspaceID, contactID2, proxyID2)
	if err != nil {
		t.Error(err)
	}

	p, err := contactModel.GetContactProxy(appspaceID, contactID)
	if err != nil {
		t.Error(err)
	}
	if p != proxyID {
		t.Error("got wrong proxy ID")
	}

	c, err := contactModel.GetByProxy(appspaceID, proxyID2)
	if err != nil {
		t.Error(err)
	}
	if c != contactID2 {
		t.Error("got wrong contact ID")
	}

	ac, err := contactModel.GetContactAppspaces(contactID)
	if err != nil {
		t.Error(err)
	}
	if len(ac) != 1 {
		t.Error("expected 1 appspace contact")
	}
	if ac[0].ProxyID != proxyID {
		t.Error("wrong proxy ID")
	}

	ac, err = contactModel.GetAppspaceContacts(appspaceID)
	if err != nil {
		t.Error(err)
	}
	if len(ac) != 2 {
		t.Error("expected 2 appspace contact")
	}

	err = contactModel.DeleteAppspaceContact(appspaceID, contactID2)
	if err != nil {
		t.Error(err)
	}

	ac, err = contactModel.GetAppspaceContacts(appspaceID)
	if err != nil {
		t.Error(err)
	}
	if len(ac) != 1 {
		t.Error("expected 1 appspace contact")
	}
}

func makeContactModel() *ContactModel {
	contactModel := &ContactModel{
		DB: &domain.DB{
			Handle: migrate.MakeSqliteDummyDB()}}

	contactModel.PrepareStatements()

	return contactModel
}
