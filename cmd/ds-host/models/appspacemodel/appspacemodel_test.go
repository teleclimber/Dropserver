package appspacemodel

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

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()
}

func TestGetFromIDError(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	// There should be an error, but no panics
	_, dsErr := model.GetFromID(10)
	if dsErr == nil {
		t.Error(dsErr)
	}
}

func TestCreate(t *testing.T) {

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	appspace, dsErr := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace")
	if dsErr != nil {
		t.Error(dsErr)
	}

	if appspace.OwnerID != domain.UserID(1) {
		t.Error("input does not match output ownerID", appspace)
	}
	if appspace.AppID != domain.AppID(10) {
		t.Error("input does not match output appID", appspace)
	}
	if appspace.AppVersion != domain.Version("0.0.1") {
		t.Error("input does not match output version", appspace)
	}
	if appspace.Subdomain != "test-appspace" {
		t.Error("input name does not match output subdomain", appspace)
	}
	if appspace.Paused {
		t.Error("appspace should be created not paused by default", appspace)
	}
}

func TestGetFromID(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	_, dsErr := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace")
	if dsErr != nil {
		t.Error(dsErr)
	}

	// There should now be one row so app id 1 should return something
	appspace, dsErr := model.GetFromID(domain.AppspaceID(1))
	if dsErr != nil {
		t.Error(dsErr)
	}

	if appspace.AppspaceID != domain.AppspaceID(1) {
		t.Error("app.AppID does not match requested ID", appspace)
	}
}

func TestGetFromSubdomain(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	_, dsErr := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace")
	if dsErr != nil {
		t.Error(dsErr)
	}

	_, dsErr = model.GetFromSubdomain("test-appspace")
	if dsErr != nil {
		t.Error(dsErr)
	}

	// test non-existent subdomain
	_, dsErr = model.GetFromSubdomain("foo")
	if dsErr == nil {
		t.Error("Should have errored trying to get non-existent subdomain")
	} else if dsErr.Code() != dserror.NoRowsInResultSet {
		t.Error("wrong error for non-existent subdomain: ", dsErr)
	}
}

func TestGetForOwner(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	ins := []struct {
		userID   domain.UserID
		appID    domain.AppID
		version  domain.Version
		location string
	}{
		{7, 4, "0.0.1", "foo-location"},
		{7, 5, "0.0.2", "2foo-location"},
		{7, 6, "0.0.3", "3foo-location"},
		{11, 6, "0.0.1", "bar-location"},
	}

	for _, i := range ins {
		_, dsErr := model.Create(i.userID, i.appID, i.version, i.location)
		if dsErr != nil {
			t.Error(dsErr)
		}
	}

	appSpaces, dsErr := model.GetForOwner(7)
	if dsErr != nil {
		t.Error(dsErr)
	}
	if len(appSpaces) != 3 {
		t.Error("expected 3 appspaces")
	}

	appSpaces, dsErr = model.GetForOwner(1)
	if dsErr != nil {
		t.Error(dsErr)
	}
	if len(appSpaces) != 0 {
		t.Error("expected ZERO appspaces")
	}
}

func TestCreateDupeSubdomain(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	_, dsErr := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace")
	if dsErr != nil {
		t.Error(dsErr)
	}

	_, dsErr = model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace")
	if dsErr == nil {
		t.Error("There should have been an error for duplicate subdomain")
	} else if dsErr.Code() != dserror.DomainNotUnique {
		t.Error("Wrong error", dsErr)
	}
}

func TestGetForApp(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	ins := []struct {
		userID   domain.UserID
		appID    domain.AppID
		version  domain.Version
		location string
	}{
		{7, 4, "0.0.1", "foo-location"},
		{7, 5, "0.0.2", "2foo-location"},
		{7, 6, "0.0.3", "3foo-location"},
		{11, 6, "0.0.1", "bar-location"},
	}

	for _, i := range ins {
		_, dsErr := model.Create(i.userID, i.appID, i.version, i.location)
		if dsErr != nil {
			t.Error(dsErr)
		}
	}

	appSpaces, dsErr := model.GetForApp(6)
	if dsErr != nil {
		t.Error(dsErr)
	}
	if len(appSpaces) != 2 {
		t.Error("expected 2 appspaces")
	}

	appSpaces, dsErr = model.GetForApp(1)
	if dsErr != nil {
		t.Error(dsErr)
	}
	if len(appSpaces) != 0 {
		t.Error("expected ZERO appspaces")
	}
}

func TestPause(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	appspace, dsErr := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace")
	if dsErr != nil {
		t.Error(dsErr)
	}

	dsErr = model.Pause(appspace.AppspaceID, true)
	if dsErr != nil {
		t.Error(dsErr)
	}

	appspace, dsErr = model.GetFromID(appspace.AppspaceID)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if !appspace.Paused {
		t.Error("appspace should be paused")
	}

}
