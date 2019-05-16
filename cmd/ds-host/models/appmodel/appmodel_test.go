package appmodel

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

// Issues with testing models against a live (in-memory DB)
// - need to consistently create a db that is at the latest schema
// - this needs to be done in a test rig close to migration so it can be reused in all models
// - in-memory dbs are tiesd to a single connection and vanish when it closes. So watch out for that.

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()
}

func TestGetFromIDError(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	// There should be an error, but no panics
	_, dsErr := appModel.GetFromID(10)
	if dsErr == nil {
		t.Error(dsErr)
	}
}

func TestCreate(t *testing.T) {

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	app, dsErr := appModel.Create("test-app", "some-location")
	if dsErr != nil {
		t.Error(dsErr)
	}

	if app.Name != "test-app" {
		t.Error("input name does not match output name", app)
	}
	if app.LocationKey != "some-location" {
		t.Error("input location key does not match output")
	}
}

func TestGetFromID(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	_, dsErr := appModel.Create("test-app", "some-location")
	if dsErr != nil {
		t.Error(dsErr)
	}

	// There should now be one row so app id 1 should return something
	app, dsErr := appModel.GetFromID(1)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if app.AppID != 1 {
		t.Error("app.AppID does not match requested ID", app)
	}
}
