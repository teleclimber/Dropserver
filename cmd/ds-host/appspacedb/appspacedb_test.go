package appspacedb

import (
	"bytes"
	"database/sql"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

// json to qeury data
func TestQueryDataFromJSON(t *testing.T) {
	j := []byte(`{"type":"exec", "sql":"SELECT * FROM table"}`)

	data, dsErr := queryDataFromJSON(j)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if data.SQL != "SELECT * FROM table" {
		t.Error("where did sql go?")
	}
}

func TestQueryDataFromJSONWithNamedParams(t *testing.T) {
	j := []byte(`{"type":"exec", "sql":"SELECT * FROM table", "named-params":{"param1":7, "param2":"foo", "param3":true}}`)
	// could also have boolean, and consider posibliity of null

	data, dsErr := queryDataFromJSON(j)
	if dsErr != nil {
		t.Error(dsErr)
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

	data, dsErr := queryDataFromJSON(j)
	if dsErr != nil {
		t.Error(dsErr)
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

	_, dsErr := queryDataFromJSON(j)
	if dsErr != nil {
		t.Error(dsErr)
	}

	// it doesn't error at the unmarshall stage, so it's going to be dealt with at the quer stage.
}

func TestStartConn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	loc := "abc"
	dir := makeAppspaceDB(t, loc)
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	appspaceID := domain.AppspaceID(13)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(domain.AppspaceID(13)).Return(&domain.Appspace{LocationKey: "abc"}, nil)

	m := &Manager{
		Config:        cfg,
		AppspaceModel: appspaceModel,
	}
	m.Init()

	key := connsKey{
		appspaceID: appspaceID,
		dbName:     "test",
	}

	readyChan := make(chan struct{})
	c := &connsVal{
		readySub: []chan struct{}{readyChan},
	}

	go m.startConn(c, key, false)

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

func TestCreateDbHandler(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	loc := "abc"
	appspaceID := domain.AppspaceID(13)

	dir := makeAppspaceDB(t, loc)
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().DBName("some-db").Return(nil)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{LocationKey: "abc"}, nil)

	m := &Manager{
		Config:        cfg,
		Validator:     validator,
		AppspaceModel: appspaceModel,
	}
	m.Init()

	jsonStr := []byte(`{"name":"some-db"}`)
	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	m.createDBHandler(rr, req, appspaceID)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, with message: %v", rr.Code, http.StatusOK, rr.Body)
	}

}

func TestGetConn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	loc := "abc"
	dir := makeAppspaceDB(t, loc)
	defer os.RemoveAll(dir)

	appspaceID := domain.AppspaceID(13)

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{LocationKey: "abc"}, nil)

	m := &Manager{
		Config:        cfg,
		AppspaceModel: appspaceModel,
	}
	m.Init()

	key := connsKey{
		appspaceID: appspaceID,
		dbName:     "test",
	}

	c := m.getConn(key)

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

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	appspaceID := domain.AppspaceID(13)

	m := &Manager{
		Config: cfg,
	}
	m.Init()

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

	c := m.getConn(key)

	// test that live requests was incremented to 11,
	// which indicates both attempts to get conn return the same conn
	if c.liveRequests != 11 {
		t.Error("expected live requests to be 11")
	}

}

func TestGetConnSecond(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	loc := "abc"
	dir := makeAppspaceDB(t, loc)
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = dir

	appspaceID := domain.AppspaceID(13)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{LocationKey: "abc"}, nil)

	m := &Manager{
		Config:        cfg,
		AppspaceModel: appspaceModel,
	}
	m.Init()

	key := connsKey{
		appspaceID: appspaceID,
		dbName:     "test",
	}

	c1 := m.getConn(key)

	c2 := m.getConn(key)

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

func TestDBRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := &Manager{}

	dbc := NewMockDbConnI(mockCtrl)
	dbc.EXPECT().run(gomock.Any()).Return([]byte(`{"results":[{"name":"some app"}]}`), nil)

	c := &connsVal{dbConn: dbc}

	req, err := http.NewRequest("GET", "/somedb?json-query="+url.QueryEscape(`{"type":"query", "sql":"SELECT * FROM apps"}`), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	m.dbRun(rr, req, c)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, with message: %v", rr.Code, http.StatusOK, rr.Body)
	}

	// Check the response body is what we expect.
	expected := `{"results":[{"name":"some app"}]}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

// TODO: test POST and errors.
// TODO: test ServeHTTP

////////////////
// helpers..
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
