package appspacedb

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

// Wonder if this should be entirely functional?
// pass a conn and a packaged request to a handler

// But what about Twine handler?

//

// V0QueryData is the structure expected when Posting a DB request
// TODO This probably needs to go in domain?
type V0QueryData struct {
	// AppspaceID  domain.AppspaceID      `json:"appspace_id`
	// DBName      string                 `json:"db_name"`
	Type        string                 `json:"type"` // "query" or "exec"
	SQL         string                 `json:"sql"`
	Params      []interface{}          `json:"params"`
	NamedParams map[string]interface{} `json:"named_params"`
}

type results struct { // this is vX api-specific stuff
	LastInsertID int64 `json:"last_insert_id"`
	RowsAffected int64 `json:"rows_affected"`
}

// V0 is the appspace db interface at dropserver API version 0
type V0 struct {
	connManager interface {
		getConn(appspaceID domain.AppspaceID, locationKey string, dbName string) *connsVal
	}
}

//GetService returns a twine service for the appspace
func (v *V0) GetService(appspace *domain.Appspace) domain.ReverseServiceI {
	service := &V0Service{
		connManager: v.connManager,
		appspace:    appspace,
		dbs:         make(map[string]*dbConn)}

	return service
}

// Run a query on the appspace db
// Not sure at all how to handle the return values
// How do we enable sending back a series of row datas, or something like that?
// Returning a []byte seems a bit contrived.
// Maybe an additional parameter? Or....?
func (v *V0) Run(appspace *domain.Appspace, dbName string, qData V0QueryData) ([]byte, error) {
	conn, err := v.getConn(appspace, dbName)
	if err != nil {
		return nil, err
	}

	stmt, err := conn.getStatement(qData.SQL)
	if err != nil {
		return nil, err
	}

	var args []interface{}
	var makeArgErr error
	if qData.NamedParams != nil {
		args = make([]interface{}, len(qData.NamedParams))
		index := 0
		for key, param := range qData.NamedParams {
			makeArgErr = v0makeArg(&args, index, param, key)
			if makeArgErr != nil {
				break
			}
			index++
		}
	} else if qData.Params != nil {
		args = make([]interface{}, len(qData.Params))
		for i, param := range qData.Params {
			makeArgErr = v0makeArg(&args, i, param, "")
			if makeArgErr != nil {
				break
			}
		}
	}
	if makeArgErr != nil {
		return nil, makeArgErr
	}

	ifs := make([]interface{}, len(args))
	for i := range args {
		ifs[i] = args[i]
	}

	if qData.Type == "query" {
		return conn.query(stmt, &ifs)
	}
	return conn.exec(stmt, &ifs)
}

func (v *V0) getConn(appspace *domain.Appspace, dbName string) (*dbConn, error) {
	connVal := v.connManager.getConn(appspace.AppspaceID, appspace.LocationKey, dbName)
	if connVal.connError != nil {
		return nil, connVal.connError
	}
	return connVal.dbConn, nil
}

// V0Service is a twine service for a given appspace.
type V0Service struct {
	connManager interface {
		getConn(appspaceID domain.AppspaceID, locationKey string, dbName string) *connsVal
	}
	appspace *domain.Appspace
	dbs      map[string]*dbConn // need one per DB
}

// HandleMessage takes a twine message and performs the desired op
func (s *V0Service) HandleMessage(message twine.ReceivedMessageI) {

	// message should include db name
	// .. so check if it's in map or get it and stash it.

}

func v0makeArg(args *[]interface{}, index int, param interface{}, name string) error {
	if str, ok := (param).(string); ok {
		(*args)[index] = sql.Named(name, str)
	} else if float, ok := (param).(float64); ok {
		(*args)[index] = sql.Named(name, float)
		// } else if num, ok := (param).(int64); ok {
		// 	(*args)[index] = sql.Named(name, num)
	} else if boo, ok := (param).(bool); ok {
		(*args)[index] = sql.Named(name, boo)
	} else {
		return fmt.Errorf("unrecognized type in makeArgs %v: %v", index, name)
	}

	// TODO: do all the types...

	// see https://github.com/mattn/go-sqlite3/issues/697
	// for creating named queries

	return nil
}

func v0queryDataFromJSON(jsonBytes []byte) (*V0QueryData, error) {
	var data V0QueryData
	err := json.Unmarshal(jsonBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("Appspace DB Query Error: Failed to parse JSON query data: %s", err.Error())
	}

	// basic validation? Or best left to later?
	if data.Type != "exec" && data.Type != "query" {
		return nil, fmt.Errorf("Appspace DB Query Error: type should be exec or query")
	}

	if data.Params != nil && data.NamedParams != nil {
		return nil, fmt.Errorf("Appspace DB Query Error: Can not have both parameter array and named parameters in query")
	}

	return &data, nil
}
