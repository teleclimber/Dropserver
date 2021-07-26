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

	h.exec(`CREATE TABLE "users" (
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
	h.exec(`CREATE UNIQUE INDEX appspace_proxy_id ON users (proxy_id)`)
	h.exec(`CREATE UNIQUE INDEX appspace_auth_id ON users (auth_type, auth_id)`)
	// you also can't have two users with the same auth id. Otherwise, upon authenticating, what proxy id do you assign?
	// Some more posible columns:
	// - self-reg versus invited
	// - self-reg status
	// - block

	// Do we need a "block" flag? We'd need it on appspaces (kind of like a "pause" but for a user)
	// Also would need a block flag at the contact level, which blocks contact from all appspaces.
	// The per-appspace block would be in the appspace meta data itself, so that non-contacts can be blocked.

	// routes is not currently used! May come back as appspace routes, but for now it stays empty
	// h.exec(`CREATE TABLE routes (
	// 	"methods" INT,
	// 	"path" TEXT,
	// 	"auth" TEXT,
	// 	"handler" TEXT
	// )`)
	// h.exec(`CREATE UNIQUE INDEX routes_index ON routes (methods, url)`)
	//^^ no that's not the index. We should def index url though
	//h.exec(`CREATE INDEX routes_path_index ON routes (path)`)
	// Do we need a inherent order? I think we might (essentially number of url path elements, and it should be inherent weight)
	// presume auth and handler are json.

	// Set schema version using pragma or wahtever. I think sqlite has a field in the DB for that.
	h.exec(`INSERT INTO info (name, value) VALUES("ds-api-version", "0")`)
}

func (h *v0handle) migrateDownToV0() {
	// I mean really, this one means deleting the DB file.

	// h.exec(`DROP TABLE routes`)

	// Set schema version using pragma or wahtever. I think sqlite has a field in the DB for that.
}
