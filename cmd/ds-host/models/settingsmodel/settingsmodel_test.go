package settingsmodel

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

	settingsModel := &SettingsModel{
		DB: db}

	settingsModel.PrepareStatements()
}

func TestSet(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	settingsModel := &SettingsModel{
		DB: db}

	settingsModel.PrepareStatements()

	dsErr := settingsModel.Set(&domain.Settings{RegistrationOpen: true})
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	settings, dsErr := settingsModel.Get()
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if !settings.RegistrationOpen {
		t.Fatal("registration open was supposed to be true")
	}

	dsErr = settingsModel.Set(&domain.Settings{RegistrationOpen: false})
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	settings, dsErr = settingsModel.Get()
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if settings.RegistrationOpen {
		t.Fatal("registration open was supposed to be false")
	}
}

func TestSetRegistrationOpen(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	settingsModel := &SettingsModel{
		DB: db}

	settingsModel.PrepareStatements()

	dsErr := settingsModel.SetRegistrationOpen(true)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	settings, dsErr := settingsModel.Get()
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if !settings.RegistrationOpen {
		t.Fatal("registration open was supposed to be true")
	}

	dsErr = settingsModel.SetRegistrationOpen(false)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	settings, dsErr = settingsModel.Get()
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if settings.RegistrationOpen {
		t.Fatal("registration open was supposed to be false")
	}
}
