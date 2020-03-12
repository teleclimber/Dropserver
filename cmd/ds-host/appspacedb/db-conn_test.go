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

	"github.com/teleclimber/DropServer/internal/dserror"
)

func TestMakeArgs(t *testing.T) {
	args := make([]interface{}, 1)
	//param := interface{}(float64(7))
	param := float64(7)

	dsErr := makeArg(&args, 0, param, "")
	if dsErr != nil {
		t.Error(dsErr)
	}

	val := (args[0]).(sql.NamedArg).Value
	if float, ok := (val).(float64); !ok {
		t.Error("expected float 64")
	} else if float != 7 {
		t.Errorf("expected value of 7, got %v", float)
	}

	// TODO test more types
}

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

	scanned, dsErr := scanRows(rows)
	if dsErr != nil {
		t.Error(dsErr)
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

	stmt, dsErr := dbc.getStatement(`CREATE TABLE "apps" (
		"owner_id" INTEGER,
		"app_id" INTEGER PRIMARY KEY ASC,
		"name" TEXT,
		"created" DATETIME,
		"usage" REAL
	)`)
	if dsErr != nil {
		t.Error(dsErr)
	}

	args := make([]interface{}, 0)

	jsonBytes, dsErr := dbc.exec(stmt, &args)
	if dsErr != nil {
		t.Error(dsErr)
	}

	var createResults results
	err = json.Unmarshal(jsonBytes, &createResults)
	if err != nil {
		t.Error(err)
	}

	stmt, dsErr = dbc.getStatement(`INSERT INTO apps VALUES (?, ?, ?, datetime("now"), ?)`)
	if dsErr != nil {
		t.Error(dsErr)
	}

	// TODO: test row insert
	args = []interface{}{
		sql.Named("", float64(1)),
		sql.Named("", float64(7)),
		sql.Named("", "some app"),
		sql.Named("", float64(999.9))}
	jsonBytes, dsErr = dbc.exec(stmt, &args)
	if dsErr != nil {
		t.Error(dsErr)
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

	stmt, dsErr := dbc.getStatement("SELECT * FROM apps WHERE app_id = ? ORDER BY owner_id ")
	if dsErr != nil {
		t.Error(dsErr)
	}

	jsonBytes, dsErr := dbc.query(stmt, &[]interface{}{sql.Named("", float64(11))})
	if dsErr != nil {
		t.Error(dsErr)
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

	stmt, dsErr := dbc.getStatement(`CREATE TABLE "apps" (
		"owner_id" INTEGER,
		"app_id" INTEGER PRIMARY KEY ASC,
		"name" TEXT,
		"created" DATETIME,
		"usage" REAL
	)`)
	if dsErr != nil {
		t.Error(dsErr)
	}

	_, dsErr = dbc.exec(stmt, &[]interface{}{})
	if dsErr != nil {
		t.Error(dsErr)
	}

	insStmt, dsErr := dbc.getStatement(`INSERT INTO apps VALUES (:owner_id, :app_id, :name, datetime("now"), :usage)`)
	if dsErr != nil {
		t.Error(dsErr)
	}

	_, dsErr = dbc.exec(insStmt, &[]interface{}{
		sql.Named("app_id", 7),
		sql.Named("usage", 999.9),
		sql.Named("name", "some app"),
		sql.Named("owner_id", 1)})
	if dsErr != nil {
		t.Error(dsErr)
	}

	_, dsErr = dbc.exec(insStmt, &[]interface{}{
		sql.Named("app_id", 11),
		sql.Named("usage", 77.77),
		sql.Named("name", "another app"),
		sql.Named("owner_id", 2)})
	if dsErr != nil {
		t.Error(dsErr)
	}

	stmt, dsErr = dbc.getStatement("SELECT * FROM apps WHERE app_id = ? ORDER BY owner_id ")
	if dsErr != nil {
		t.Error(dsErr)
	}

	jsonBytes, dsErr := dbc.query(stmt, &[]interface{}{sql.Named("", float64(11))})
	if dsErr != nil {
		t.Error(dsErr)
	}

	jsonStr := string(jsonBytes[:])

	if !strings.Contains(jsonStr, `"usage":77.77`) {
		t.Errorf("json should contain substring. json: %v", jsonStr)
	}

	// test bad param lists
	_, dsErr = dbc.exec(insStmt, &[]interface{}{
		sql.Named("app_id", 7),
		sql.Named("usage", 999.9),
		//sql.Named("name", "some app"),
		sql.Named("owner_id", 1)})
	if dsErr == nil {
		t.Error("expected an error for missing a parameter")
	}

	_, dsErr = dbc.exec(insStmt, &[]interface{}{
		sql.Named("app_id", 7),
		sql.Named("usage", 999.9),
		sql.Named("nameZZZ", "some app"),
		sql.Named("owner_id", 1)})
	if dsErr == nil {
		t.Error("expected an error for non-existent nameZZZ parameter")
	}

	_, dsErr = dbc.exec(insStmt, &[]interface{}{
		sql.Named("app_id", "foo"), // string as number
		sql.Named("usage", 999.9),
		sql.Named("name", "some app"),
		sql.Named("owner_id", 1)})
	if dsErr == nil {
		t.Error("expected an error for wrong type parameter")
	}
}

func TestRun(t *testing.T) {
	handle, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	dbc := &dbConn{
		handle:     handle,
		statements: make(map[string]*sql.Stmt),
	}

	qd := QueryData{
		SQL: `CREATE TABLE "apps" (
			"owner_id" INTEGER,
			"app_id" INTEGER PRIMARY KEY ASC,
			"name" TEXT,
			"created" DATETIME,
			"usage" REAL
		)`,
		Type: "exec"}

	_, dsErr := dbc.run(&qd)
	if dsErr != nil {
		t.Error(dsErr)
	}

	qd = QueryData{
		SQL:    `INSERT INTO apps VALUES (?, ?, ?, datetime("now"),?)`,
		Type:   "exec",
		Params: []interface{}{float64(1), float64(7), "some app", 77.77}}
	_, dsErr = dbc.run(&qd)
	if dsErr != nil {
		t.Error(dsErr)
	}

	qd.Params = []interface{}{float64(1), float64(11), "some other app", 999.9}
	_, dsErr = dbc.run(&qd)
	if dsErr != nil {
		t.Error(dsErr)
	}

	np := make(map[string]interface{})
	np["app_id"] = float64(11)
	qd = QueryData{
		SQL:         `SELECT * FROM apps WHERE app_id = :app_id`,
		Type:        "query",
		NamedParams: np}
	jsonBytes, dsErr := dbc.run(&qd)
	if dsErr != nil {
		t.Error(dsErr)
	}

	jsonStr := string(jsonBytes[:])
	if !strings.Contains(jsonStr, `"usage":999.9`) {
		t.Errorf("json should contain substring. json: %v", jsonStr)
	}
}

func TestOpenNoDB(t *testing.T) {
	// create temp dir for the DB
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	_, dsErr := openConn(dir, "test", false)
	if dsErr == nil { // actually it should error!
		t.Error("Should have failed to open non existent DB")
	}
}

func TestCreate(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	dsc, dsErr := openConn(dir, "test", true)
	if dsErr != nil {
		t.Error(dsErr)
	}
	defer dsc.close()
}

func TestCreateExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	emptyFile, err := os.Create(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Error(err)
	}
	emptyFile.Close()

	_, dsErr := openConn(dir, "test", true)
	if dsErr == nil {
		t.Error("should have errored trying to create pre-existing file")
	} else if dsErr.Code() != dserror.AppspaceDBFileExists {
		t.Error("wrong error")
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
