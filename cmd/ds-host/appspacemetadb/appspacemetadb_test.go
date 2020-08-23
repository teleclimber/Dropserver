package appspacemetadb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestStartConn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	readyChan := make(chan struct{})
	conn := &DbConn{
		readySub:     []chan struct{}{readyChan},
		liveRequests: 1,
	}

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	AppspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	AppspaceModel.EXPECT().GetFromID(domain.AppspaceID(13)).Return(&domain.Appspace{LocationKey: "abc"}, nil)

	mdb := &AppspaceMetaDB{
		Config:        cfg,
		AppspaceModel: AppspaceModel,
	}

	mdb.startConn(conn, domain.AppspaceID(13), true)

	_ = <-readyChan

	// test shutdown too
}

// More tests needed.

func TestCreateAndGet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	appspaceID := domain.AppspaceID(13)

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	AppspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	AppspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{LocationKey: "abc"}, nil)

	mdb := &AppspaceMetaDB{
		Config:        cfg,
		AppspaceModel: AppspaceModel,
	}
	mdb.Init()

	err = mdb.Create(appspaceID, 0)
	if err != nil {
		t.Error(err)
	}

	// OK, now test Get

	dbConn, err := mdb.GetConn(appspaceID)
	if err != nil {
		t.Error(err)
	}
	h := dbConn.GetHandle()

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
