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

	err := settingsModel.Set(domain.Settings{RegistrationOpen: true})
	if err != nil {
		t.Fatal(err)
	}

	settings, err := settingsModel.Get()
	if err != nil {
		t.Fatal(err)
	}
	if !settings.RegistrationOpen {
		t.Fatal("registration open was supposed to be true")
	}

	err = settingsModel.Set(domain.Settings{RegistrationOpen: false})
	if err != nil {
		t.Fatal(err)
	}

	settings, err = settingsModel.Get()
	if err != nil {
		t.Fatal(err)
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

	err := settingsModel.SetRegistrationOpen(true)
	if err != nil {
		t.Fatal(err)
	}

	settings, err := settingsModel.Get()
	if err != nil {
		t.Fatal(err)
	}
	if !settings.RegistrationOpen {
		t.Fatal("registration open was supposed to be true")
	}

	err = settingsModel.SetRegistrationOpen(false)
	if err != nil {
		t.Fatal(err)
	}

	settings, err = settingsModel.Get()
	if err != nil {
		t.Fatal(err)
	}
	if settings.RegistrationOpen {
		t.Fatal("registration open was supposed to be false")
	}
}
