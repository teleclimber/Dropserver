package appspacetsnetmodel

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
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

	appspaceTSNetModelEvents := testmocks.NewMockAppspaceTSNetModelEvents(mockCtrl)
	appspaceTSNetModelEvents.EXPECT().Send(domain.AppspaceTSNetModelEvent{
		Deleted: false,
		AppspaceTSNet: domain.AppspaceTSNet{
			AppspaceID: aID,
			ControlURL: controlURL,
			Hostname:   hostname,
			Connect:    connect},
	})

	model := &AppspaceTSNetModel{
		DB:                       &domain.DB{Handle: h},
		AppspaceTSNetModelEvents: appspaceTSNetModelEvents,
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

	appspaceTSNetModelEvents := testmocks.NewMockAppspaceTSNetModelEvents(mockCtrl)
	appspaceTSNetModelEvents.EXPECT().Send(gomock.Any())

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &AppspaceTSNetModel{
		DB:                       &domain.DB{Handle: h},
		AppspaceTSNetModelEvents: appspaceTSNetModelEvents}

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
	if !reflect.DeepEqual(tsnet, domain.AppspaceTSNet{AppspaceID: aID, ControlURL: controlURL, Hostname: "somenode", Connect: true}) {
		t.Error("Got wrong tsnet:", tsnet)
	}
}

func TestUpsert(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	aID := domain.AppspaceID(7)

	appspaceTSNetModelEvents := testmocks.NewMockAppspaceTSNetModelEvents(mockCtrl)
	appspaceTSNetModelEvents.EXPECT().Send(domain.AppspaceTSNetModelEvent{
		Deleted: false,
		AppspaceTSNet: domain.AppspaceTSNet{
			AppspaceID: aID,
			ControlURL: "https://www.example.com",
			Hostname:   "somenode",
			Connect:    true}})
	appspaceTSNetModelEvents.EXPECT().Send(domain.AppspaceTSNetModelEvent{
		Deleted: false,
		AppspaceTSNet: domain.AppspaceTSNet{
			AppspaceID: aID,
			ControlURL: "https://www.example2.com",
			Hostname:   "othernode",
			Connect:    false}})

	model := &AppspaceTSNetModel{
		DB:                       &domain.DB{Handle: h},
		AppspaceTSNetModelEvents: appspaceTSNetModelEvents}
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
	if !reflect.DeepEqual(tsnet, domain.AppspaceTSNet{AppspaceID: aID, ControlURL: "https://www.example2.com", Hostname: "othernode", Connect: false}) {
		t.Error("Got wrong tsnet:", tsnet)
	}
}

func TestSetConnect(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	appspaceTSNetModelEvents := testmocks.NewMockAppspaceTSNetModelEvents(mockCtrl)
	appspaceTSNetModelEvents.EXPECT().Send(gomock.Any()).Times(2)

	model := &AppspaceTSNetModel{
		DB:                       &domain.DB{Handle: h},
		AppspaceTSNetModelEvents: appspaceTSNetModelEvents}
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

	appspaceTSNetModelEvents := testmocks.NewMockAppspaceTSNetModelEvents(mockCtrl)
	appspaceTSNetModelEvents.EXPECT().Send(gomock.Any()).Times(3)

	model := &AppspaceTSNetModel{
		DB:                       &domain.DB{Handle: h},
		AppspaceTSNetModelEvents: appspaceTSNetModelEvents}
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

	appspaceTSNetModelEvents := testmocks.NewMockAppspaceTSNetModelEvents(mockCtrl)
	appspaceTSNetModelEvents.EXPECT().Send(gomock.Any())
	appspaceTSNetModelEvents.EXPECT().Send(domain.AppspaceTSNetModelEvent{
		Deleted:       true,
		AppspaceTSNet: domain.AppspaceTSNet{AppspaceID: aID7}})

	model := &AppspaceTSNetModel{
		DB:                       &domain.DB{Handle: h},
		AppspaceTSNetModelEvents: appspaceTSNetModelEvents}
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
