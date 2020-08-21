//go:generate mockgen -destination=mocks.go -package=appspacedb -self_package=github.com/teleclimber/DropServer/cmd/ds-host/appspacedb github.com/teleclimber/DropServer/cmd/ds-host/appspacedb DbConnI

// The API for interacting with these DBs has to be versioned!

// somehow execute and return db statements for appspace dbs
// - there are large numbers of appspace DBs
// - they are probably all sqlite (for now)
// - they may have different interfaces: sql, kv, and json?
// - There can be multiple DBs per appspace
// - For now let's make this an http handler, basically

// Request comes in with values:
// - has appspace (id or pointer to)
// - has app (why?)
// - has a user id??
// - has an urltail

// Request itself:
// - get/post
// - db name
// - query

// Process:
// - from appspace and db name (from urltail) and config, can get path to relevant DB
// - It's possible it's already open. Use existing conn if it's open.
// - Perform query based on url tail and params and payload
// - return data as appropriate

// I think we'll need an appspace db manager
// not unlike sandbox manager
// that sees if there is an open connection to that DB, and uses that if so.
// It should also routinely kill the least recently used ones.

// Then there are the actual connections and query executors,
// .. which will probably be their own struct.

package appspacedb

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// some copy-pasta from the host db manager

// TODO launch this from ds-host?

// Manager manages the connection for the database
type Manager struct {
	Config        *domain.RuntimeConfig
	Validator     domain.Validator
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, domain.Error)
	}

	connsMux sync.Mutex
	conns    map[connsKey]*connsVal
}

type connsKey struct {
	appspaceID domain.AppspaceID
	dbName     string
}

// DbConnI is an interface exported for testing purposes, not for use outside package
type DbConnI interface {
	close()
	run(queryData *QueryData) ([]byte, domain.Error)
}

type connsVal struct {
	dbConn       DbConnI
	statusMux    sync.Mutex // not 100% sure what it's covering.
	connError    domain.Error
	readySub     []chan struct{}
	liveRequests int // I think this counts ongoing requests that are "claimed" towards this conn. Can't close unless it's zero
}

// CreateDbData is the daata structure for creating a new database file
// add type for db type (sql, kv, ...)
type CreateDbData struct {
	Name string `json:"name"`
}

// QueryData is the structure expected when Posting a DB request
type QueryData struct {
	Type        string                 `json:"type"` // "query" or "exec"
	SQL         string                 `json:"sql"`
	Params      []interface{}          `json:"params"`
	NamedParams map[string]interface{} `json:"named-params"`
}

// Init makes the necessary values.
func (m *Manager) Init() {
	m.conns = make(map[connsKey]*connsVal)
}

// TODO we need an additional handler for rev listenr commands

// ServeHTTP is the entry point for all db requests
// appspace ID is already determined
// UrlTail is /<db-name?>/ or / to create a new DB
// ..or we could do /relational/<db-name?> in case we want to use / for other things later?
// ..also allows us to put other DB types under /keyvalue/
func (m *Manager) ServeHTTP(res http.ResponseWriter, req *http.Request, urlTail string, appspaceID domain.AppspaceID) {
	head, _ := shiftpath.ShiftPath(urlTail)

	if head == "" {
		if req.Method == http.MethodPost {
			m.createDBHandler(res, req, appspaceID)
		} else {
			http.Error(res, "Must be POST", http.StatusNotFound)
			return
		}
		// TODO: Actually GET should be a listing of DBs with their type. Duh.

	} else {
		// here head supposedly means DB name.
		// let's validate its size and characters
		dbName := strings.ToLower(head)
		dsErr := m.Validator.DBName(dbName)
		if dsErr != nil {
			// should log these kinds of errors too?
			dsErr.HTTPError(res)
			return
		}

		key := connsKey{
			appspaceID: appspaceID,
			dbName:     dbName}
		c := m.getConn(key)

		// now we have a connsval, but it's either ready to query or it errored
		if c.connError != nil {
			c.connError.HTTPError(res)
		} else {
			// here you should have a dbConn
			// prob send that to another function to actually do qeury / exec
			// m.dbCall...
			m.dbRun(res, req, c)
		}

		c.statusMux.Lock()
		c.liveRequests-- // this may need to work differently, but ok for now
		c.statusMux.Unlock()
	}
}

