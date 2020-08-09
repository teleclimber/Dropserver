package appspacemetadb

import (
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestStartConn(t *testing.T) {
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
	cfg.Exec.AppspacesMetaPath = dir

	mdb := &AppspaceMetaDB{
		Config: cfg,
	}

	mdb.startConn(conn, domain.AppspaceID(13), true)

	_ = <-readyChan

	// test shutdown too
}

// More tests needed.

func TestCreateAndGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesMetaPath = dir
	mdb := &AppspaceMetaDB{
		Config: cfg,
	}
	mdb.Init()

	err = mdb.Create(domain.AppspaceID(13), 0)
	if err != nil {
		t.Error(err)
	}

	// OK, now test Get

	dbConn := mdb.GetConn(domain.AppspaceID(13))
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
