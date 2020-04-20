package appspacemetadb

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// can create and destroy appspace meta db file
// gets and holds connection to appspace meta db
// Models stored in that db should be separate files/structs/interfaces
// Like routes for example

// AppspaceMetaDB opens and tracks connections to appspace meta DBs
type AppspaceMetaDB struct {
	Config    *domain.RuntimeConfig
	Validator domain.Validator

	connsMux sync.Mutex
	conns    map[domain.AppspaceID]*DbConn
}

// Create an apspace meta DB file for an appspace.
// Should specify schema version or DS API version, and branch accordingly.
func (mdb *AppspaceMetaDB) Create(appspaceID domain.AppspaceID, dsAPIVersion int) domain.Error {

	readyChan := make(chan struct{})
	conn := &DbConn{
		readySub:     []chan struct{}{readyChan},
		liveRequests: 1,
	}

	mdb.startConn(conn, appspaceID, false)

	_ = <-readyChan

	if conn.connError != nil {
		return conn.connError
	}

	// create tables  ->  need to branch out to different models for different API versions.
	dsErr := conn.RunMigrationStep(dsAPIVersion, true)
	if dsErr != nil {
		// nothing really to revert to.
		// Just means something is borked and we can't create an appspace right now?
		return dsErr
	}

	mdb.connsMux.Lock()
	mdb.conns[appspaceID] = conn
	mdb.connsMux.Unlock()

	return nil
}

// need a migrate function too. It's just a fact.

// GetConn returns the existing conn for the appspace ID or creates one if necessary
func (mdb *AppspaceMetaDB) GetConn(appspaceID domain.AppspaceID) domain.DbConn {
	// lock, get from map, start if not there, wait if not ready, then unlock or somesuch

	var readyChan chan struct{}
	mdb.connsMux.Lock()
	conn, ok := mdb.conns[appspaceID]
	if ok {
		conn.statusMux.Lock()
		conn.liveRequests++
		if conn.handle == nil && conn.connError == nil { // not ready yet
			readyChan = make(chan struct{})
			conn.readySub = append(conn.readySub, readyChan)
		}
		conn.statusMux.Unlock()
	} else {
		readyChan = make(chan struct{})
		mdb.conns[appspaceID] = &DbConn{
			readySub:     []chan struct{}{readyChan},
			liveRequests: 1,
		}
		conn = mdb.conns[appspaceID]

		go mdb.startConn(conn, appspaceID, false)
	}
	mdb.connsMux.Unlock()

	if readyChan != nil {
		_ = <-readyChan
	}

	return conn
}
func (mdb *AppspaceMetaDB) startConn(conn *DbConn, appspaceID domain.AppspaceID, create bool) {
	defer conn.statusMux.Unlock()

	dbFile := getAppspaceMetaDbFile(mdb.Config, appspaceID)

	dsn := "file:" + dbFile + "?mode=rw"

	if create {
		_, err := os.Stat(dbFile)
		if !os.IsNotExist(err) {
			conn.statusMux.Lock()
			conn.connError = dserror.New(dserror.AppspaceDBFileExists, "DB file: "+dbFile)
			return
		}
		dsn += "c"
	}

	handle, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		conn.statusMux.Lock()
		conn.connError = dserror.FromStandard(err)
		return
	}

	conn.statusMux.Lock()
	err = handle.Ping()
	if err != nil {
		conn.connError = dserror.FromStandard(err)
		return
	}

	conn.handle = handle
	conn.stmts = make(map[string]*sqlx.Stmt)

	// then release all the channels that are waiting
	// TODO: what if we return early?
	// maybe we need an error chan, and have a select?
	// ..or close all chanels in a defer statement.
	for _, ch := range conn.readySub {
		close(ch)
	}
}

// This should be brought out to a package that can be injected
func getAppspaceMetaDbPath(cfg *domain.RuntimeConfig, appspaceID domain.AppspaceID) string {
	return filepath.Join(cfg.Exec.AppspacesMetaPath, fmt.Sprintf("appspace-%v", appspaceID), "appspace-meta.db")
}
func getAppspaceMetaDbFile(cfg *domain.RuntimeConfig, appspaceID domain.AppspaceID) string {
	return filepath.Join(getAppspaceMetaDbPath(cfg, appspaceID), "appspace-meta.db")
}

// need a place to create the DB itself, and add the tables
// Q: who determines the schema? Is it the host? Has to be versioned in some way?

// DbConn holds the db handle and relevant request data
type DbConn struct {
	statusMux    sync.Mutex // not 100% sure what it's covering.
	handle       *sqlx.DB   // maybe sqlx for this one?
	connError    domain.Error
	readySub     []chan struct{}
	liveRequests int // I think this counts ongoing requests that are "claimed" towards this conn. Can't close unless it's zero

	stmts map[string]*sqlx.Stmt
}

// GetHandle returns the DB handle for theis connection
func (dbc *DbConn) GetHandle() *sqlx.DB {
	return dbc.handle
}

// RunMigrationStep runs a single migration step
func (dbc *DbConn) RunMigrationStep(toVersion int, up bool) domain.Error {
	var dsErr domain.Error
	switch toVersion {
	case 0:
		v0h := &v0handle{handle: dbc.handle}
		if up {
			v0h.migrateUpToV0()
		} else {
			v0h.migrateDownToV0()
		}
		dsErr = v0h.checkErr()
	default:
		dsErr = dserror.New(dserror.AppspaceAPIVersionNotFound)
	}

	return dsErr
}
