package migrate

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// functions here are used solely to create databases
// for use in testing other things, like Models.

// MakeSqliteDummyDB creates a Sqlite in-memory DB
// and migrates it to the current schema.
// It is to be used for testing purposes only.
func MakeSqliteDummyDB() *sqlx.DB {
	// Beware of in-memory DBs: they vanish as soon as the connection closes!
	// We may be able to start a sqlx transaction to avoid problems with that?
	// See: https://github.com/jmoiron/sqlx/issues/164
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	db := &domain.DB{
		Handle: handle}

	args := &stepArgs{
		db: db}

	for _, s := range MigrationSteps {
		err := s.up(args)
		if err != nil {
			panic(err)
		}
	}

	// we should probably set schema?
	schema := MigrationSteps[len(MigrationSteps)-1].name
	_, err = handle.Exec(`UPDATE params SET value=? WHERE name="db_schema"`, schema)
	if err != nil {
		panic(err)
	}

	return handle
}
