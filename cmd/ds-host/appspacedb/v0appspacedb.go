package appspacedb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
	"github.com/teleclimber/DropServer/internal/validator"
)

type results struct { // this is vX api-specific stuff
	LastInsertID int64 `json:"last_insert_id"`
	RowsAffected int64 `json:"rows_affected"`
}

// V0 is the appspace db interface at dropserver API version 0
type V0 struct {
	connManager interface {
		createDB(appspaceID domain.AppspaceID, locationKey string, dbName string) (*connsVal, error)
		deleteDB(appspaceID domain.AppspaceID, locationKey string, dbName string) error
		getConn(appspaceID domain.AppspaceID, locationKey string, dbName string) *connsVal
	}
}

//GetService returns a twine service for the appspace
func (v *V0) GetService(appspace *domain.Appspace) domain.ReverseServiceI {
	service := &V0Service{
		connManager: v.connManager,
		appspace:    appspace}

	return service
}

// Run a query on the appspace db
// Not sure at all how to handle the return values
// How do we enable sending back a series of row datas, or something like that?
// Returning a []byte seems a bit contrived.
// Maybe an additional parameter? Or....?
func (v *V0) Run(appspace *domain.Appspace, dbName string, qData domain.V0AppspaceDBQuery) ([]byte, error) {
	conn, err := v.getConn(appspace, dbName)
	if err != nil {
		return nil, err
	}

	stmt, err := conn.getStatement(qData.SQL)
	if err != nil {
		return nil, err
	}

	args, err := v0makeArgs(qData)
	if err != nil {
		return nil, err
	}

	if qData.Type == "query" {
		return conn.query(stmt, args)
	}
	return conn.exec(stmt, args)
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
		createDB(appspaceID domain.AppspaceID, locationKey string, dbName string) (*connsVal, error)
		deleteDB(appspaceID domain.AppspaceID, locationKey string, dbName string) error
		getConn(appspaceID domain.AppspaceID, locationKey string, dbName string) *connsVal
	}
	appspace *domain.Appspace
}

const v0createDBCommand = 11
const v0deleteDBCommand = 13
const v0queryDBCommand = 20

// HandleMessage takes a twine message and performs the desired op
func (s *V0Service) HandleMessage(message twine.ReceivedMessageI) {
	switch message.CommandID() {
	case v0createDBCommand:
		s.handleCreateDB(message)
	case v0deleteDBCommand:
		s.handleDeleteDB(message)
	case v0queryDBCommand:
		s.handleQueryDB(message)
	default:
		message.SendError("appspace db command not recognized")
	}
}

// createDbData is the daata structure for creating a new database file
// add type for db type (sql, kv, ...)
type createDbData struct {
	DBName string `json:"db_name"`
}

func (s *V0Service) handleCreateDB(message twine.ReceivedMessageI) {
	var data createDbData
	err := json.Unmarshal(message.Payload(), &data)
	if err != nil {
		message.SendError("failed to decode query data json: " + err.Error())
		return
	}

	dbName := strings.ToLower(data.DBName)
	err = validator.DBName(dbName)
	if err != nil {
		message.SendError("dbname validation error")
		return
	}

	_, err = s.connManager.createDB(s.appspace.AppspaceID, s.appspace.LocationKey, dbName)
	if err != nil {
		message.SendError(fmt.Sprintf("failed to create DB: %s", err.Error()))
		return
	}

	message.SendOK()
}

func (s *V0Service) handleDeleteDB(message twine.ReceivedMessageI) {
	var data createDbData // same data structure for delete
	err := json.Unmarshal(message.Payload(), &data)
	if err != nil {
		message.SendError("failed to decode query data json: " + err.Error())
		return
	}

	dbName := strings.ToLower(data.DBName)
	err = validator.DBName(dbName)
	if err != nil {
		message.SendError("dbname validation error")
		return
	}

	err = s.connManager.deleteDB(s.appspace.AppspaceID, s.appspace.LocationKey, dbName)
	if err != nil {
		message.SendError(fmt.Sprintf("failed to delete DB: %s", err.Error()))
		return
	}

	message.SendOK()
}

func (s *V0Service) handleQueryDB(message twine.ReceivedMessageI) {
	var qData domain.V0AppspaceDBQuery
	err := json.Unmarshal(message.Payload(), &qData)
	if err != nil {
		message.SendError("failed to decode query data json: " + err.Error())
		return
	}

	connVal := s.connManager.getConn(s.appspace.AppspaceID, s.appspace.LocationKey, qData.DBName)
	if connVal.connError != nil {
		message.SendError(fmt.Sprintf("failed to get appspace db connection for db name %s: %s", qData.DBName, err.Error()))
		return
	}

	stmt, err := connVal.dbConn.getStatement(qData.SQL)
	if err != nil {
		message.SendError(fmt.Sprintf("failed to prepare statement \"%s\": %s", qData.SQL, err.Error()))
		return
	}

	args, err := v0makeArgs(qData)
	if err != nil {
		message.SendError(fmt.Sprintf("failed to make args %s", err.Error()))
		return
	}

	var result []byte
	if qData.Type == "query" {
		result, err = connVal.dbConn.query(stmt, args)
	} else if qData.Type == "exec" {
		result, err = connVal.dbConn.exec(stmt, args)
	} else {
		message.SendError(fmt.Sprintf("bad appspace query data type: %s", qData.Type))
		return
	}
	if err != nil {
		message.SendError(fmt.Sprintf("failed to %s on appspace DB %s: %s", qData.Type, qData.DBName, err.Error()))
		return
	}

	message.Reply(0, result)
}

func v0makeArgs(qData domain.V0AppspaceDBQuery) (args []interface{}, makeArgErr error) {
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
		return
	}

	ifs := make([]interface{}, len(args))
	for i := range args {
		ifs[i] = args[i]
	}

	return
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

func v0queryDataFromJSON(jsonBytes []byte) (*domain.V0AppspaceDBQuery, error) {
	var data domain.V0AppspaceDBQuery
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
