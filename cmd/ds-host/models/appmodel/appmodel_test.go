package appmodel

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

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()
}

func TestGetFromNonExistentID(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	// There should be an error, but no panics
	_, err := appModel.GetFromID(10)
	if err == nil {
		t.Error(err)
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

	app, err := appModel.Create(domain.UserID(1), "test-app")
	if err != nil {
		t.Error(err)
	}

	if app.Name != "test-app" {
		t.Error("input name does not match output name", app)
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

	_, err := appModel.Create(7, "test-app")
	if err != nil {
		t.Error(err)
	}

	// There should now be one row so app id 1 should return something
	app, err := appModel.GetFromID(1)
	if err != nil {
		t.Error(err)
	}

	if app.AppID != 1 {
		t.Error("app.AppID does not match requested ID", app)
	}
}

func TestGetOwner(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	apps, err := appModel.GetForOwner(7)
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.Create(7, "test-app")
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.Create(7, "test-app2")
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.Create(11, "bad-app")
	if err != nil {
		t.Error(err)
	}

	apps, err = appModel.GetForOwner(7)
	if err != nil {
		t.Error(err)
	}

	if len(apps) != 2 {
		t.Error("There should be two apps found")
	}
}

func TestDelete(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	app, err := appModel.Create(7, "test-app")
	if err != nil {
		t.Error(err)
	}

	err = appModel.Delete(app.AppID)
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.GetFromID(app.AppID)
	if err != sql.ErrNoRows {
		t.Error("expecte err no rows")
	}
}

func TestDeleteWithVersions(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	app, err := appModel.Create(7, "test-app")
	if err != nil {
		t.Error(err)
	}
	_, err = appModel.CreateVersion(app.AppID, domain.Version("1.2.3"), 0, domain.APIVersion(0), "loc")
	if err != nil {
		t.Error(err)
	}

	err = appModel.Delete(app.AppID)
	if err == nil {
		t.Error("expected error")
	}
}

func TestVersion(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	appVersion, err := appModel.CreateVersion(1, "0.0.1", 7, 1, "foo-location")
	if err != nil {
		t.Error(err)
	}

	if appVersion.Version != "0.0.1" {
		t.Error("input version does not match output version", appVersion)
	}
	if appVersion.Schema != 7 {
		t.Error("input schema does not match output schema", appVersion)
	}
	if appVersion.APIVersion != 1 {
		t.Error("input does not match output api version", appVersion)
	}
	if appVersion.LocationKey != "foo-location" {
		t.Error("input does not match output location key", appVersion)
	}

	// just go ahead and test GetVersion here for completeness
	appVersion, err = appModel.GetVersion(1, "0.0.1")
	if err != nil {
		t.Error(err)
	}
	if appVersion.Version != "0.0.1" {
		t.Error("input version does not match output version", appVersion)
	}
	if appVersion.LocationKey != "foo-location" {
		t.Error("input does not match output location key", appVersion)
	}

	// test to make sure we get right error if no rows
	_, err = appModel.GetVersion(1, "0.0.13")
	if err == nil || err != sql.ErrNoRows {
		t.Fatal("should have been no rows error", err)
	}

	// then test inserting a duplicate version
	_, err = appModel.CreateVersion(1, "0.0.1", 7, 1, "bar-location")
	if err == nil {
		t.Error("expected error on inserting duplicate")
	}

}

func TestGetVersionForApp(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	ins := []struct {
		appID    domain.AppID
		version  domain.Version
		schema   int
		location string
	}{
		{7, "0.0.2", 2, "2foo-location"},
		{7, "0.0.3", 3, "3foo-location"},
		{7, "0.0.1", 1, "1foo-location"},
		{11, "0.0.1", 1, "bar-location"},
	}

	for _, i := range ins {
		_, err := appModel.CreateVersion(i.appID, i.version, i.schema, 1, i.location)
		if err != nil {
			t.Error(err)
		}
	}

	vers, err := appModel.GetVersionsForApp(7)
	if err != nil {
		t.Error(err)
	}
	if len(vers) != 3 {
		t.Error("Got wrong number of results: should be 3")
	}
	if vers[0].Version != domain.Version("0.0.1") {
		t.Error("Go wrong sort order")
	}

	vers, err = appModel.GetVersionsForApp(1)
	if err != nil {
		t.Error(err)
	}
	if len(vers) != 0 {
		t.Error("Got wrong number of results: should be 0")
	}
}

func TestDeleteVersion(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	appID := domain.AppID(7)
	version := domain.Version("0.0.2")

	_, err := appModel.CreateVersion(appID, version, 4, 1, "foobar")
	if err != nil {
		t.Fatal(err)
	}

	err = appModel.DeleteVersion(appID, version)
	if err != nil {
		t.Fatal(err)
	}

	_, err = appModel.GetVersion(appID, version)
	if err == nil || err != sql.ErrNoRows {
		t.Fatal("should have been no rows error", err)
	}
}
