package database

import (
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

func TestGetSchemaNoHandle(t *testing.T) {
	dbm := &Manager{}

	s := dbm.GetSchema()
	if s != "" {
		t.Error("should have been empty string")
	}
}

func TestGetSchemaNoTables(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	dbm := &Manager{
		handle: handle}

	s := dbm.GetSchema()
	if s != "" {
		t.Error("should have been empty string")
	}
}

func TestSetSchema(t *testing.T) {
	handle := migrate.MakeSqliteDummyDB()
	defer handle.Close()

	dbm := &Manager{
		handle: handle}

	dsErr := dbm.SetSchema("foo")
	if dsErr != nil {
		t.Error(dsErr)
	}
}

func TestGetSchema(t *testing.T) {
	handle := migrate.MakeSqliteDummyDB()
	defer handle.Close()

	dbm := &Manager{
		handle: handle}

	dsErr := dbm.SetSchema("foo")
	if dsErr != nil {
		t.Error(dsErr)
	}

	s := dbm.GetSchema()
	if s != "foo" {
		t.Error("should have been foo")
	}
}

func TestGetSetupKey(t *testing.T) {
	handle := migrate.MakeSqliteDummyDB()
	defer handle.Close()

	dbm := &Manager{
		handle: handle}

	key, err := dbm.GetSetupKey()
	if err != nil {
		t.Error(err)
	}
	if key == "" {
		t.Error("migration should have set the key to something")
	}

	results, err := handle.Exec(`UPDATE "params" SET value = ? WHERE name = ?`, "abcdef", "setup_key")
	if err != nil {
		t.Error(err)
	}
	rows, err := results.RowsAffected()
	if err != nil {
		t.Error(err)
	}
	if rows != 1 {
		t.Error("bad test set up: failed to change row")
	}

	key, err = dbm.GetSetupKey()
	if err != nil {
		t.Error(err)
	}
	if key != "abcdef" {
		t.Error("migration should have set the key to something")
	}
}

func TestDeleteSetupKey(t *testing.T) {
	handle := migrate.MakeSqliteDummyDB()
	defer handle.Close()

	dbm := &Manager{
		handle: handle}

	err := dbm.DeleteSetupKey()
	if err != nil {
		t.Error(err)
	}
	key, err := dbm.GetSetupKey()
	if err != nil {
		t.Error(err)
	}
	if key != "" {
		t.Error("migration should have set the key to something")
	}
}
