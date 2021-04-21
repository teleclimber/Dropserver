package appspacedb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type connsKey struct {
	appspaceID domain.AppspaceID
	dbName     string
}

type connsVal struct {
	dbConn       *dbConn
	statusMux    sync.Mutex // not 100% sure what it's covering.
	connError    error
	readySub     []chan struct{}
	liveRequests int // I think this counts ongoing requests that are "claimed" towards this conn. Can't close unless it's zero
}

// ConnManager keeps tabs on open conns and can open more or close them
type ConnManager struct {
	appspacesPath string
	connsMux      sync.Mutex
	conns         map[connsKey]*connsVal
}

// Init makes the necessary maps
func (m *ConnManager) Init(appspacesPath string) {
	m.appspacesPath = appspacesPath
	m.conns = make(map[connsKey]*connsVal)
}

func (m *ConnManager) closeAppspaceDBs(appspaceID domain.AppspaceID) {
	var wg sync.WaitGroup
	m.connsMux.Lock()
	for key, val := range m.conns {
		if key.appspaceID == appspaceID {
			go func(v *connsVal) {
				wg.Add(1)
				v.dbConn.close()
				wg.Done()
			}(val)
			delete(m.conns, key)
		}
	}
	m.connsMux.Unlock()

	wg.Wait()
}

// createDB creates a new database
func (m *ConnManager) createDB(appspaceID domain.AppspaceID, locationKey string, dbName string) (*connsVal, error) {
	key := connsKey{
		appspaceID: appspaceID,
		dbName:     dbName}

	connVal, err := m.create(key)
	if err != nil {
		return nil, err
	}

	connVal.liveRequests++

	m.startConn(key, locationKey, connVal, true)

	connVal.liveRequests--

	if connVal.connError != nil {
		return nil, connVal.connError
	}
	return connVal, nil
}
func (m *ConnManager) create(key connsKey) (*connsVal, error) {
	m.connsMux.Lock()
	defer m.connsMux.Unlock()

	if _, ok := m.conns[key]; ok {
		return nil, fmt.Errorf("Can't create DB %s: already in cached keys", key.dbName)
	}

	m.conns[key] = &connsVal{
		readySub: []chan struct{}{},
	}

	return m.conns[key], nil
}

func (m *ConnManager) deleteDB(appspaceID domain.AppspaceID, locationKey string, dbName string) error {
	key := connsKey{
		appspaceID: appspaceID,
		dbName:     dbName}

	m.connsMux.Lock()
	conn, ok := m.conns[key]
	if ok {
		delete(m.conns, key)
	}
	defer m.connsMux.Unlock()

	if conn != nil && conn.dbConn != nil && conn.dbConn.handle != nil {
		conn.dbConn.handle.Close()
	}

	// then delete the file.
	err := os.Remove(m.getDBFullFilename(locationKey, dbName))
	if err != nil {
		return err
	}

	return nil
}

// getConn should open a conn and return the dbconn
// or return the existing dbconn, after waiting for it to be ready
// OR, if there was an error condition, return or mitigate....
func (m *ConnManager) getConn(appspaceID domain.AppspaceID, locationKey string, dbName string) *connsVal {
	key := connsKey{appspaceID, dbName}
	var readyChan chan struct{}
	m.connsMux.Lock()
	conn, ok := m.conns[key]
	if ok {
		conn.statusMux.Lock()
		conn.liveRequests++
		if conn.dbConn == nil && conn.connError == nil {
			readyChan = make(chan struct{})
			conn.readySub = append(conn.readySub, readyChan)
		}
		conn.statusMux.Unlock()
	} else {
		readyChan = make(chan struct{})
		m.conns[key] = &connsVal{
			readySub:     []chan struct{}{readyChan},
			liveRequests: 1,
		}
		conn = m.conns[key]

		go m.startConn(key, locationKey, conn, false)
	}
	m.connsMux.Unlock()

	if readyChan != nil {
		<-readyChan
	}

	return conn
}

func (m *ConnManager) startConn(key connsKey, locationKey string, c *connsVal, create bool) {
	dbConn, err := openConn(m.getDBFullFilename(locationKey, key.dbName), create)
	c.statusMux.Lock()
	if err != nil {
		c.connError = err
	} else {
		c.dbConn = dbConn
	}

	// then release all the channels that are waiting
	for _, ch := range c.readySub {
		close(ch)
	}
	c.statusMux.Unlock()
}

