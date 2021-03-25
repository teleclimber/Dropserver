package remoteappspacemodel

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &RemoteAppspaceModel{
		DB: db}

	model.PrepareStatements()
}

func TestCreateGet(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &RemoteAppspaceModel{
		DB: &domain.DB{
			Handle: h}}
	model.PrepareStatements()

	userID := domain.UserID(7)
	dom := "abc.def"
	dropid := "ooo.name/hi"

	_, err := model.Get(userID, dom)
	if err == nil {
		t.Error("expected error")
	}
	if err != sql.ErrNoRows {
		t.Error(err)
	}

	err = model.Create(userID, dom, "", dropid)
	if err != nil {
		t.Error(err)
	}

	r, err := model.Get(userID, dom)
	if err != nil {
		t.Error(err)
	}
	if r.UserDropID != dropid {
		t.Error("got the wrong drop id")
	}
}

func TestGetForUser(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &RemoteAppspaceModel{
		DB: &domain.DB{
			Handle: h}}
	model.PrepareStatements()

	userID := domain.UserID(7)

	err := model.Create(userID, "abc.def", "", "")
	if err != nil {
		t.Error(err)
	}
	err = model.Create(userID, "ggg.def", "", "")
	if err != nil {
		t.Error(err)
	}
	err = model.Create(domain.UserID(13), "abc.def", "", "")
	if err != nil {
		t.Error(err)
	}

	remotes, err := model.GetForUser(userID)
	if err != nil {
		t.Error(err)
	}
	if len(remotes) != 2 {
		t.Error("expected two remotes")
	}
}

func TestDuplicateRemote(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &RemoteAppspaceModel{
		DB: &domain.DB{
			Handle: h}}
	model.PrepareStatements()

	userID := domain.UserID(7)

	err := model.Create(userID, "abc.def", "", "")
	if err != nil {
		t.Error(err)
	}
	err = model.Create(userID, "abc.def", "", "")
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Error("expect error string to contain: already exists")
	}
}

func TestDelete(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &RemoteAppspaceModel{
		DB: &domain.DB{
			Handle: h}}
	model.PrepareStatements()

	userID := domain.UserID(7)
	dom := "abc.def"
	dropid := "ooo.name/hi"

	err := model.Create(userID, dom, "", dropid)
	if err != nil {
		t.Error(err)
	}

	_, err = model.Get(userID, dom)
	if err != nil {
		t.Error(err)
	}

	err = model.Delete(userID, dom)
	if err != nil {
		t.Error(err)
	}

	_, err = model.Get(userID, dom)
	if err == nil {
		t.Error("expected error gettign deleted remote")
	}
	if err != sql.ErrNoRows {
		t.Error("expected error to be sql no rows")
	}
}
