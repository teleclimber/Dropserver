package appspacedb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// individual connection

type dbConn struct {
	//config *domain.RuntimeConfig	// or maybe it could just be dumb, and take a full path and that's it.
	handle     *sql.DB
	statements map[string]*sql.Stmt // doe sit need to be a pointer?
}

type results struct {
	LastInsertID int64 `json:"last_insert_id"`
	RowsAffected int64 `json:"rows_affected"`
}

// should know something about itself? like appspace,path, ...
// should track its lru time
// should hold on to prepared statements

// Copy from host db

func openConn(dbPath string, dbName string, create bool) (*dbConn, domain.Error) { // maybe this should take "create" flag
	dbFile := filepath.Join(dbPath, dbName+".db")
	dsn := "file:" + dbFile + "?mode=rw"

	if create {
		_, err := os.Stat(dbFile)
		if !os.IsNotExist(err) {
			return nil, dserror.New(dserror.AppspaceDBFileExists, "DB file: "+dbFile)
		}
		dsn += "c"
	}

	handle, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	err = handle.Ping()
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	return &dbConn{
		handle:     handle,
		statements: make(map[string]*sql.Stmt),
	}, nil
}

func (dbc *dbConn) close() {
	if dbc.handle != nil {
		dbc.handle.Close()
	}
}

func (dbc *dbConn) run(queryData *QueryData) ([]byte, domain.Error) {
	stmt, dsErr := dbc.getStatement(queryData.SQL)
	if dsErr != nil {
		return nil, dsErr
	}

	var args []interface{}
	var makeArgErr domain.Error
	if queryData.NamedParams != nil {
		args = make([]interface{}, len(queryData.NamedParams))
		index := 0
		for key, param := range queryData.NamedParams {
			makeArgErr = makeArg(&args, index, param, key)
			if makeArgErr != nil {
				break
			}
			index++
		}
	} else if queryData.Params != nil {
		args = make([]interface{}, len(queryData.Params))
		for i, param := range queryData.Params {
			makeArgErr = makeArg(&args, i, param, "")
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

	if queryData.Type == "query" {
		return dbc.query(stmt, &ifs)
	}
	return dbc.exec(stmt, &ifs)
}

// placeholder so we can an idea what is needed.
func (dbc *dbConn) query(stmt *sql.Stmt, args *[]interface{}) ([]byte, domain.Error) {
	rows, err := stmt.Query(*args...)
	if err != nil {
		return nil, dserror.New(dserror.AppspaceDBQueryError) //would be nice to know the actual error!
	}
	defer rows.Close()

	scanned, dsErr := scanRows(rows)
	if dsErr != nil {
		return nil, dserror.New(dserror.AppspaceDBScanError)
	}

	results := map[string]interface{}{
		"results": scanned,
	}

	json, err := json.Marshal(results)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	return json, nil
}

func (dbc *dbConn) exec(stmt *sql.Stmt, args *[]interface{}) ([]byte, domain.Error) {
	r, err := stmt.Exec(*args...)
	if err != nil {
		return nil, dserror.New(dserror.AppspaceDBQueryError, err.Error()) //would be nice to know the actual error!
	}

	var results results

	results.LastInsertID, err = r.LastInsertId()
	if err != nil {
		return nil, dserror.FromStandard(err)
	}
	results.RowsAffected, err = r.RowsAffected()
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	json, err := json.Marshal(results)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	return json, nil
}

func (dbc *dbConn) getStatement(q string) (*sql.Stmt, domain.Error) {
	s, ok := dbc.statements[q]
	if !ok {
		var err error
		s, err = dbc.handle.Prepare(q)
		if err != nil {
			return nil, dserror.FromStandard(err)
		}
		dbc.statements[q] = s
	}
	return s, nil
}

func makeArg(args *[]interface{}, index int, param interface{}, name string) domain.Error {
	if str, ok := (param).(string); ok {
		(*args)[index] = sql.Named(name, str)
	} else if float, ok := (param).(float64); ok {
		(*args)[index] = sql.Named(name, float)
		// } else if num, ok := (param).(int64); ok {
		// 	(*args)[index] = sql.Named(name, num)
	} else if boo, ok := (param).(bool); ok {
		(*args)[index] = sql.Named(name, boo)
	} else {
		return dserror.New(dserror.AppspaceDBQueryError, fmt.Sprintf("unrecognized type in makeArgs %v: %v", index, name))
	}

	// TODO: do all the types...

	// see https://github.com/mattn/go-sqlite3/issues/697
	// for creating named queries

	return nil
}

// from https://stackoverflow.com/a/60386531/472819
// TODO: make this stream json instead of returning a big data structure.
func scanRows(rows *sql.Rows) ([]map[string]interface{}, domain.Error) {
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, dserror.FromStandard(err)
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
			return nil, dserror.FromStandard(err)
		}

		masterData := map[string]interface{}{}

		for i, v := range columnTypes {

			if z, ok := (scanArgs[i]).(*sql.NullString); ok {
				if !z.Valid {
					return nil, dserror.New(dserror.AppspaceDBScanError, "Invalid string scan", "column name: "+v.Name())
				}
				masterData[v.Name()] = z.String
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullInt64); ok {
				if !z.Valid {
					return nil, dserror.New(dserror.AppspaceDBScanError, "Invalid int64 scan", "column name: "+v.Name())
				}
				masterData[v.Name()] = z.Int64
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
				if !z.Valid {
					return nil, dserror.New(dserror.AppspaceDBScanError, "Invalid float64 scan", "column name: "+v.Name())
				}
				masterData[v.Name()] = z.Float64
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullTime); ok {
				if !z.Valid {
					return nil, dserror.New(dserror.AppspaceDBScanError, "Invalid time scan", "column name: "+v.Name())
				}
				masterData[v.Name()] = z.Time
				continue
			}

			return nil, dserror.New(dserror.AppspaceDBScanError, "Failed to match scan arg type", "column name: "+v.Name())
		}

		finalRows = append(finalRows, masterData)
	}

	return finalRows, nil
}
