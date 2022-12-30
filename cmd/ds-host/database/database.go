package database

import (
	"database/sql"
	"errors"
	"fmt"
	"path"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// Manager manages the connection for the database
type Manager struct {
	Config *domain.RuntimeConfig `checkinject:"required"`
	handle *sqlx.DB
}

// Open connects to Database and returns the handle
func (dbm *Manager) Open() (*domain.DB, error) {
	// In the context of app startup, DB errors are like user errors, where the user is the admin.
	// Could give nice errors using error codes.

	dbPath := path.Join(dbm.Config.DataDir, "host-data.db")

	handle, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		panic("Failed to open DB " + err.Error())
	}

	dbm.handle = handle

	err = handle.Ping()
	if err != nil {
		panic("Failed to ping DB " + err.Error())
	}

	return &domain.DB{
		Handle: handle}, nil

}

// maybe a close function?

// GetHandle returns the db handle
func (dbm *Manager) GetHandle() *domain.DB { // return error in case the handle is nil?
	// handle may not exist if we were unable to open the DB
	return &domain.DB{
		Handle: dbm.handle}

}

// GetSchema returns the schema as its written in the db metadata table.
func (dbm *Manager) GetSchema() string {

	if dbm.handle == nil {
		fmt.Println("GetSchema handle is nil")
		return ""
	}

	h := dbm.handle

	var numTable int
	err := h.Get(&numTable, `SELECT count(*) FROM sqlite_master WHERE type="table" AND name="params"`)
	if err != nil {
		panic(err)
	}

	if numTable == 0 {
		fmt.Println("GetSchema num table is 0")
		return ""
	}

	var dbSchema string
	err = h.Get(&dbSchema, `SELECT value FROM params WHERE name="db_schema"`)
	if err != nil {
		panic(err)
	}

	return dbSchema
}

// SetSchema sets the schema on the db metada table.
func (dbm *Manager) SetSchema(schema string) error {
	_, err := dbm.handle.Exec(`UPDATE params SET value=? WHERE name="db_schema"`, schema)
	if err != nil {
		return err
	}
	return nil
}

// GetSchema returns the schema as its written in the db metadata table.
func (dbm *Manager) GetSetupKey() (string, error) {
	if dbm.handle == nil {
		err := errors.New("no db handle available")
		dbm.getLogger("GetSetupKey()").Error(err)
		return "", err
	}
	h := dbm.handle

	var numTable int
	err := h.Get(&numTable, `SELECT count(*) FROM sqlite_master WHERE type="table" AND name="params"`)
	if err != nil {
		dbm.getLogger("GetSetupKey() Select table").Error(err)
		return "", err
	}
	if numTable == 0 {
		err = errors.New("no params table available")
		dbm.getLogger("GetSetupKey() Select table").Error(err)
		return "", err
	}

	var setupKey string
	err = h.Get(&setupKey, `SELECT value FROM params WHERE name="setup_key"`)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		dbm.getLogger("GetSetupKey() Select setup_key").Error(err)
		return "", err
	}

	return setupKey, nil
}

func (dbm *Manager) DeleteSetupKey() error {
	if dbm.handle == nil {
		fmt.Println("GetSchema handle is nil")
		return errors.New("no db handle available")
	}
	h := dbm.handle

	_, err := h.Exec(`DELETE FROM params WHERE name="setup_key"`)
	if err != nil {
		dbm.getLogger("DeleteSetupKey()").Error(err)
	}
	return err
}

func (dbm *Manager) getLogger(note string) *record.DsLogger {
	return record.NewDsLogger("DatabaseManager", note)
}