func (m *ConnManager) getDBFullFilename(locationKey string, dbName string) string {
	return filepath.Join(m.appspacesPath, locationKey, "data", "dbs", dbName+".db")
}

// should know something about itself? like appspace,path, ...
// should track its lru time
// should hold on to prepared statements

// Copy from host db

// this is generic db fopen, so can live outside vX
func openConn(filePath string, create bool) (*dbConn, error) {
	dsn := "file:" + filePath + "?mode=rw"

	if create {
		_, err := os.Stat(filePath)
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("Appspace DB File Exists: %s", filePath)
		}
		dsn += "c"

		// do we not need to create the dir at some point?
	}

	handle, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	err = handle.Ping()
	if err != nil {
		return nil, err
	}

	return &dbConn{
		handle:     handle,
		statements: make(map[string]*sql.Stmt),
	}, nil
}

type dbConn struct {
	handle     *sql.DB
	statements map[string]*sql.Stmt // does it need to be a pointer?
}

// placeholder so we can an idea what is needed.
func (dbc *dbConn) query(stmt *sql.Stmt, args []interface{}) ([]byte, error) {
	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scanned, err := scanRows(rows)
	if err != nil {
		return nil, err
	}

	results := map[string]interface{}{
		"rows": scanned,
	}

	json, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}

	return json, nil
}

func (dbc *dbConn) exec(stmt *sql.Stmt, args []interface{}) ([]byte, error) {
	r, err := stmt.Exec(args...)
	if err != nil {
		return nil, err
	}

	var results results

	results.LastInsertID, err = r.LastInsertId()
	if err != nil {
		return nil, err
	}
	results.RowsAffected, err = r.RowsAffected()
	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(results)
	if err != nil {
		return nil, err
	}

	return json, nil
}

// from https://stackoverflow.com/a/60386531/472819
// TODO: make this stream json instead of returning a big data structure.
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	count := len(columnTypes)
	finalRows := []map[string]interface{}{}

	for rows.Next() {
		scanArgs := make([]interface{}, count)

		for i, v := range columnTypes {
			switch v.DatabaseTypeName() {
			case "TEXT":
				scanArgs[i] = new(sql.NullString)
				break
			case "DATETIME":
				scanArgs[i] = new(sql.NullTime)
				break
			case "INTEGER":
				scanArgs[i] = new(sql.NullInt64)
				break
			case "REAL":
				scanArgs[i] = new(sql.NullFloat64)
				break
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		err := rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		masterData := map[string]interface{}{}

		for i, v := range columnTypes {

			if z, ok := (scanArgs[i]).(*sql.NullString); ok {
				if !z.Valid {
					return nil, fmt.Errorf("Appspace DB Scan Error: Invalid string scan, column name: %s", v.Name())
				}
				masterData[v.Name()] = z.String
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullInt64); ok {
				if !z.Valid {
					return nil, fmt.Errorf("Appspace DB Scan Error: Invalid Int64 scan, column name: %s", v.Name())
				}
				masterData[v.Name()] = z.Int64
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
				if !z.Valid {
					return nil, fmt.Errorf("Appspace DB Scan Error: Invalid float64 scan, column name: %s", v.Name())
				}
				masterData[v.Name()] = z.Float64
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullTime); ok {
				if !z.Valid {
					return nil, fmt.Errorf("Appspace DB Scan Error: Invalid time scan, column name: %s", v.Name())
				}
				masterData[v.Name()] = z.Time
				continue
			}

			return nil, fmt.Errorf("Appspace DB Scan Error: Failed to match scan arg type, column name: %s", v.Name())
		}

		finalRows = append(finalRows, masterData)
	}

	return finalRows, nil
}

func (dbc *dbConn) getStatement(q string) (*sql.Stmt, error) {
	s, ok := dbc.statements[q]
	if !ok {
		var err error
		s, err = dbc.handle.Prepare(q)
		if err != nil {
			return nil, err
		}
		dbc.statements[q] = s
	}
	return s, nil
}

func (dbc *dbConn) close() {
	if dbc.handle != nil {
		dbc.handle.Close()
	}
}
