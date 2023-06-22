package appmodel

import (
	"strings"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/google/go-cmp/cmp"
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

	_, err := appModel.Create(domain.UserID(1))
	if err != nil {
		t.Error(err)
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

	_, err := appModel.Create(7)
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

	_, err := appModel.GetForOwner(7)
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.Create(7)
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.Create(7)
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.Create(11)
	if err != nil {
		t.Error(err)
	}

	apps, err := appModel.GetForOwner(7)
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

	appID, err := appModel.Create(7)
	if err != nil {
		t.Error(err)
	}

	err = appModel.Delete(appID)
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.GetFromID(appID)
	if err != domain.ErrNoRowsInResultSet {
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

	appID, err := appModel.Create(7)
	if err != nil {
		t.Error(err)
	}
	_, err = appModel.CreateVersion(appID, "loc", domain.AppVersionManifest{
		Version: domain.Version("1.2.3"),
		Schema:  0})
	if err != nil {
		t.Error(err)
	}

	err = appModel.Delete(appID)
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

	appVersion, err := appModel.CreateVersion(1, "foo-location", domain.AppVersionManifest{
		Name:             "Test App",
		Version:          domain.Version("0.0.1"),
		Schema:           7,
		ShortDescription: "abc-123",
		Icon:             "icon.icon",
		AccentColor:      "#aabbcc"})
	if err != nil {
		t.Error(err)
	}

	if appVersion.Version != "0.0.1" {
		t.Error("input version does not match output version", appVersion)
	}
	if appVersion.Schema != 7 {
		t.Error("input schema does not match output schema", appVersion)
	}
	if appVersion.LocationKey != "foo-location" {
		t.Error("input does not match output location key", appVersion)
	}

	// test GetVersion here
	appVersionOut, err := appModel.GetVersion(1, "0.0.1")
	if err != nil {
		t.Error(err)
	}
	if appVersionOut.Version != "0.0.1" {
		t.Error("input version does not match output version", appVersionOut)
	}
	if appVersionOut.LocationKey != "foo-location" {
		t.Error("input does not match output location key", appVersionOut)
	}

	// test getting app version manifest
	manifestStr, err := appModel.GetVersionManifestJSON(1, "0.0.1")
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(manifestStr, "\"schema\":7") {
		t.Error("expected json to contain schema, got: " + manifestStr)
	}

	manifest, err := appModel.GetVersionManifest(1, "0.0.1")
	if err != nil {
		t.Error(err)
	}
	if manifest.Version != "0.0.1" {
		t.Error("input version does not match manifest version", manifest)
	}
	if manifest.Schema != 7 {
		t.Error("input does not match manifest schema key", manifest)
	}

	// test to make sure we get right error if no rows
	_, err = appModel.GetVersion(1, "0.0.13")
	if err == nil || err != domain.ErrNoRowsInResultSet {
		t.Fatal("should have been no rows error", err)
	}

	// then test inserting a duplicate version
	_, err = appModel.CreateVersion(1, "bar-location", domain.AppVersionManifest{
		Version: domain.Version("0.0.1"),
		Schema:  7})
	if err == nil {
		t.Error("expected error on inserting duplicate")
	}

}

func TestGetVersionUI(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	authors := []domain.ManifestAuthor{{
		Name:  "OF",
		Email: "me@of",
		URL:   "of.net"}}

	appVersion, err := appModel.CreateVersion(1, "loc", domain.AppVersionManifest{
		Name:             "Test App",
		Version:          domain.Version("0.0.1"),
		Schema:           7,
		ShortDescription: "abc-123",
		Icon:             "icon.icon",
		AccentColor:      "#aabbcc",
		Authors:          authors,
		Website:          "https://website",
		Code:             "https://code",
		Funding:          "https://finding",
		ReleaseDate:      "2023-06-21",
		License:          "MIT"})
	if err != nil {
		t.Error(err)
	}

	// then insert one with a minimal manifest (version and schema included only)
	// to ensure that missing values in manifest don't cause errors.
	manifest := `{"version":"0.0.2", "schema": 9}`
	_, err = h.Exec(`INSERT INTO app_versions
		(app_id, version, location_key, manifest, created) VALUES (?, ?, ?, json(?), datetime("now"))`,
		1, "0.0.2", "loc2", manifest)
	if err != nil {
		t.Error(err)
	}

	appVersionUIOut, err := appModel.GetVersionForUI(1, "0.0.1")
	if err != nil {
		t.Error(err)
	}
	expectedVerUI := domain.AppVersionUI{
		AppID:            1,
		Name:             "Test App",
		Version:          domain.Version("0.0.1"),
		Schema:           7,
		ShortDescription: "abc-123",
		AccentColor:      "#aabbcc",
		Created:          appVersion.Created, // cheat on created because it's just easier that way.
		Authors:          authors,
		Website:          "https://website",
		Code:             "https://code",
		Funding:          "https://finding",
		ReleaseDate:      "2023-06-21",
		License:          "MIT",
	}
	if !cmp.Equal(expectedVerUI, appVersionUIOut) {
		t.Errorf("app version out unexpected: %v, %v", expectedVerUI, appVersionUIOut)
	}

	appVersionUIOut, err = appModel.GetVersionForUI(1, "0.0.2")
	if err != nil {
		t.Error(err)
	}
	if appVersionUIOut.Authors == nil {
		t.Error("authors should not be nil")
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
		_, err := appModel.CreateVersion(i.appID, i.location, domain.AppVersionManifest{
			Version: i.version,
			Schema:  i.schema})
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

func TestSortVersions(t *testing.T) {
	vers := []string{"0.3.0", "0.0.1", "5.1.2"}
	pVers := make([]parsedVersion, len(vers))
	for i, v := range vers {
		p, _ := semver.Parse(v)
		pVers[i] = parsedVersion{domain.Version(v), p}
	}
	sortVersions(pVers)
	if pVers[0].dom != domain.Version("0.0.1") {
		t.Log(pVers)
		t.Error("expected 0.0.1 to be the 0-index version")
	}
	if pVers[2].dom != domain.Version("5.1.2") {
		t.Log(pVers)
		t.Error("expected 5.1.2 to be the last-index version")
	}
}

func TestGetParsedVersions(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}
	appModel.PrepareStatements()

	ins := []domain.Version{"5.1.2", "0.0.1"}
	for _, i := range ins {
		_, err := h.Exec(`INSERT INTO app_versions (app_id, version) VALUES (1, ?)`, i)
		if err != nil {
			t.Fatal(err)
		}
	}

	pVers, err := appModel.getParsedVersions(1)
	if err != nil {
		t.Error(err)
	}

	if pVers[0].dom != domain.Version("0.0.1") ||
		!cmp.Equal(pVers[0].parsed, semver.Version{Major: 0, Minor: 0, Patch: 1}) {
		t.Log(pVers)
		t.Error("didn't get expected result")
	}
	if pVers[1].dom != domain.Version("5.1.2") ||
		!cmp.Equal(pVers[1].parsed, semver.Version{Major: 5, Minor: 1, Patch: 2}) {
		t.Log(pVers)
		t.Error("didn't get expected result")
	}
}

func TestGetCurrentVersion(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}
	appModel.PrepareStatements()

	_, err := appModel.GetCurrentVersion(1)
	if err != domain.ErrNoRowsInResultSet {
		t.Error("expected domain.ErroNoRowsInResultSet")
	}

	ins := []domain.Version{"5.1.2", "0.0.1"}
	for _, i := range ins {
		_, err := h.Exec(`INSERT INTO app_versions (app_id, version) VALUES (1, ?)`, i)
		if err != nil {
			t.Fatal(err)
		}
	}

	curVer, err := appModel.GetCurrentVersion(1)
	if err != nil {
		t.Error(err)
	}
	if curVer != domain.Version("5.1.2") {
		t.Errorf("expected 5.1.2 got %v", curVer)
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

	_, err := appModel.CreateVersion(appID, "foobar", domain.AppVersionManifest{
		Version: version,
		Schema:  4})
	if err != nil {
		t.Fatal(err)
	}

	err = appModel.DeleteVersion(appID, version)
	if err != nil {
		t.Fatal(err)
	}

	_, err = appModel.GetVersion(appID, version)
	if err == nil || err != domain.ErrNoRowsInResultSet {
		t.Fatal("should have been no rows error", err)
	}
}
