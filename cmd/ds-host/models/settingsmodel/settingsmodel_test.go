package settingsmodel

import (
	"reflect"
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

func TestSetTSNet(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	settingsModel := &SettingsModel{
		DB: &domain.DB{
			Handle: h}}
	settingsModel.PrepareStatements()

	// get empty
	ret, err := settingsModel.GetTSNet()
	if err != nil {
		t.Error(err)
	}
	if ret.Connect {
		t.Error("expected connect to be false")
	}

	// set tsnet
	set := domain.TSNetCommon{
		ControlURL: "sume-url",
		Hostname:   "some-hostname",
		Connect:    true,
	}
	err = settingsModel.SetTSNet(set)
	if err != nil {
		t.Error(err)
	}
	ret, err = settingsModel.GetTSNet()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(set, ret) {
		t.Error("got different values for tsnet", ret, set)
	}

	// set Connect
	err = settingsModel.SetTSNetConnect(false)
	if err != nil {
		t.Error(err)
	}
	ret, err = settingsModel.GetTSNet()
	if err != nil {
		t.Error(err)
	}
	if ret.Connect {
		t.Error("expected connect to be false")
	}
}
