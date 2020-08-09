package appspacemetadb

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestV0Exec(t *testing.T) {
	v0h := &v0handle{handle: getTestDBHandle()}

	v0h.exec(`CREATE TABLE info (
		"name" TEXT,
		"value" TEXT
	)`)

	if v0h.err != nil {
		t.Error(v0h.err)
	}

	//check the table was actually created please.
	var tables []string
	err := v0h.handle.Select(&tables, `SELECT name FROM sqlite_master WHERE type ='table' AND name NOT LIKE 'sqlite_%'`)
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

func TestV0ExecError(t *testing.T) {
	v0h := &v0handle{handle: getTestDBHandle()}

	v0h.exec(`CREATEzzzzz TABLE info (
		"name" TEXT,
		"value" TEXT
	)`)

	err := v0h.checkErr()
	if err == nil {
		t.Error("expected error")
	}
}

func TestMigrateUpToV0(t *testing.T) {
	v0h := &v0handle{handle: getTestDBHandle()}

	v0h.migrateUpToV0()

	err := v0h.checkErr()
	if err != nil {
		t.Error(err)
	}
}

func getTestDBHandle() *sqlx.DB {
	// Beware of in-memory DBs: they vanish as soon as the connection closes!
	// We may be able to start a sqlx transaction to avoid problems with that?
	// See: https://github.com/jmoiron/sqlx/issues/164
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	return handle
}
