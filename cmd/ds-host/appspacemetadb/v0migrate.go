package appspacemetadb

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type dbExec struct {
	handle *sqlx.DB
	err    error
}

func (d *dbExec) exec(q string) {
	if d.err != nil {
		return
	}

	_, err := d.handle.Exec(q)
	if err != nil {
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

var upMigrations = []migrationFn{migrateUpToV0}
var downMigrations = []migrationFn{} // There is no down migration from 0.

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

	// Set appspace meta db schema version using pragma or wahtever. I think sqlite has a field in the DB for that.
	d.exec(`INSERT INTO info (name, value) VALUES("ds-api-version", "0")`)
}
