package appspacemetadb

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestExec(t *testing.T) {
	dbe := getTestDBExec()

	dbe.exec(`CREATE TABLE info (
		"name" TEXT,
		"value" TEXT
	)`)

	if dbe.err != nil {
		t.Error(dbe.err)
	}

	//check the table was actually created please.
	var tables []string
	err := dbe.handle.Select(&tables, `SELECT name FROM sqlite_master WHERE type ='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Error(err)
	}
	if len(tables) != 1 {
		t.Error("expected one table")
	}
	if tables[0] != "info" {
		t.Error("expected table info")
	}
}

func TestExecError(t *testing.T) {
	dbe := getTestDBExec()

	dbe.exec(`CREATEzzzzz TABLE info (
		"name" TEXT,
		"value" TEXT
	)`)

	err := dbe.checkErr()
	if err == nil {
		t.Error("expected error")
	}
}

func TestMigrateUpToV0(t *testing.T) {
	dbe := getTestDBExec()

	migrateUpToV0(dbe)

	err := dbe.checkErr()
	if err != nil {
		t.Error(err)
	}
}

func getTestDBExec() *dbExec {
	// Beware of in-memory DBs: they vanish as soon as the connection closes!
	// We may be able to start a sqlx transaction to avoid problems with that?
	// See: https://github.com/jmoiron/sqlx/issues/164
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	dbe := &dbExec{
		handle: handle,
	}

	return dbe
}
