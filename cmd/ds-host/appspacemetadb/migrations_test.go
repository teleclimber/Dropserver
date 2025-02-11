package appspacemetadb

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/internal/nulltypes"
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

func TestMigrateUpToV1Blank(t *testing.T) {
	dbe := getTestDBExec()

	migrateUpToV0(dbe)
	migrateUpToV1(dbe)

	err := dbe.checkErr()
	if err != nil {
		t.Error(err)
	}
}

type SqliteSchemaRow struct {
	Type     string         `db:"type"`
	Name     string         `db:"name"`
	TblName  string         `db:"tbl_name"`
	RootPage int            `db:"rootpage"`
	Sql      sql.NullString `db:"sql"`
}

func TestMigrateUpToV1(t *testing.T) {
	dbe := getTestDBExec()
	migrateUpToV0(dbe)

	dbe.exec(`INSERT INTO users (proxy_id, auth_type, auth_id, created) 
		VALUES ("abc", "email", "a@b.c", datetime("now")),
		("def", "email", "d@e.f", datetime("now"))`)

	migrateUpToV1(dbe)
	err := dbe.checkErr()
	if err != nil {
		t.Error(err)
	}

	auths := []AuthsV1{}
	dbe.handle.Select(&auths, `SELECT * FROM user_auth_ids ORDER BY proxy_id`)
	t.Log(auths)
	if len(auths) != 2 {
		t.Error("expected 2 auth rows")
	}
	if !auths[0].Created.Valid {
		t.Error("expected created to be non-null")
	}
	expected := []AuthsV1{
		{"abc", "email", "a@b.c", "", auths[0].Created},
		{"def", "email", "d@e.f", "", auths[1].Created},
	}
	if !cmp.Equal(auths, expected) {
		t.Error(cmp.Diff(auths, expected))
	}

	err = dbe.checkErr()
	if err != nil {
		t.Error(err)
	}
}

type UserRowV0 struct {
	ProxyID     string             `db:"proxy_id"`
	AuthType    string             `db:"auth_type"`
	AuthID      string             `db:"auth_id"`
	DisplayName string             `db:"display_name"`
	Avatar      string             `db:"avatar"`
	Permissions string             `db:"permissions"`
	Created     time.Time          `db:"created"`
	LastSeen    nulltypes.NullTime `db:"last_seen"`
}

func TestMigrateDownFromV1(t *testing.T) {
	dbe := getTestDBExec()
	migrateUpToV0(dbe)

	dbe.exec(`INSERT INTO users (proxy_id, auth_type, auth_id, created) 
		VALUES ("abc", "email", "a@b.c", datetime("now")),
		("def", "dropid", "d@e.f", datetime("now"))`)

	startSchema := getSqliteSchema(t, dbe.handle)
	startUsers := getUsersV0(t, dbe.handle)

	migrateUpToV1(dbe)

	// Add another auth with a later creation date to make sure the down-migration keeps the earlier one:
	dbe.exec(`INSERT INTO user_auth_ids (proxy_id, type, identifier, extra_name, created)
		VALUES ("abc", "dropid", "new.dropid", "", datetime("now", "+1 day"))`)

	migrateDownFromV1(dbe)

	err := dbe.checkErr()
	if err != nil {
		t.Error(err)
	}

	endSchema := getSqliteSchema(t, dbe.handle)
	endUsers := getUsersV0(t, dbe.handle)

	if !cmp.Equal(startSchema, endSchema) {
		t.Error(cmp.Diff(startSchema, endSchema))
	}
	if !cmp.Equal(startUsers, endUsers) {
		t.Error(cmp.Diff(startUsers, endUsers))
	}
}

func getSqliteSchema(t *testing.T, handle *sqlx.DB) (rows []SqliteSchemaRow) {
	err := handle.Select(&rows, `SELECT * FROM sqlite_schema ORDER BY name`)
	if err != nil {
		t.Error(err)
	}
	return
}
func getUsersV0(t *testing.T, handle *sqlx.DB) (rows []UserRowV0) {
	err := handle.Select(&rows, `SELECT * FROM users ORDER BY proxy_id`)
	if err != nil {
		t.Error(err)
	}
	return
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
