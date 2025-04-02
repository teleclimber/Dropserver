package appspacetsnetmodel

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &AppspaceTSNetModel{
		DB: db}

	model.PrepareStatements()
}

func TestCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	aID := domain.AppspaceID(7)
	controlURL := "https://www.example.com"
	hostname := "somenode"
	connect := true

	model := &AppspaceTSNetModel{
		DB: &domain.DB{Handle: h},
	}

	model.PrepareStatements()

	err := model.CreateOrUpdate(aID, controlURL, hostname, connect)
	if err != nil {
		t.Error(err)
	}
}

func TestGet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &AppspaceTSNetModel{
		DB: &domain.DB{Handle: h},
	}

	model.PrepareStatements()

	aID := domain.AppspaceID(7)
	controlURL := "https://www.example.com"

	_, err := model.Get(aID)
	if err != domain.ErrNoRowsInResultSet {
		t.Error("expected domain.ErrNoRowsInResultSet")
	}

	err = model.CreateOrUpdate(aID, controlURL, "somenode", true)
	if err != nil {
		t.Error(err)
	}

	tsnet, err := model.Get(aID)
	if err != nil {
		t.Error(err)
	}
	expected := domain.AppspaceTSNet{
		AppspaceID: aID,
		TSNetCommon: domain.TSNetCommon{
			ControlURL: controlURL,
			Hostname:   "somenode",
			Connect:    true,
		},
	}
	if !reflect.DeepEqual(tsnet, expected) {
		t.Error("Got wrong tsnet:", tsnet)
	}
}

func TestUpsert(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	aID := domain.AppspaceID(7)

	model := &AppspaceTSNetModel{
		DB: &domain.DB{Handle: h}}
	model.PrepareStatements()

	err := model.CreateOrUpdate(aID, "https://www.example.com", "somenode", true)
	if err != nil {
		t.Error(err)
	}

	err = model.CreateOrUpdate(aID, "https://www.example2.com", "othernode", false)
	if err != nil {
		t.Error(err)
	}

	tsnet, err := model.Get(aID)
	if err != nil {
		t.Error(err)
	}
	expected := domain.AppspaceTSNet{
		AppspaceID: aID,
		TSNetCommon: domain.TSNetCommon{
			ControlURL: "https://www.example2.com",
			Hostname:   "othernode",
			Connect:    false,
		}}
	if !reflect.DeepEqual(tsnet, expected) {
		t.Error("Got wrong tsnet:", tsnet)
	}
}

func TestSetConnect(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &AppspaceTSNetModel{
		DB: &domain.DB{Handle: h}}
	model.PrepareStatements()

	aID := domain.AppspaceID(7)

	err := model.SetConnect(aID, false)
	if err != nil {
		t.Error(err) // setting connect on non-existing row does not error.
	}

	err = model.CreateOrUpdate(aID, "", "somenode", true)
	if err != nil {
		t.Error(err)
	}

	err = model.SetConnect(aID, false)
	if err != nil {
		t.Error(err)
	}

	tsnet, err := model.Get(aID)
	if err != nil {
		t.Error(err)
	}
	if tsnet.Connect {
		t.Error("expected tsnet.Connect: false")
	}
}

func TestGetConnect(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &AppspaceTSNetModel{
		DB: &domain.DB{Handle: h}}
	model.PrepareStatements()

	tsnets, err := model.GetAllConnect()
	if err != nil {
		t.Error(err)
	}
	if len(tsnets) != 0 {
		t.Error("expected 2 tsnets.")
	}

	aID7 := domain.AppspaceID(7)
	aID11 := domain.AppspaceID(11)

	err = model.CreateOrUpdate(aID7, "", "somenode", true)
	if err != nil {
		t.Error(err)
	}
	err = model.CreateOrUpdate(aID11, "", "somenode", true)
	if err != nil {
		t.Error(err)
	}

	tsnets, err = model.GetAllConnect()
	if err != nil {
		t.Error(err)
	}
	if len(tsnets) != 2 {
		t.Error("expected 2 tsnets.")
	}

	err = model.SetConnect(aID11, false)
	if err != nil {
		t.Error(err)
	}

	tsnets, err = model.GetAllConnect()
	if err != nil {
		t.Error(err)
	}
	if len(tsnets) != 1 {
		t.Error("expected 1 tsnets.")
	}
	if tsnets[0].AppspaceID != aID7 {
		t.Error("geto the grong ts net?", tsnets[0])
	}
}

func TestDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	aID7 := domain.AppspaceID(7)

	model := &AppspaceTSNetModel{
		DB: &domain.DB{Handle: h}}
	model.PrepareStatements()

	err := model.CreateOrUpdate(aID7, "", "somenode", true)
	if err != nil {
		t.Error(err)
	}

	err = model.Delete(aID7)
	if err != nil {
		t.Error(err)
	}

	_, err = model.Get(aID7)
	if err != domain.ErrNoRowsInResultSet {
		t.Error("expecte no rows.")
	}
}
