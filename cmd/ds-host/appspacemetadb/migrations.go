package appspacemetadb

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

type dbExec struct {
	handle *sqlx.DB
	err    error
}

func (d *dbExec) exec(q string, args ...any) {
	if d.err != nil {
		return
	}

	_, err := d.handle.Exec(q, args...)
	if err != nil {
		d.err = fmt.Errorf("error executing SQL: '%s': %w", q, err)
	}
}

func (d *dbExec) takeErr(err error) {
	if d.err == nil {
		d.err = err
	}
}

func (d *dbExec) checkErr() error {
	if d.err != nil {
		return d.err
	}
	return nil
}

type migrationFn func(*dbExec)

var upMigrations = []migrationFn{migrateUpToV0, migrateUpToV1}
var downMigrations = []migrationFn{migrateDownFromV1} // There is no down migration from 0.

var curSchema = len(upMigrations) - 1

func migrateUpToV0(d *dbExec) {
	// info table. Could be key/value, but could also be single-row table.
	// For now, just holds the current API version and the schema of the appspace files.
	// I think meta db schema could be buried in the special thing that sqlite has for this.
	// App description stuff should not need to be stashed in DB, I think stashing the manifest json is good enough?
	// Or stash the app manifest in the DB, and associate it with a date, so that you can trace history?
	d.exec(`CREATE TABLE info (
		"name" TEXT,
		"value" TEXT
	)`)
	d.exec(`CREATE UNIQUE INDEX info_index ON info (name)`)

	createUsersV0(d)

	// Set appspace meta db schema version using pragma or wahtever. I think sqlite has a field in the DB for that.
	d.exec(`INSERT INTO info (name, value) VALUES("ds-api-version", "0")`)
}

func createUsersV0(d *dbExec) {
	d.exec(`CREATE TABLE "users" (
		"proxy_id" TEXT,
		"auth_type" TEXT,
		"auth_id" TEXT,
		"display_name" TEXT NOT NULL DEFAULT "",
		"avatar" TEXT NOT NULL DEFAULT "",
		"permissions" TEXT NOT NULL DEFAULT "",
		"created" DATETIME,
		"last_seen" DATETIME,
		PRIMARY KEY (proxy_id)
	)`)
	d.exec(`CREATE UNIQUE INDEX appspace_proxy_id ON users (proxy_id)`)
	d.exec(`CREATE UNIQUE INDEX appspace_auth_id ON users (auth_type, auth_id)`)
	// you also can't have two users with the same auth id. Otherwise, upon authenticating, what proxy id do you assign?
}

type AuthsV1 struct {
	ProxyID    string             `db:"proxy_id"`
	Type       string             `db:"type"`
	Identifier string             `db:"identifier"`
	Created    nulltypes.NullTime `db:"created"`
	LastSeen   nulltypes.NullTime `db:"last_seen"`
}

func migrateUpToV1(d *dbExec) {
	// create a new table for auth ids.
	createUserAuthIdsV1(d)

	// move user auths over to new table
	d.exec(`INSERT INTO user_auth_ids (proxy_id, type, identifier, created, last_seen)
		SELECT proxy_id, auth_type, auth_id, created, last_seen FROM users`)

	// rename users to users_old
	d.exec(`ALTER TABLE users RENAME TO users_old`)

	// create new users table
	createUsersV1(d)

	// transfer data to new users table
	d.exec(`INSERT INTO users (proxy_id, display_name, avatar, permissions, created, last_seen)
		SELECT proxy_id, display_name, avatar, permissions, created, last_seen FROM users_old`)

	// Then drop the users_old
	d.exec(`DROP TABLE users_old`)

	// move "ds-api-version" to appspace meta data schema in sqlite pragma thingo.
	d.exec(`PRAGMA user_version = 1`)
	d.exec(`DELETE FROM info WHERE name = "ds-api-version"`)
}

// Table-building is in separate function to support future
// down-migration functions that may need to re-create these.
func createUserAuthIdsV1(d *dbExec) {
	d.exec(`CREATE TABLE "user_auth_ids" (
		"proxy_id" TEXT,
		"type" TEXT,
		"identifier" TEXT,
		"created" DATETIME,
		"last_seen" DATETIME
	)`)
	d.exec(`CREATE INDEX user_auth_ids_proxy ON user_auth_ids (proxy_id)`)
	// auth_type+auth_id is unique in the table: no two users can have the same auth_id,
	// and one user shouldn't have the same twice.
	d.exec(`CREATE UNIQUE INDEX user_auth_ids_auths ON user_auth_ids (type, identifier)`)
}

func createUsersV1(d *dbExec) {
	d.exec(`CREATE TABLE "users" (
		"proxy_id" TEXT,
		"display_name" TEXT NOT NULL DEFAULT "",
		"avatar" TEXT NOT NULL DEFAULT "",
		"permissions" TEXT NOT NULL DEFAULT "",
		"created" DATETIME,
		"last_seen" DATETIME,
		PRIMARY KEY (proxy_id)
	)`)
	d.exec(`CREATE UNIQUE INDEX users_proxy_id ON users (proxy_id)`)
}

func migrateDownFromV1(d *dbExec) {
	// rename to users_old
	d.exec(`ALTER TABLE users RENAME TO users_old`)

	// create v0 users table
	createUsersV0(d)

	// move data to users
	d.exec(`INSERT INTO users (proxy_id, display_name, avatar, permissions, created, last_seen)
		SELECT proxy_id, display_name, avatar, permissions, created, last_seen FROM users_old`)

	d.exec(`DROP TABLE users_old`)

	auths := []AuthsV1{}
	d.takeErr(d.handle.Select(&auths, `SELECT * FROM user_auth_ids ORDER BY created DESC`))
	q, err := d.handle.Preparex(`UPDATE users SET auth_type = ?, auth_id = ? WHERE proxy_id = ?`)
	d.takeErr(err)
	for _, a := range auths {
		// "email" and "dropid" were the auth types used in v0, so only preserve those
		if a.Type == "email" || a.Type == "dropid" {
			_, err = q.Exec(a.Type, a.Identifier, a.ProxyID)
			d.takeErr(err)
		}
	}

	// then drop user_auth_ids
	d.exec(`DROP TABLE user_auth_ids`)

	d.exec(`PRAGMA user_version = 0`)
	d.exec(`INSERT INTO info (name, value) VALUES("ds-api-version", "0")`)
}
