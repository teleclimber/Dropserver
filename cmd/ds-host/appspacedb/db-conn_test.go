package appspacedb

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestScanRows(t *testing.T) {
	handle, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	_, err = handle.Exec(`CREATE TABLE "apps" (
		"owner_id" INTEGER,
		"app_id" INTEGER PRIMARY KEY ASC,
		"name" TEXT,
		"created" DATETIME,
		"usage" REAL
	)`)
	if err != nil {
		t.Error(err)
	}

	_, err = handle.Exec(`INSERT INTO apps VALUES (?, ?, ?, datetime("now"), ?)`, sql.Named("", 1), 7, "some app", 999.9)
	if err != nil {
		t.Error(err)
	}

	_, err = handle.Exec(`INSERT INTO apps VALUES (?, ?, ?, strftime('%s','now'), ?)`, 2, "11", "some app", "77.77")
	if err != nil {
		t.Error(err)
	}

	/// testing tests...
	rowz, err := handle.Query(`SELECT created FROM apps ORDER BY owner_id`)
	if err != nil {
		t.Error(err)
	}
	var created time.Time
	var zero time.Time

	for rowz.Next() {
		err = rowz.Scan(&created)
		if err != nil {
			t.Error(err)
		}
		if created == zero {
			t.Errorf("Got zero value for time")
		}
	}
	rowz.Close()
	////////////////////////////

	rows, err := handle.Query(`SELECT * FROM apps ORDER BY owner_id`)
	if err != nil {
		t.Error(err)
	}

	scanned, err := scanRows(rows)
	if err != nil {
		t.Error(err)
	}
	rows.Close()

	if len(scanned) != 2 {
		t.Errorf("Scanned rows is only %v long", len(scanned))
	}

	int64Value(t, pluck(t, scanned, 0, "owner_id"), 1)
	stringValue(t, pluck(t, scanned, 0, "name"), "some app")
	int64Value(t, pluck(t, scanned, 1, "app_id"), 11)

	// For now just make sure it's a date type.
	dateType(t, pluck(t, scanned, 0, "created"))
	dateType(t, pluck(t, scanned, 1, "created"))

	float64Value(t, pluck(t, scanned, 0, "usage"), 999.9)
	float64Value(t, pluck(t, scanned, 1, "usage"), 77.77)

}

func TestGetStatement(t *testing.T) {

}

func TestExec(t *testing.T) {
	handle, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	dbc := &dbConn{
		handle:     handle,
		statements: make(map[string]*sql.Stmt),
	}

	stmt, err := dbc.getStatement(`CREATE TABLE "apps" (
		"owner_id" INTEGER,
		"app_id" INTEGER PRIMARY KEY ASC,
		"name" TEXT,
		"created" DATETIME,
		"usage" REAL
	)`)
	if err != nil {
		t.Error(err)
	}

	args := make([]interface{}, 0)

	jsonBytes, err := dbc.exec(stmt, args)
	if err != nil {
		t.Error(err)
	}

	var createResults results
	err = json.Unmarshal(jsonBytes, &createResults)
	if err != nil {
		t.Error(err)
	}

	stmt, err = dbc.getStatement(`INSERT INTO apps VALUES (?, ?, ?, datetime("now"), ?)`)
	if err != nil {
		t.Error(err)
	}

	// TODO: test row insert
	args = []interface{}{
		sql.Named("", float64(1)),
		sql.Named("", float64(7)),
		sql.Named("", "some app"),
		sql.Named("", float64(999.9))}
	jsonBytes, err = dbc.exec(stmt, args)
	if err != nil {
		t.Error(err)
	}

	var insResults results
	err = json.Unmarshal(jsonBytes, &insResults)
	if err != nil {
		t.Error(err)
	}

	if insResults.RowsAffected != 1 {
		t.Errorf("expected rows affected to be 1, got %v", insResults.RowsAffected)
	}
}

func TestQuery(t *testing.T) {
	handle, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	_, err = handle.Exec(`CREATE TABLE "apps" (
		"owner_id" INTEGER,
		"app_id" INTEGER PRIMARY KEY ASC,
		"name" TEXT,
		"created" DATETIME,
		"usage" REAL
	)`)
	if err != nil {
		t.Error(err)
	}

	_, err = handle.Exec(`INSERT INTO apps VALUES (?, ?, ?, datetime("now"), ?)`, 1, 7, "some app", 999.9)
	if err != nil {
		t.Error(err)
	}

	_, err = handle.Exec(`INSERT INTO apps VALUES (?, ?, ?, strftime('%s','now'), ?)`, 2, "11", "some app", "77.77")
	if err != nil {
		t.Error(err)
	}

	dbc := &dbConn{
		handle:     handle,
		statements: make(map[string]*sql.Stmt),
	}

	stmt, err := dbc.getStatement("SELECT * FROM apps WHERE app_id = ? ORDER BY owner_id ")
	if err != nil {
		t.Error(err)
	}

	jsonBytes, err := dbc.query(stmt, []interface{}{sql.Named("", float64(11))})
	if err != nil {
		t.Error(err)
	}

	jsonStr := string(jsonBytes[:])

	if !strings.Contains(jsonStr, `"usage":77.77`) {
		t.Errorf("json should contain substring. json: %v", jsonStr)
	}
}

