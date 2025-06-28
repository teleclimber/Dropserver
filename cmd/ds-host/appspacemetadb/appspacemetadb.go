package appspacemetadb

import (
	"database/sql"
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
	conns    map[domain.AppspaceID]*dbConn
}

// Init initializes data structures as needed
func (mdb *AppspaceMetaDB) Init() {
	mdb.conns = make(map[domain.AppspaceID]*dbConn)
}

// Create an apspace meta DB file for an appspace.
func (mdb *AppspaceMetaDB) Create(appspaceID domain.AppspaceID) error {

	readyChan := make(chan struct{})
	conn := &dbConn{
		readySub:   []chan struct{}{readyChan},
		appspaceID: appspaceID,
	}

	mdb.startConn(conn, appspaceID, true)

	<-readyChan

	if conn.connError != nil {
		mdb.getLogger("Create(), connError").Error(conn.connError)
		return conn.connError
	}

	err := conn.migrateTo(curSchema)
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

// GetCurSchema returns the MetaDB schema for thes version of the code
// Just right now it's unused.
func (mdb *AppspaceMetaDB) GetCurSchema() int {
	return curSchema
}

// GetSchema returns the schema of the appspace meta DB for that appspaceID
func (mdb *AppspaceMetaDB) GetSchema(appspaceID domain.AppspaceID) (int, error) {
	conn, err := mdb.getConn(appspaceID)
	if err != nil {
		return 0, err
	}

	schema, err := conn.getUnknownSchema()

	return schema, err
}

// Migrate the appspace's meta DB to the current schema
// It no-ops without an error if migration is not necessary
func (mdb *AppspaceMetaDB) Migrate(appspaceID domain.AppspaceID) error {
	conn, err := mdb.getConn(appspaceID)
	if err != nil {
		return err
	}
	return conn.migrateTo(curSchema)
}

// OfflineMigrate the appspace's meta DB to the curent shcema
// But it does not check that the appspace is in a state that allows this
func (mdb *AppspaceMetaDB) OfflineMigrate(appspaceID domain.AppspaceID) error {
	conn, err := mdb.getConnNoLockCheck(appspaceID)
	if err != nil {
		return err
	}
	return conn.migrateTo(curSchema)
}

// getConn returns the existing conn for the appspace ID or creates one if necessary
func (mdb *AppspaceMetaDB) getConn(appspaceID domain.AppspaceID) (*dbConn, error) {
	locked := mdb.AppspaceStatus.IsLockedClosed(appspaceID)
	if locked {
		return nil, domain.ErrAppspaceLockedClosed
	}
	return mdb.getConnNoLockCheck(appspaceID)
}
func (mdb *AppspaceMetaDB) getConnNoLockCheck(appspaceID domain.AppspaceID) (*dbConn, error) {
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
		mdb.conns[appspaceID] = &dbConn{
			readySub:   []chan struct{}{readyChan},
			appspaceID: appspaceID,
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
	return conn.getHandle(), nil
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

func (mdb *AppspaceMetaDB) startConn(conn *dbConn, appspaceID domain.AppspaceID, create bool) { //maybe just pass location key instead of appspace id?
	appspace, err := mdb.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		setConnError(conn, err)
		return
	}

	if appspace.LocationKey == "" {
		panic("appspace location key empty")
	}

	appspacePath := filepath.Join(mdb.Config.Exec.AppspacesPath, appspace.LocationKey, "data") //TODO asl2p
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

func setConnError(conn *dbConn, e error) {
	conn.statusMux.Lock()
	conn.connError = e
	conn.statusMux.Unlock()

	for _, ch := range conn.readySub {
		close(ch)
	}
}

// dbConn holds the db handle and relevant request data
type dbConn struct {
	statusMux  sync.Mutex // not 100% sure what it's covering.
	handle     *sqlx.DB
	connError  error
	readySub   []chan struct{}
	appspaceID domain.AppspaceID
}

// getHandle returns the DB handle for theis connection
func (dbc *dbConn) getHandle() *sqlx.DB {
	return dbc.handle
}

// getUnknownSchema of an Appspace Meta DB
// Returns a schema of -1 if DB is freshly created
func (dbc *dbConn) getUnknownSchema() (int, error) {
	// schema may be in info table.
	// or it might be in schema thing of sqlite
	// or it may not have a schema at all.

	// if table info exists, then look up schema
	var dbSchema int
	var numInfo int
	err := dbc.handle.Get(&numInfo, `SELECT count(*) AS num FROM sqlite_schema WHERE type='table' AND name='info'`)
	if err != nil && err != sql.ErrNoRows { // how does No Rows make sense here?
		//actual sql error,
		return 0, fmt.Errorf("error in getUnknownSchema: error getting info tables: %w", err)
	}
	if numInfo == 1 {
		err = dbc.handle.Get(&dbSchema, `SELECT value FROM info WHERE name = ?`, "ds-api-version")
		if err != nil && err != sql.ErrNoRows {
			//actual sql error,
			return 0, fmt.Errorf("error in getUnknownSchema: error getting value of ds-api-version: %w", err)
		}
		if err != sql.ErrNoRows {
			// schema value is probably valid.
			// Also it should be 0 anyways.
			return dbSchema, nil
		}
	}

	// If there are no tables at all, freshly created DB, return -1
	var numTable int
	err = dbc.handle.Get(&numTable, `SELECT count(*) AS num FROM sqlite_schema WHERE type='table'`)
	if err != nil && err != sql.ErrNoRows {
		//actual sql error,
		return 0, fmt.Errorf("error in getUnknownSchema: error getting total table count: %w", err)
	}
	if numTable == 0 {
		// freshfly created DB. return -1
		return -1, nil
	}

	err = dbc.handle.Get(&dbSchema, `PRAGMA user_version`)
	if err != nil { // Pragma user_version does not return ErrNoRows
		return dbSchema, fmt.Errorf("error in getUnknownSchema: error reading PRAGMA user_version: %w", err)
	}
	if dbSchema > 0 {
		return dbSchema, nil
	}

	// maybe return sentinel error.
	return 0, errors.New("unable to determine schema of appspace Meta DB")
}

// migrateTo runs migration steps necesary to take db to desired version
func (dbc *dbConn) migrateTo(to int) error {
	if to < 0 {
		return fmt.Errorf("invalid to value for migrateTo: %d", to)
	}
	if to > curSchema {
		return fmt.Errorf("invalid to value for migrateTo: %d, highest is %d", to, curSchema)
	}
	cur, err := dbc.getUnknownSchema()
	if err != nil {
		return err
	}
	if cur > curSchema {
		// This may need to be a sentinel error so that we can propgate a good message to the user
		return fmt.Errorf("unknown appspace Meta DB schema: %d, the highest known is %d", cur, curSchema)
	}
	if cur == to {
		return nil
	}

	dbc.getLogger("migrateTo").AppspaceID(dbc.appspaceID).Log(fmt.Sprintf("About to migrate meta DB from %v to %v", cur, to))

	d := &dbExec{handle: dbc.handle}
	for i := cur + 1; i <= to; i++ {
		dbc.getLogger("migrateTo").AppspaceID(dbc.appspaceID).Log(fmt.Sprintf("Migrating meta DB up to %v", i))
		upMigrations[i](d)
	}
	for i := cur - 1; i >= to; i-- {
		dbc.getLogger("migrateTo").AppspaceID(dbc.appspaceID).Log(fmt.Sprintf("Migrating meta DB down from %v", i))
		downMigrations[i](d)
	}

	return d.checkErr()

	// TODO maybe add a step that reads the schema? Just to be sure?
}

func (dbc *dbConn) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceMetaDB dbConn")
	if note != "" {
		r.AddNote(note)
	}
	return r
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
