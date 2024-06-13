package appspacemetadb

import (
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestStartConn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	readyChan := make(chan struct{})
	conn := &dbConn{
		readySub: []chan struct{}{readyChan},
	}

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(domain.AppspaceID(13)).Return(&domain.Appspace{LocationKey: "abc"}, nil)

	mdb := &AppspaceMetaDB{
		Config:        cfg,
		AppspaceModel: appspaceModel,
	}

	mdb.startConn(conn, domain.AppspaceID(13), true)

	<-readyChan

	// test shutdown too
	err = mdb.CloseConn(domain.AppspaceID(13))
	if err != nil {
		t.Error(err)
	}

	// find out if file is actually closed?
}

// More tests needed.

func TestCreateAndGet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	appspaceID := domain.AppspaceID(13)

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{LocationKey: "abc"}, nil)

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().IsLockedClosed(appspaceID).Return(false)

	mdb := &AppspaceMetaDB{
		Config:         cfg,
		AppspaceModel:  appspaceModel,
		AppspaceStatus: appspaceStatus,
	}
	mdb.Init()

	err = mdb.Create(appspaceID, 0)
	if err != nil {
		t.Error(err)
	}

	// OK, now test Get

	h, err := mdb.GetHandle(appspaceID)
	if err != nil {
		t.Error(err)
	}

	var res struct {
		Value int
	}
	h.Get(&res, `SELECT value FROM info WHERE name ='ds-api-version'`)
	if err != nil {
		t.Error(err)
	}

	if res.Value != 0 {
		t.Error("expected value to be 0")
	}
}

func TestGetUnknownSchema(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}
	handle.SetMaxOpenConns(1)

	dbc := &dbConn{
		handle: handle,
	}

	s, err := dbc.getUnknownSchema()
	if err != nil {
		t.Error(err)
	}
	if s != -1 {
		t.Errorf("expected -1, got shcmea of %d", s)
	}

	err = dbc.migrateTo(0)
	if err != nil {
		t.Error(err)
	}

	s, err = dbc.getUnknownSchema()
	if err != nil {
		t.Error(err)
	}
	if s != 0 {
		t.Errorf("expected 0, got shcmea of %d", s)
	}

	// test additional schemas when we have them.

}

// Add MigrateTo Tests when we have more versions.
