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
