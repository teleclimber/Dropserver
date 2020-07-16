package appspacemetadb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
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

// Init initializes data structures as needed
func (mdb *AppspaceMetaDB) Init() {
	mdb.conns = make(map[domain.AppspaceID]*DbConn)
}

// Create an apspace meta DB file for an appspace.
// Should specify schema version or DS API version, and branch accordingly.
func (mdb *AppspaceMetaDB) Create(appspaceID domain.AppspaceID, dsAPIVersion int) error {

	readyChan := make(chan struct{})
	conn := &DbConn{
		readySub:     []chan struct{}{readyChan},
		liveRequests: 1,
	}

	mdb.startConn(conn, appspaceID, true)

	_ = <-readyChan

	if conn.connError != nil {
		mdb.getLogger("Create(), connError").Error(conn.connError)
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

// Need a stop conn, or some way to automatically shut things off if idle?

func (mdb *AppspaceMetaDB) startConn(conn *DbConn, appspaceID domain.AppspaceID, create bool) {
	defer conn.statusMux.Unlock()

	metaPath := getAppspaceMetaPath(mdb.Config, appspaceID)
	dbFile := filepath.Join(metaPath, "appspace-meta.db")
	dsn := "file:" + dbFile + "?mode=rw"

	if create {
		_, err := os.Stat(dbFile)
		if !os.IsNotExist(err) {
			conn.statusMux.Lock()
			conn.connError = errors.New("Appspace DB file already exists: " + dbFile)
			return
		}
		err = os.MkdirAll(metaPath, 0700)
		if err != nil {
			conn.statusMux.Lock()
			conn.connError = err
		}

		dsn += "c"
	}

	handle, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		conn.statusMux.Lock()
		conn.connError = err
		return
	}

	conn.statusMux.Lock()
	err = handle.Ping()
	if err != nil {
		conn.connError = err
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

func (mdb *AppspaceMetaDB) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceMetaDB")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

// this should be extracted and moved to a thing that can be imported by anything that needs it
func getAppspaceMetaPath(cfg *domain.RuntimeConfig, appspaceID domain.AppspaceID) string {
	return filepath.Join(cfg.Exec.AppspacesMetaPath, fmt.Sprintf("appspace-%v", appspaceID))
}

// DbConn holds the db handle and relevant request data
type DbConn struct {
	statusMux    sync.Mutex // not 100% sure what it's covering.
	handle       *sqlx.DB   // maybe sqlx for this one?
	connError    error
	readySub     []chan struct{}
	liveRequests int // I think this counts ongoing requests that are "claimed" towards this conn. Can't close unless it's zero

	stmts map[string]*sqlx.Stmt
}

// GetHandle returns the DB handle for theis connection
func (dbc *DbConn) GetHandle() *sqlx.DB {
	return dbc.handle
}

// RunMigrationStep runs a single migration step
// This is exported but doesn't match the domain.DbConn Interface that is returned above.
func (dbc *DbConn) RunMigrationStep(toVersion int, up bool) error {
	var err error
	switch toVersion {
	case 0:
		v0h := &v0handle{handle: dbc.handle}
		if up {
			v0h.migrateUpToV0()
		} else {
			v0h.migrateDownToV0()
		}
		err = v0h.checkErr()
	default:
		err = fmt.Errorf("Appspace API version not handled: %v", toVersion)
	}

	return err
}
