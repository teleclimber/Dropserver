package appspacedb

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestMakeArgs(t *testing.T) {
	args := make([]interface{}, 1)
	param := float64(7)

	err := v0makeArg(&args, 0, param, "")
	if err != nil {
		t.Error(err)
	}

	val := (args[0]).(sql.NamedArg).Value
	if float, ok := (val).(float64); !ok {
		t.Error("expected float 64")
	} else if float != 7 {
		t.Errorf("expected value of 7, got %v", float)
	}

	// TODO test more types
}

// json to qeury data
func TestQueryDataFromJSON(t *testing.T) {
	j := []byte(`{"type":"exec", "sql":"SELECT * FROM table"}`)

	data, err := v0queryDataFromJSON(j)
	if err != nil {
		t.Error(err)
	}

	if data.SQL != "SELECT * FROM table" {
		t.Error("where did sql go?")
	}
}

func TestQueryDataFromJSONWithNamedParams(t *testing.T) {
	j := []byte(`{"type":"exec", "sql":"SELECT * FROM table", "named_params":{"param1":7, "param2":"foo", "param3":true}}`)
	// could also have boolean, and consider posibliity of null

	data, err := v0queryDataFromJSON(j)
	if err != nil {
		t.Error(err)
	}

	if data.SQL != "SELECT * FROM table" {
		t.Error("where did sql go?")
	}
	if data.NamedParams == nil {
		t.Error("expected named params")
	}
	if data.Params != nil {
		t.Error("Params should be nil")
	}

	val, ok := data.NamedParams["param1"]
	if !ok {
		t.Error("param1 should exist")
	}
	num, ok := (val).(float64)
	if !ok {
		t.Error("param1 should be float64")
	} else if num != 7 {
		t.Errorf("expected 7, got %v", num)
	}

	val, ok = data.NamedParams["param2"]
	if !ok {
		t.Error("param2 should exist")
	}
	str, ok := (val).(string)
	if !ok {
		t.Error("param2 should be string")
	} else if str != "foo" {
		t.Errorf("expected foo, got %v", str)
	}

	val, ok = data.NamedParams["param3"]
	if !ok {
		t.Error("param3 should exist")
	}
	boo, ok := (val).(bool)
	if !ok {
		t.Error("param3 should be bool")
	} else if !boo {
		t.Errorf("expected true,got false")
	}

}

func TestQueryDataFromJSONWithParams(t *testing.T) {
	j := []byte(`{"type":"exec", "sql":"SELECT * FROM table", "params":[7,"foo", true]}`)
	// what about dates?

	data, err := v0queryDataFromJSON(j)
	if err != nil {
		t.Error(err)
	}

	if data.SQL != "SELECT * FROM table" {
		t.Error("where did sql go?")
	}
	if data.NamedParams != nil {
		t.Error("named params should be nil")
	}
	if data.Params == nil {
		t.Error("Params should not be nil")
	}

	if len(data.Params) != 3 {
		t.Error("should be elements in Params")
	}

	num, ok := (data.Params[0]).(float64)
	if !ok {
		t.Error("param 0 should be float 64")
	}
	if num != 7 {
		t.Errorf("num should be 7, got %v", num)
	}

	str, ok := (data.Params[1]).(string)
	if !ok {
		t.Error("param 0 should be string")
	}
	if str != "foo" {
		t.Errorf("str should be foo, got %v", str)
	}

	boo, ok := (data.Params[2]).(bool)
	if !ok {
		t.Error("param 2 should be boolean")
	}
	if !boo {
		t.Errorf("boolean should be true, got false")
	}

}

func TestQueryDataFromJSONWithNullParam(t *testing.T) {
	j := []byte(`{"type":"exec", "sql":"SELECT * FROM table", "params":[null,"foo"]}`)

	_, err := v0queryDataFromJSON(j)
	if err != nil {
		t.Error(err)
	}

	// it doesn't error at the unmarshall stage, so it's going to be dealt with at the quer stage.
}

func TestRun(t *testing.T) {
	appspace := &domain.Appspace{}

	v0 := &V0{
		connManager: &singleConnManager{},
	}

	qd := V0QueryData{
		SQL: `CREATE TABLE "apps" (
			"owner_id" INTEGER,
			"app_id" INTEGER PRIMARY KEY ASC,
			"name" TEXT,
			"created" DATETIME,
			"usage" REAL
		)`,
		Type: "exec"}

	_, err := v0.Run(appspace, "yada", qd)
	if err != nil {
		t.Error(err)
	}

	qd = V0QueryData{
		SQL:    `INSERT INTO apps VALUES (?, ?, ?, datetime("now"),?)`,
		Type:   "exec",
		Params: []interface{}{float64(1), float64(7), "some app", 77.77}}
	_, err = v0.Run(appspace, "yada", qd)
	if err != nil {
		t.Error(err)
	}

	qd.Params = []interface{}{float64(1), float64(11), "some other app", 999.9}
	_, err = v0.Run(appspace, "yada", qd)
	if err != nil {
		t.Error(err)
	}

	np := make(map[string]interface{})
	np["app_id"] = float64(11)
	qd = V0QueryData{
		SQL:         `SELECT * FROM apps WHERE app_id = :app_id`,
		Type:        "query",
		NamedParams: np}
	jsonBytes, err := v0.Run(appspace, "yada", qd)
	if err != nil {
		t.Error(err)
	}

	jsonStr := string(jsonBytes[:])
	if !strings.Contains(jsonStr, `"usage":999.9`) {
		t.Errorf("json should contain substring. json: %v", jsonStr)
	}
}

// singleConnManager is a dummy manager that creates a single in-memory db and alwasy returns it
type singleConnManager struct {
	conn *connsVal
}

func (m *singleConnManager) getConn(appspaceID domain.AppspaceID, locationKey string, dbName string) *connsVal {
	if m.conn != nil {
		return m.conn
	}

	// create an in-memory db
	handle, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	err = handle.Ping()
	if err != nil {
		panic("Failed to ping in-memory DB " + err.Error())
	}

	c := &dbConn{
		handle:     handle,
		statements: make(map[string]*sql.Stmt),
	}

	m.conn = &connsVal{
		dbConn: c,
		// other stuff?
	}

	return m.conn
}
