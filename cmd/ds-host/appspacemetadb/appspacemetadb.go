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
// Like users for example

// AppspaceMetaDB opens and tracks connections to appspace meta DBs
type AppspaceMetaDB struct {
	Config *domain.RuntimeConfig `checkinject:"required"`
	//Validator     domain.Validator
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceStatus interface {
		IsLockedClosed(domain.AppspaceID) bool
	} `checkinject:"required"`

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
		readySub: []chan struct{}{readyChan},
	}

	mdb.startConn(conn, appspaceID, true)

	<-readyChan

	if conn.connError != nil {
		mdb.getLogger("Create(), connError").Error(conn.connError)
		return conn.connError
	}

	// create tables  ->  need to branch out to different models for different API versions.
	err := conn.RunMigrationStep(dsAPIVersion, true)
	if err != nil {
		// nothing really to revert to.
		// Just means something is borked and we can't create an appspace right now?
		return err
	}

	mdb.connsMux.Lock()
	mdb.conns[appspaceID] = conn
	mdb.connsMux.Unlock()

	return nil
}

// getConn returns the existing conn for the appspace ID or creates one if necessary
func (mdb *AppspaceMetaDB) getConn(appspaceID domain.AppspaceID) (*DbConn, error) {
	locked := mdb.AppspaceStatus.IsLockedClosed(appspaceID)
	if locked {
		return nil, domain.ErrAppspaceLockedClosed
	}

	// lock, get from map, start if not there, wait if not ready, then unlock or somesuch
	var readyChan chan struct{}
	mdb.connsMux.Lock()
	conn, ok := mdb.conns[appspaceID]
	if ok {
		conn.statusMux.Lock()
		if conn.handle == nil && conn.connError == nil { // not ready yet
			readyChan = make(chan struct{})
			conn.readySub = append(conn.readySub, readyChan)
		}
		conn.statusMux.Unlock()
	} else {
		readyChan = make(chan struct{})
		mdb.conns[appspaceID] = &DbConn{
			readySub: []chan struct{}{readyChan},
		}
		conn = mdb.conns[appspaceID]

		go mdb.startConn(conn, appspaceID, false)
	}
	mdb.connsMux.Unlock()

	if readyChan != nil {
		<-readyChan
	}

	if conn.connError != nil {
		mdb.getLogger("getConn(), connError").Error(conn.connError)
	}

	return conn, conn.connError
}

// GetHandle returns the db handle for the appspace
func (mdb *AppspaceMetaDB) GetHandle(appspaceID domain.AppspaceID) (*sqlx.DB, error) {
	conn, err := mdb.getConn(appspaceID)
	if err != nil {
		mdb.getLogger("GetHandle(), getConn()").AppspaceID(appspaceID).Error(err)
		return nil, err
	}
	return conn.GetHandle(), nil
}

// CloseConn closes the db file and removes connection from conns
// The expectation is that this is called after the appspace has been confirmed stopped
func (mdb *AppspaceMetaDB) CloseConn(appspaceID domain.AppspaceID) error {

	mdb.connsMux.Lock()
	conn, ok := mdb.conns[appspaceID]
	if ok {
		delete(mdb.conns, appspaceID)
	}
	mdb.connsMux.Unlock()

	if ok {
		err := conn.handle.Close()
		if err != nil {
			return err
		}
		conn.handle = nil
	}
	return nil
}

func (mdb *AppspaceMetaDB) startConn(conn *DbConn, appspaceID domain.AppspaceID, create bool) { //maybe just pass location key instead of appspace id?
	appspace, err := mdb.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		setConnError(conn, err)
		return
	}

	appspacePath := filepath.Join(mdb.Config.Exec.AppspacesPath, appspace.LocationKey, "data")
	dbFile := filepath.Join(appspacePath, "appspace-meta.db")
	dsn := "file:" + dbFile + "?mode=rw"

	if create {
		_, err := os.Stat(dbFile)
		if !os.IsNotExist(err) {
			setConnError(conn, errors.New("Appspace DB file already exists: "+dbFile))
			return
		}
		err = os.MkdirAll(appspacePath, 0700)
		if err != nil {
			setConnError(conn, err)
			return
		}

		dsn += "c"
	}

	handle, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		setConnError(conn, err)
		return
	}

	err = handle.Ping()
	if err != nil {
		setConnError(conn, err)
		return
	}

	conn.statusMux.Lock()
	conn.handle = handle
	conn.statusMux.Unlock()

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

func setConnError(conn *DbConn, e error) {
	conn.statusMux.Lock()
	conn.connError = e
	conn.statusMux.Unlock()

	for _, ch := range conn.readySub {
		close(ch)
	}
}

// DbConn holds the db handle and relevant request data
type DbConn struct {
	statusMux sync.Mutex // not 100% sure what it's covering.
	handle    *sqlx.DB
	connError error
	readySub  []chan struct{}
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
		err = fmt.Errorf("appspace API version not handled: %v", toVersion)
	}

	return err
}

// getDb returns a db handle for an appspace meta db located at dataPath
// This should only be used to open DBs that are not part of an active appspace
// Meaning: use this to open appspace meta db of backup files, or imported files
func getDb(dataPath string) (*sqlx.DB, error) {
	dbFile := filepath.Join(dataPath, "appspace-meta.db")
	dsn := "file:" + dbFile + "?mode=rw"

	handle, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	err = handle.Ping()
	if err != nil {
		return nil, err
	}
	return handle, err
}