func TestNamedParams(t *testing.T) {
	handle, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	dbc := &dbConn{
		handle:     handle,
		statements: make(map[string]*sql.Stmt),
	}

	stmt, err := dbc.getStatement(`CREATE TABLE "apps" (
		"owner_id" INTEGER,
		"app_id" INTEGER PRIMARY KEY ASC,
		"name" TEXT,
		"created" DATETIME,
		"usage" REAL
	)`)
	if err != nil {
		t.Error(err)
	}

	_, err = dbc.exec(stmt, []interface{}{})
	if err != nil {
		t.Error(err)
	}

	insStmt, err := dbc.getStatement(`INSERT INTO apps VALUES (:owner_id, :app_id, :name, datetime("now"), :usage)`)
	if err != nil {
		t.Error(err)
	}

	_, err = dbc.exec(insStmt, []interface{}{
		sql.Named("app_id", 7),
		sql.Named("usage", 999.9),
		sql.Named("name", "some app"),
		sql.Named("owner_id", 1)})
	if err != nil {
		t.Error(err)
	}

	_, err = dbc.exec(insStmt, []interface{}{
		sql.Named("app_id", 11),
		sql.Named("usage", 77.77),
		sql.Named("name", "another app"),
		sql.Named("owner_id", 2)})
	if err != nil {
		t.Error(err)
	}

	stmt, err = dbc.getStatement("SELECT * FROM apps WHERE app_id = ? ORDER BY owner_id ")
	if err != nil {
		t.Error(err)
	}

	jsonBytes, err := dbc.query(stmt, []interface{}{sql.Named("", float64(11))})
	if err != nil {
		t.Error(err)
	}

	jsonStr := string(jsonBytes[:])

	if !strings.Contains(jsonStr, `"usage":77.77`) {
		t.Errorf("json should contain substring. json: %v", jsonStr)
	}

	// test bad param lists
	_, err = dbc.exec(insStmt, []interface{}{
		sql.Named("app_id", 7),
		sql.Named("usage", 999.9),
		//sql.Named("name", "some app"),
		sql.Named("owner_id", 1)})
	if err == nil {
		t.Error("expected an error for missing a parameter")
	}

	_, err = dbc.exec(insStmt, []interface{}{
		sql.Named("app_id", 7),
		sql.Named("usage", 999.9),
		sql.Named("nameZZZ", "some app"),
		sql.Named("owner_id", 1)})
	if err == nil {
		t.Error("expected an error for non-existent nameZZZ parameter")
	}

	_, err = dbc.exec(insStmt, []interface{}{
		sql.Named("app_id", "foo"), // string as number
		sql.Named("usage", 999.9),
		sql.Named("name", "some app"),
		sql.Named("owner_id", 1)})
	if err == nil {
		t.Error("expected an error for wrong type parameter")
	}
}

func TestOpenNoDB(t *testing.T) {
	// create temp dir for the DB
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	_, err = openConn(filepath.Join(dir, "test.db"), false)
	if err == nil { // actually it should error!
		t.Error("Should have failed to open non existent DB")
	}
}

func TestCreate(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	dsc, err := openConn(filepath.Join(dir, "test.db"), true)
	if err != nil {
		t.Error(err)
	}
	defer dsc.close()
}

func TestCreateExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "test.db")
	emptyFile, err := os.Create(file)
	if err != nil {
		t.Error(err)
	}
	emptyFile.Close()

	_, err = openConn(file, true)
	if err == nil {
		t.Error("should have errored trying to create pre-existing file")
	}
	// else if err.Code() != error.AppspaceDBFileExists { //TODO create sentinel error?
	// 	t.Error("wrong error")
	// }
}

func TestCreateDB(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	loc := "abc-loc"
	dbName := "test-db"

	err = os.MkdirAll(filepath.Join(dir, loc), 0700)
	if err != nil {
		t.Error(err)
	}

	m := &ConnManager{}
	m.Init(dir)

	_, err = m.createDB(domain.AppspaceID(7), loc, dbName)
	if err != nil {
		t.Error(err)
	}

	// then test delete
	err = m.deleteDB(domain.AppspaceID(7), loc, dbName)
	if err != nil {
		t.Error(err)
	}

	filePath := filepath.Join(m.appspacesPath, loc, dbName+".db")
	_, err = os.Stat(filePath)
	if err == nil || !os.IsNotExist(err) {
		t.Error("Expect file to not exist")
	}
}