func (m *Manager) createDBHandler(res http.ResponseWriter, req *http.Request, appspaceID domain.AppspaceID) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var data CreateDbData
	err = json.Unmarshal(body, &data)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	dbName := strings.ToLower(data.Name)
	dsErr := m.Validator.DBName(dbName)
	if dsErr != nil {
		// should log these kinds of errors too?
		dsErr.HTTPError(res)
		return
	}

	m.connsMux.Lock()
	defer m.connsMux.Unlock()

	key := connsKey{
		appspaceID: appspaceID,
		dbName:     dbName}
	m.conns[key] = &connsVal{
		readySub:     []chan struct{}{},
		liveRequests: 1,
	}

	m.startConn(m.conns[key], key, true)

	m.conns[key].liveRequests--

	res.WriteHeader(http.StatusOK)
}

// getConn should open a conn and return the dbconn
// or return the existing dbconn, after waiting for it to be ready
// OR, if there was an error condition, return or mitigate....
func (m *Manager) getConn(key connsKey) *connsVal {
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

		go m.startConn(conn, key, false)
	}
	m.connsMux.Unlock()

	if readyChan != nil {
		_ = <-readyChan
	}

	return conn
}

func (m *Manager) startConn(c *connsVal, key connsKey, create bool) {
	appspace, dsErr := m.AppspaceModel.GetFromID(key.appspaceID)
	if dsErr != nil {
		c.connError = dsErr
		return // TODO: need to release chanels that are waiting
	}

	dbPath := filepath.Join(m.Config.Exec.AppspacesPath, appspace.LocationKey)
	dbConn, dsErr := openConn(dbPath, key.dbName, create)
	c.statusMux.Lock()
	if dsErr != nil {
		c.connError = dsErr
	} else {
		c.dbConn = dbConn
	}

	// then release all the channels that are waiting
	for _, ch := range c.readySub {
		close(ch)
	}
	c.statusMux.Unlock()
}

// I guess we'll alaso have a createDB call that actually creates things

func (m *Manager) dbRun(res http.ResponseWriter, req *http.Request, c *connsVal) {
	var qData *QueryData
	var dsErr domain.Error

	switch req.Method {
	case http.MethodGet:
		URLParams, err := url.ParseQuery(req.URL.RawQuery)
		if err != nil {
			// just return an error. No need to try to do db stuf with a malformed query string.
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if jsonStr, ok := URLParams["json-query"]; ok {
			if len(jsonStr) != 1 {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
			qData, dsErr = queryDataFromJSON([]byte(jsonStr[0]))
			if dsErr != nil {
				dsErr.HTTPError(res)
				return
			}

		} else {
			// throw error for now, but consider a different form of db get:
			// ?sql=select%20from&foo=bar
			// ..and build the QueryData struct from individual params in the url
			// this should probably be exec=...sql... or query=...sql... to avoid having to do type=exec...
			res.WriteHeader(http.StatusNotFound)
			return
		}
	case http.MethodPost:
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(body, qData)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
	default:
		res.WriteHeader(http.StatusNotFound)
		// ^^ need to make errors more useful!
	}

	if qData != nil {
		jsonBytes, dsErr := c.dbConn.run(qData) // TODO: run should accept io.Writer and we can stream the whole thing.
		if dsErr != nil {
			http.Error(res, dsErr.ExtraMessage(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		_, err := res.Write(jsonBytes)
		if err != nil {
			// TODO: log it because the response is already gone.
		}
	}
}

func queryDataFromJSON(jsonBytes []byte) (*QueryData, domain.Error) {
	var data QueryData
	err := json.Unmarshal(jsonBytes, &data)
	if err != nil {
		return nil, dserror.New(dserror.AppspaceDBQueryError, "Failed to parse JSON query data:"+err.Error())
	}

	// basic validation? Or best left to later?
	if data.Type != "exec" && data.Type != "query" {
		return nil, dserror.New(dserror.AppspaceDBQueryError, "Error in query JSON: type shoul be exec or query")
	}

	if data.Params != nil && data.NamedParams != nil {
		return nil, dserror.New(dserror.AppspaceDBQueryError, "Can not have both parameter array and named parameters in query.")
	}

	return &data, nil
}
