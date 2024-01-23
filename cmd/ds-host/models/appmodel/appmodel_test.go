package appmodel

import (
	"strings"
	"testing"
	"time"

	"github.com/blang/semver/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/internal/nulltypes"
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

// TestTxRollback tests that the code pattern used for creating transactions
// and rolling them back on error does actually work as intended.
// This is the recommended pattern, but I just want to be sure.
func TestTxRollback(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()
	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}

	appID, err := appModel.CreateFromURL(11, "abc.com/app", true, domain.AppListingFetch{})
	if err != nil {
		t.Error(err)
	}

	func() {
		tx, err := h.Beginx()
		if err != nil {
			t.Error(err)
		}
		defer tx.Rollback()

		err = setNewUrl(appID, "new.url/app", nulltypes.NewTime(time.Now(), true), tx)
		if err != nil {
			t.Error(err)
		}
		// imagine more setters getting called...
		// if an error is detected, the function returns early and tx.Commit() never gets called.
		// tx.Commit()	// commented on purpose.
	}()

	urlData, err := appModel.GetAppUrlData(appID)
	if err != nil {
		t.Error(err)
	}
	if urlData.NewURL != "" {
		t.Error("expected rollback to prevent new_ulr col from being written")
	}
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

func TestDeleteWithAppUrl(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	appID, err := appModel.CreateFromURL(7, "abc.com", true, domain.AppListingFetch{})
	if err != nil {
		t.Error(err)
	}

	err = appModel.Delete(appID)
	if err != nil {
		t.Error(err)
	}

	_, err = appModel.GetAppUrlData(appID)
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

func TestGetUrlDataNone(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	urlData, err := appModel.GetAppUrlData(domain.AppID(1))
	if err == nil {
		t.Log(urlData)
		t.Error("expected error because there is no row")
	} else if err != domain.ErrNoRowsInResultSet {
		t.Errorf("expected no rows in result set error, got %v", err)
	}

	listing, urlData, err := appModel.GetAppUrlListing(domain.AppID(1))
	if err == nil {
		t.Log(urlData, listing)
		t.Error("expected error because there is no row")
	} else if err != domain.ErrNoRowsInResultSet {
		t.Errorf("expected no rows in result set error, got %v", err)
	}
}

func TestInternalCreateAppUrlData(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	tx, err := h.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	err = createAppUrlData(domain.AppID(1), "abc.com/app", true, tx)
	if err != nil {
		t.Error(err)
	}
	err = tx.Commit()
	if err != nil {
		t.Error(err)
	}

	urlData, err := appModel.GetAppUrlData(domain.AppID(1))
	if err == nil {
		t.Log(urlData)
		t.Error("expected error because we haven't populated all the columns")
	}
}

func TestCreateAppUrlData(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	dt := time.Now()
	expected := domain.AppURLData{
		URL:             "abc.com/app",
		Automatic:       true,
		Last:            dt,
		LastResult:      "ok",
		NewURL:          "",
		NewUrlDatetime:  nulltypes.NewTime(time.Time{}, false),
		ListingDatetime: dt,
		Etag:            "etag-abc",
		LatestVersion:   "1.2.3"}

	expectedListing := domain.AppListing{
		Base: "abc.com/app",
		Versions: map[domain.Version]domain.AppListingVersion{
			domain.Version("1.2.3"): {
				Manifest:  "manifest-1.2.3.json",
				Package:   "manifest-1.2.3.tar.gz",
				Changelog: "changelog-1.2.3.txt",
				Icon:      "icon-1.2.3.gif"},
			domain.Version("1.5.0"): {
				Manifest:  "manifest-1.5.0.json",
				Package:   "manifest-1.5.0.tar.gz",
				Changelog: "changelog-1.5.0.txt",
				Icon:      "icon-1.5.0.gif"},
		},
	}

	lf := domain.AppListingFetch{
		FetchDatetime:   dt,
		Listing:         expectedListing,
		ListingDatetime: dt,
		Etag:            expected.Etag,
		LatestVersion:   expected.LatestVersion}

	appID, err := appModel.CreateFromURL(domain.UserID(1), expected.URL, expected.Automatic, lf)
	if err != nil {
		t.Error(err)
	}
	expected.AppID = appID

	urlData, err := appModel.GetAppUrlData(appID)
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(urlData, expected) {
		t.Error(cmp.Diff(urlData, expected))
	}

	listing, urlData, err := appModel.GetAppUrlListing(appID)
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(urlData, expected) {
		t.Error(cmp.Diff(urlData, expected))
	}
	if !cmp.Equal(listing, expectedListing) {
		t.Error(cmp.Diff(listing, expectedListing))
	}
}

func TestGetAutoUrlDataByLastDt(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	// Test with no data:
	_, err := appModel.GetAutoUrlDataByLastDt(time.Now())
	if err != nil {
		t.Error(err)
	}

	appID := domain.AppID(7)

	last := time.Now().Add(-time.Second)

	tx, _ := h.Beginx()
	createAppUrlData(appID, "abc", true, tx)
	setLast(appID, "ok", last, tx)
	tx.Commit()

	appIDs, err := appModel.GetAutoUrlDataByLastDt(last.Add(-time.Second))
	if err != nil {
		t.Error(err)
	}
	if len(appIDs) != 0 {
		t.Errorf("expected no app ids, got %v", appIDs)
	}

	appIDs, err = appModel.GetAutoUrlDataByLastDt(time.Now())
	if err != nil {
		t.Error(err)
	}
	if len(appIDs) != 1 {
		t.Errorf("expected one app ids, got %v", appIDs)
	}
}

func TestAppUrlUpdates(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	dt1 := time.Now()
	url := "abc.com/app"

	lf := domain.AppListingFetch{
		FetchDatetime:   dt1,
		Listing:         domain.AppListing{},
		ListingDatetime: dt1,
		Etag:            "etag1",
		LatestVersion:   "0.0.1"}

	appID, err := appModel.CreateFromURL(domain.UserID(1), url, false, lf)
	if err != nil {
		t.Error(err)
	}

	// Test UpdtaeAutomatic
	err = appModel.UpdateAutomatic(appID, true)
	if err != nil {
		t.Error(err)
	}
	urlData, err := appModel.GetAppUrlData(appID)
	if err != nil {
		t.Error(err)
	}
	expected := domain.AppURLData{
		AppID:           appID,
		URL:             url,
		Automatic:       true,
		Last:            dt1,
		LastResult:      "ok",
		NewURL:          "",
		NewUrlDatetime:  nulltypes.NewTime(time.Time{}, false),
		ListingDatetime: dt1,
		Etag:            "etag1",
		LatestVersion:   "0.0.1"}
	if !cmp.Equal(urlData, expected) {
		t.Error(cmp.Diff(urlData, expected))
	}

	// Test SetLastFetch
	dt2 := dt1.Add(time.Second)
	err = appModel.SetLastFetch(appID, dt2, "no-change")
	if err != nil {
		t.Error(err)
	}
	expected.Last = dt2
	expected.LastResult = "no-change"
	urlData, _ = appModel.GetAppUrlData(appID)
	if !cmp.Equal(urlData, expected) {
		t.Error(cmp.Diff(urlData, expected))
	}

	// Test SetNewURL
	dt3 := dt2.Add(time.Second)
	newUrl := "abc.new/app"
	err = appModel.SetNewUrl(appID, newUrl, dt3)
	if err != nil {
		t.Error(err)
	}
	expected.Last = dt3
	expected.NewURL = newUrl
	expected.NewUrlDatetime = nulltypes.NewTime(dt3, true)
	expected.LastResult = "ok"
	urlData, _ = appModel.GetAppUrlData(appID)
	if !cmp.Equal(urlData, expected) {
		t.Error(cmp.Diff(urlData, expected))
	}
}

func TestSetListing(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	dt1 := time.Now()
	url := "abc.com/app"

	lf := domain.AppListingFetch{
		FetchDatetime:   dt1,
		Listing:         domain.AppListing{},
		ListingDatetime: dt1,
		Etag:            "etag1",
		LatestVersion:   "0.0.1"}

	appID, err := appModel.CreateFromURL(domain.UserID(1), url, false, lf)
	if err != nil {
		t.Error(err)
	}

	// set New URL so we can check that gets reset
	err = setNewUrl(appID, "new.url/app", nulltypes.NewTime(dt1, true), db.Handle)
	if err != nil {
		t.Error(err)
	}

	// Test SetListing
	dt2 := dt1.Add(time.Second)
	expectedListing := domain.AppListing{
		Base: "abc.com/app",
		Versions: map[domain.Version]domain.AppListingVersion{
			domain.Version("1.2.3"): {
				Manifest:  "manifest-1.2.3.json",
				Package:   "manifest-1.2.3.tar.gz",
				Changelog: "changelog-1.2.3.txt",
				Icon:      "icon-1.2.3.gif"},
			domain.Version("1.5.0"): {
				Manifest:  "manifest-1.5.0.json",
				Package:   "manifest-1.5.0.tar.gz",
				Changelog: "changelog-1.5.0.txt",
				Icon:      "icon-1.5.0.gif"},
		},
	}
	lf.FetchDatetime = dt2
	lf.Listing = expectedListing
	lf.ListingDatetime = dt2
	lf.Etag = "etag2"
	lf.LatestVersion = domain.Version("1.5.0")

	err = appModel.SetListing(appID, lf)
	if err != nil {
		t.Error(err)
	}
	listing, urlData, _ := appModel.GetAppUrlListing(appID)
	expected := domain.AppURLData{
		AppID:           appID,
		URL:             url,
		Automatic:       false,
		Last:            dt2,
		LastResult:      "ok",
		NewURL:          "",
		NewUrlDatetime:  nulltypes.NewTime(time.Time{}, false),
		ListingDatetime: dt2,
		Etag:            "etag2",
		LatestVersion:   "1.5.0"}
	if !cmp.Equal(urlData, expected) {
		t.Error(cmp.Diff(urlData, expected))
	}
	if !cmp.Equal(listing, expectedListing) {
		t.Error(cmp.Diff(listing, expectedListing))
	}
}

func TestUpdateUrl(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}
	appModel := &AppModel{
		DB: db}

	appModel.PrepareStatements()

	dt1 := time.Now()
	url1 := "abc.com/app"
	url2 := "new.site/app"

	lf := domain.AppListingFetch{
		FetchDatetime:   dt1,
		Listing:         domain.AppListing{},
		ListingDatetime: dt1,
		Etag:            "etag1",
		LatestVersion:   "0.0.1"}

	appID, err := appModel.CreateFromURL(domain.UserID(1), url1, false, lf)
	if err != nil {
		t.Error(err)
	}

	// set New URL so we can check that gets reset
	err = setNewUrl(appID, url2, nulltypes.NewTime(dt1, true), db.Handle)
	if err != nil {
		t.Error(err)
	}

	dt2 := dt1.Add(time.Second)
	expectedListing := domain.AppListing{
		Base: "abc.com/app",
		Versions: map[domain.Version]domain.AppListingVersion{
			domain.Version("1.2.3"): {
				Manifest:  "manifest-1.2.3.json",
				Package:   "manifest-1.2.3.tar.gz",
				Changelog: "changelog-1.2.3.txt",
				Icon:      "icon-1.2.3.gif"},
			domain.Version("1.5.0"): {
				Manifest:  "manifest-1.5.0.json",
				Package:   "manifest-1.5.0.tar.gz",
				Changelog: "changelog-1.5.0.txt",
				Icon:      "icon-1.5.0.gif"},
		},
	}
	lf.FetchDatetime = dt2
	lf.Listing = expectedListing
	lf.ListingDatetime = dt2
	lf.Etag = "etag2"
	lf.LatestVersion = domain.Version("1.5.0")

	err = appModel.UpdateURL(appID, url2, lf)
	if err != nil {
		t.Error(err)
	}
	listing, urlData, _ := appModel.GetAppUrlListing(appID)
	expected := domain.AppURLData{
		AppID:           appID,
		URL:             url2,
		Automatic:       false,
		Last:            dt2,
		LastResult:      "ok",
		NewURL:          "",
		NewUrlDatetime:  nulltypes.NewTime(time.Time{}, false),
		ListingDatetime: dt2,
		Etag:            "etag2",
		LatestVersion:   "1.5.0"}
	if !cmp.Equal(urlData, expected) {
		t.Error(cmp.Diff(urlData, expected))
	}
	if !cmp.Equal(listing, expectedListing) {
		t.Error(cmp.Diff(listing, expectedListing))
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