func TestStartConn(t *testing.T) {
	loc := "abc"
	dir := makeAppspaceDB(t, loc)
	defer os.RemoveAll(dir)

	appspaceID := domain.AppspaceID(13)

	m := &ConnManager{}
	m.Init(dir)

	key := connsKey{
		appspaceID: appspaceID,
		dbName:     "test",
	}

	readyChan := make(chan struct{})
	c := &connsVal{
		readySub: []chan struct{}{readyChan},
	}

	go m.startConn(key, loc, c, false)

	_ = <-readyChan

	if c.connError != nil {
		t.Error(c.connError)
	}
	if c.dbConn == nil {
		t.Error("there should be a dbConn")
	}

	c.dbConn.close()
}

// also do a startConn that triggers an error

func TestGetConn(t *testing.T) {
	loc := "abc"
	dir := makeAppspaceDB(t, loc)
	defer os.RemoveAll(dir)

	appspaceID := domain.AppspaceID(13)

	m := &ConnManager{}
	m.Init(dir)

	c := m.getConn(appspaceID, loc, "test")

	if c.connError != nil {
		t.Error(c.connError)
	}
	if c.dbConn == nil {
		t.Error("there should be a dbConn")
	}

	c.dbConn.close()
}

// TODO: do a TestGetConnError

// Test that a second request for DB that comes in before the DB is ready
// works OK by receivng the conn when it's ready
func TestGetConnSecondOverlap(t *testing.T) {
	loc := "abc"
	dir := makeAppspaceDB(t, loc)

	defer os.RemoveAll(dir)

	appspaceID := domain.AppspaceID(13)

	m := &ConnManager{}
	m.Init(dir)

	key := connsKey{
		appspaceID: appspaceID,
		dbName:     "test",
	}

	readyChan := make(chan struct{})
	c1 := &connsVal{
		liveRequests: 10,
		readySub:     []chan struct{}{readyChan},
	}
	m.connsMux.Lock()
	m.conns[key] = c1
	m.connsMux.Unlock()

	go func() {
		time.Sleep(100 * time.Millisecond)
		c1.statusMux.Lock()
		for _, ch := range c1.readySub {
			close(ch)
		}
		c1.statusMux.Unlock()
	}()

	c := m.getConn(appspaceID, loc, "test")

	// test that live requests was incremented to 11,
	// which indicates both attempts to get conn return the same conn
	if c.liveRequests != 11 {
		t.Error("expected live requests to be 11")
	}

}

func TestGetConnSecond(t *testing.T) {
	loc := "abc"
	dir := makeAppspaceDB(t, loc)
	defer os.RemoveAll(dir)

	appspaceID := domain.AppspaceID(13)

	m := &ConnManager{}
	m.Init(dir)

	c1 := m.getConn(appspaceID, loc, "test")

	c2 := m.getConn(appspaceID, loc, "test")

	if c1 != c2 {
		t.Error("should be the same conn")
	}
	if c1.connError != nil {
		t.Error("should not be an error")
	}
	if c1.dbConn == nil {
		t.Error("there should be a db conn")
	}
	if c2.liveRequests != 2 {
		t.Error("expected live requests to be 2")
	}

}

// TODO: need a test open when there is a DB

//////////////////////////////////////////////////////
//// helpers to check returned values:

func pluck(t *testing.T, rows []map[string]interface{}, rowI int, key string) interface{} {
	i, ok := rows[rowI][key]
	if !ok {
		t.Error("no such key " + key)
	}

	return i
}

func int64Value(t *testing.T, val interface{}, cmp int64) {
	s, ok := val.(int64)
	if !ok {
		t.Error("not an int64")
		return
	}
	if s != cmp {
		t.Errorf("wrong value: %v instead of %v", s, cmp)
	}
}
func float64Value(t *testing.T, val interface{}, cmp float64) {
	s, ok := val.(float64)
	if !ok {
		t.Error("not an float64")
		return
	}
	if s != cmp {
		t.Errorf("wrong value: %v instead of %v", s, cmp)
	}
}
func stringValue(t *testing.T, val interface{}, cmp string) {
	s, ok := val.(string)
	if !ok {
		t.Error("not an string")
		return
	}
	if s != cmp {
		t.Errorf("wrong value: %v instead of %v", s, cmp)
	}
}
func dateType(t *testing.T, val interface{}) {
	s, ok := val.(time.Time)
	if !ok {
		t.Error("not an time.Time")
		return
	}
	var zero time.Time
	if s == zero {
		t.Errorf("Got zero value for time")
	}
}

func makeAppspaceDB(t *testing.T, locationKey string) string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}

	asDir := filepath.Join(dir, locationKey)

	err = os.Mkdir(asDir, 0700)
	if err != nil {
		t.Error(err)
	}

	dsn := filepath.Join(asDir, "test.db?mode=rwc")

	handle, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Error(err)
	}
	err = handle.Ping()
	if err != nil {
		t.Error(err)
	}
	err = handle.Close()
	if err != nil {
		t.Error(err)
	}

	return dir
}
