package appspacemodel

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
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

func TestGetAll(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	a, err := model.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(a) != 0 {
		t.Error("expected empty array")
	}

	inAppspace := domain.Appspace{
		OwnerID:     domain.UserID(7),
		AppID:       domain.AppID(11),
		AppVersion:  domain.Version("0.0.1"),
		DomainName:  "test-appSPACE",
		LocationKey: "as123",
	}

	_, err = model.Create(inAppspace)
	if err != nil {
		t.Error(err)
	}

	a, err = model.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(a) != 1 {
		t.Error("expected array of length 1")
	}
	if a[0].AppID != inAppspace.AppID {
		t.Error("didn't get the data expected")
	}
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
	_, err := model.GetFromID(10)
	if err == nil {
		t.Error("expected an error")
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

	inAppspace := domain.Appspace{
		OwnerID:     domain.UserID(7),
		AppID:       domain.AppID(11),
		AppVersion:  domain.Version("0.0.1"),
		DomainName:  "test-appspace",
		LocationKey: "as123",
	}

	storedAppspace, err := model.Create(inAppspace)
	if err != nil {
		t.Error(err)
	}

	inAppspace.AppspaceID = storedAppspace.AppspaceID
	inAppspace.Created = storedAppspace.Created

	if inAppspace != *storedAppspace {
		t.Error("input appspace different from stored appspace")
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

	inAppspace := domain.Appspace{
		OwnerID:     domain.UserID(7),
		AppID:       domain.AppID(11),
		AppVersion:  domain.Version("0.0.1"),
		DomainName:  "test-appspace",
		LocationKey: "as123",
	}

	_, err := model.Create(inAppspace)
	if err != nil {
		t.Error(err)
	}

	// There should now be one row so app id 1 should return something
	appspace, err := model.GetFromID(domain.AppspaceID(1))
	if err != nil {
		t.Error(err)
	}

	if appspace.AppID != inAppspace.AppID {
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

	inAppspace := domain.Appspace{
		OwnerID:     domain.UserID(7),
		AppID:       domain.AppID(11),
		AppVersion:  domain.Version("0.0.1"),
		DomainName:  "test-appSPACE",
		LocationKey: "as123",
	}

	_, err := model.Create(inAppspace)
	if err != nil {
		t.Error(err)
	}

	_, err = model.GetFromDomain("TEST-appspace")
	if err != nil {
		t.Error(err)
	}

	// test non-existent subdomain
	appspace, err := model.GetFromDomain("foo")
	if err != nil {
		t.Error(err)
	}
	if appspace != nil {
		t.Error("Should return nil trying to get non-existent subdomain")
	}
	// TODO: add sentinel error?
}

func TestGetters(t *testing.T) {
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
		domain   string
		location string
	}{
		{7, 4, "0.0.1", "foo-subdomain", "as123"},
		{7, 5, "0.0.2", "2foo-subdomain", "as124"},
		{7, 6, "0.0.3", "3foo-subdomain", "as125"},
		{11, 6, "0.0.1", "bar-subdomain", "as126"},
	}

	for _, i := range ins {
		in := domain.Appspace{
			OwnerID:     i.userID,
			AppID:       i.appID,
			AppVersion:  i.version,
			DomainName:  i.domain,
			LocationKey: i.location,
		}
		_, err := model.Create(in)
		if err != nil {
			t.Error(err)
		}
	}

	appSpaces, err := model.GetForOwner(7)
	if err != nil {
		t.Error(err)
	}
	if len(appSpaces) != 3 {
		t.Error("expected 3 appspaces")
	}

	appSpaces, err = model.GetForOwner(1)
	if err != nil {
		t.Error(err)
	}
	if len(appSpaces) != 0 {
		t.Error("expected ZERO appspaces")
	}

	appSpaces, err = model.GetForApp(6)
	if err != nil {
		t.Error(err)
	}
	if len(appSpaces) != 2 {
		t.Error("expected 2 appspaces")
	}

	appSpaces, err = model.GetForAppVersion(6, "0.0.3")
	if err != nil {
		t.Error(err)
	}
	if len(appSpaces) != 1 {
		t.Error("expected 1 appspaces")
	}

	domains, err := model.GetAllDomains()
	if err != nil {
		t.Error(err)
	}
	if len(domains) != 4 {
		t.Error("expected 4 domains")
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

	inAppspace := domain.Appspace{
		OwnerID:     domain.UserID(7),
		AppID:       domain.AppID(11),
		AppVersion:  domain.Version("0.0.1"),
		DomainName:  "test-appspace",
		LocationKey: "as123",
	}

	_, err := model.Create(inAppspace)
	if err != nil {
		t.Error(err)
	}

	inAppspace.LocationKey = "as789"

	_, err = model.Create(inAppspace)
	if err == nil {
		t.Error("There should have been an error for duplicate subdomain")
	}
	// TODO add sentinel error?
}

//TODO: test dupe locationKey?

func TestPause(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	inAppspace := domain.Appspace{
		OwnerID:     domain.UserID(7),
		AppID:       domain.AppID(11),
		AppVersion:  domain.Version("0.0.1"),
		DomainName:  "test-appspace",
		LocationKey: "as123",
	}

	appspace, err := model.Create(inAppspace)
	if err != nil {
		t.Error(err)
	}

	err = model.Pause(appspace.AppspaceID, true)
	if err != nil {
		t.Error(err)
	}

	appspace, err = model.GetFromID(appspace.AppspaceID)
	if err != nil {
		t.Error(err)
	}
	if !appspace.Paused {
		t.Error("appspace should be paused")
	}
}

func TestSetVersion(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	inAppspace := domain.Appspace{
		OwnerID:     domain.UserID(7),
		AppID:       domain.AppID(11),
		AppVersion:  domain.Version("0.0.1"),
		DomainName:  "test-appspace",
		LocationKey: "as123",
	}

	appspace, err := model.Create(inAppspace)
	if err != nil {
		t.Error(err)
	}

	err = model.SetVersion(appspace.AppspaceID, domain.Version("0.0.2"))
	if err != nil {
		t.Error(err)
	}

	appspace, err = model.GetFromID(appspace.AppspaceID)
	if err != nil {
		t.Error(err)
	}
	if appspace.AppVersion != domain.Version("0.0.2") {
		t.Error("appspace version incorrect")
	}
}

func TestDelete(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	err := model.Delete(domain.AppspaceID(9999))
	if err == nil {
		t.Error("expected error")
	}
	if err != domain.ErrNoRowsAffected {
		t.Error(err)
	}

	inAppspace := domain.Appspace{
		OwnerID:     domain.UserID(7),
		AppID:       domain.AppID(11),
		AppVersion:  domain.Version("0.0.1"),
		DomainName:  "test-appspace",
		LocationKey: "as123",
	}

	appspace, err := model.Create(inAppspace)
	if err != nil {
		t.Error(err)
	}

	err = model.Delete(appspace.AppspaceID)
	if err != nil {
		t.Error(err)
	}

	_, err = model.GetFromID(appspace.AppspaceID)
	if err != domain.ErrNoRowsInResultSet {
		t.Error("expected no rows in result set error")
	}
}

// TODO test getalldomains
