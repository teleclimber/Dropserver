package appspacemodel

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
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

	appspace, err := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace", "as123")
	if err != nil {
		t.Error(err)
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
	if appspace.LocationKey != "as123" {
		t.Error("appspace location key mismatch", appspace)
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

	_, err := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace", "as123")
	if err != nil {
		t.Error(err)
	}

	// There should now be one row so app id 1 should return something
	appspace, err := model.GetFromID(domain.AppspaceID(1))
	if err != nil {
		t.Error(err)
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

	_, err := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace", "as123")
	if err != nil {
		t.Error(err)
	}

	_, err = model.GetFromSubdomain("test-appspace")
	if err != nil {
		t.Error(err)
	}

	// test non-existent subdomain
	appspace, err := model.GetFromSubdomain("foo")
	if err != nil {
		t.Error(err)
	}
	if appspace != nil {
		t.Error("Should return nil trying to get non-existent subdomain")
	}
	// else if err.Code() != dserror.NoRowsInResultSet {
	// 	t.Error("wrong error for non-existent subdomain: ", err)
	// }
	// TODO: add sentinel error?
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
		userID    domain.UserID
		appID     domain.AppID
		version   domain.Version
		subDomain string
		location  string
	}{
		{7, 4, "0.0.1", "foo-subdomain", "as123"},
		{7, 5, "0.0.2", "2foo-subdomain", "as124"},
		{7, 6, "0.0.3", "3foo-subdomain", "as125"},
		{11, 6, "0.0.1", "bar-subdomain", "as126"},
	}

	for _, i := range ins {
		_, err := model.Create(i.userID, i.appID, i.version, i.subDomain, i.location)
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
}

func TestCreateDupeSubdomain(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	_, err := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace", "as123")
	if err != nil {
		t.Error(err)
	}

	_, err = model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace", "as789")
	if err == nil {
		t.Error("There should have been an error for duplicate subdomain")
	}
	// else if err.Code() != dserror.DomainNotUnique {
	// 	t.Error("Wrong error", err)
	// }
	// TODO add sentinel error?
}

//TODO: test dupe locationKey?

func TestGetForApp(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceModel{
		DB: db}

	model.PrepareStatements()

	ins := []struct {
		userID    domain.UserID
		appID     domain.AppID
		version   domain.Version
		subDomain string
		location  string
	}{
		{7, 4, "0.0.1", "foo-subdomain", "as123"},
		{7, 5, "0.0.2", "2foo-subdomain", "as124"},
		{7, 6, "0.0.3", "3foo-subdomain", "as125"},
		{11, 6, "0.0.1", "bar-subdomain", "as126"},
	}

	for _, i := range ins {
		_, err := model.Create(i.userID, i.appID, i.version, i.subDomain, i.location)
		if err != nil {
			t.Error(err)
		}
	}

	appSpaces, err := model.GetForApp(6)
	if err != nil {
		t.Error(err)
	}
	if len(appSpaces) != 2 {
		t.Error("expected 2 appspaces")
	}

	appSpaces, err = model.GetForApp(1)
	if err != nil {
		t.Error(err)
	}
	if len(appSpaces) != 0 {
		t.Error("expected ZERO appspaces")
	}
}

func TestPause(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	asPausedEvents := testmocks.NewMockAppspacePausedEvents(mockCtrl)
	asPausedEvents.EXPECT().Send(gomock.Any(), true)
	model := &AppspaceModel{
		AsPausedEvent: asPausedEvents,
		DB:            db}

	model.PrepareStatements()

	appspace, err := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace", "as123")
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

	appspace, err := model.Create(domain.UserID(1), domain.AppID(10), domain.Version("0.0.1"), "test-appspace", "as123")
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
