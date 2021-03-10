package appspacemetadb

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type v0handle struct {
	handle *sqlx.DB
	err    error
}

func (h *v0handle) exec(q string) {
	if h.err != nil {
		return
	}

	_, err := h.handle.Exec(q)
	if err != nil {
		h.err = err
	}
}

func (h *v0handle) checkErr() error {
	if h.err != nil {
		return h.err
	}
	return nil
}

func (h *v0handle) migrateUpToV0() {
	// info table. Could be key/value, but could also be single-row table.
	// For now, just holds the current API version, but probably other things later too,
	// like app info (detailed, with url, etc.. so it can be found again),
	h.exec(`CREATE TABLE info (
		"name" TEXT,
		"value" TEXT
	)`)
	h.exec(`CREATE UNIQUE INDEX info_index ON info (name)`)

	h.exec(`CREATE TABLE routes (
		"methods" INT,
		"path" TEXT,
		"auth" TEXT,
		"handler" TEXT
	)`)
	// h.exec(`CREATE UNIQUE INDEX routes_index ON routes (methods, url)`)
	//^^ no that's not the index. We should def index url though
	h.exec(`CREATE INDEX routes_path_index ON routes (path)`)
	// Do we need a inherent order? I think we might (essentially number of url path elements, and it should be inherent weight)
	// presume auth and handler are json.

	h.exec(`CREATE TABLE users (
		"proxy_id" TEXT,
		"display_name" TEXT,
		"permissions" TEXT
	)`)
	h.exec(`CREATE UNIQUE INDEX users_proxy_id ON users (proxy_id)`)
	// Should we have an is_owner here? Might simplify some things:
	// - app does getUser, it knows right away if user is owner.
	// - in ds-dev, is_owner is readily availble, no need to set it (since appspace is divorced from host)
	// - it might mean that the appspace contacts table need not have the owner  in it? Though maybe that makes things mre complicated

	// possibility of adding a users_identification table or something
	// to store registration credentials for self-registration

	// Set schema version using pragma or wahtever. I think sqlite has a field in the DB for that.
	h.exec(`INSERT INTO info (name, value) VALUES("ds-api-version", "0")`)
}

func (h *v0handle) migrateDownToV0() {
	// I mean really, this one means deleting the DB file.

	// h.exec(`DROP TABLE routes`)

	// Set schema version using pragma or wahtever. I think sqlite has a field in the DB for that.
}
