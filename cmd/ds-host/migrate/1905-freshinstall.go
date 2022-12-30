package migrate

import (
	"errors"
	"math/rand"
	"time"
)

// freshInstallUp means full instalation.
func freshInstallUp(args *stepArgs) error {

	args.dbExec(`CREATE TABLE "params" ( "name" TEXT, "value" TEXT )`)
	args.dbExec(`INSERT INTO "params" (name, value) VALUES("db_schema", "")`)
	args.dbExec(`INSERT INTO "params" (name, value) VALUES("setup_key", ?)`, randomKey())

	args.dbExec(`CREATE TABLE "settings" (
		"id" INTEGER PRIMARY KEY CHECK (id = 1),
		"registration_open" INTEGER
	)`)

	// here we're forced to create a row with some values. This is some sort of ad-hoc defaults. But OK.
	args.dbExec(`INSERT INTO "settings" (id, registration_open) VALUES (1, 0)`)

	args.dbExec(`CREATE TABLE "users" (
		"user_id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"email" TEXT,
		"password" TEXT
	)`)
	args.dbExec(`CREATE UNIQUE INDEX user_emails ON users (email)`)

	args.dbExec(`CREATE TABLE "dropids" (
		"user_id" INTEGER,
		"handle" TEXT,
		"domain" TEXT,
		"display_name" TEXT,
		"created" DATETIME
	)`)
	args.dbExec(`CREATE INDEX dropids_users ON dropids (user_id)`)
	args.dbExec(`CREATE UNIQUE INDEX dropids_handle_domains ON dropids (handle, domain)`)

	args.dbExec(`CREATE TABLE "admin_users" (
		"user_id" INTEGER
	)`)
	args.dbExec(`CREATE UNIQUE INDEX admin_user_ids ON admin_users (user_id)`)

	args.dbExec(`CREATE TABLE "user_invitations" (
		"email"	TEXT UNIQUE ON CONFLICT IGNORE
	)`)
	args.dbExec(`CREATE INDEX emails ON user_invitations ( email )`)

	args.dbExec(`CREATE TABLE cookies (
		"cookie_id" TEXT,
		"user_id" INTEGER,
		"expires" DATETIME,
		"user_account" INTEGER,
		"appspace_id" INTEGER,
		"proxy_id" TEXT,
		"domain" TEXT
	)`)
	args.dbExec(`CREATE UNIQUE INDEX cookies_cookie_id ON cookies (cookie_id)`)
	// could index on user_id and appspace_id too
	// Might need two separate cookie tables: one for admin and one for appspaces?
	// What is meaning of user_account?

	args.dbExec(`CREATE TABLE "apps" (
		"owner_id" INTEGER,
		"app_id" INTEGER PRIMARY KEY ASC,
		"name" TEXT,
		"created" DATETIME
	)`)
	// probably need to index owner-id
	// TODO: use autoincrement on all *-id to prevent id reuse from deleted rows

	args.dbExec(`CREATE TABLE "app_versions" (
		"app_id" INTEGER,
		"version" TEXT,
		"schema" INTEGER,
		"api" INTEGER,
		"location_key" TEXT,
		"created" DATETIME
	)`)
	args.dbExec(`CREATE UNIQUE INDEX app_id_versions ON app_versions (app_id, version)`)

	// appspaces:
	args.dbExec(`CREATE TABLE "appspaces" (
		"appspace_id" INTEGER PRIMARY KEY ASC,
		"owner_id" INTEGER,
		"dropid" TEXT,
		"app_id" INTEGER,
		"app_version" TEXT,
		"domain_name" TEXT,
		"paused" INTEGER DEFAULT 0,
		"location_key" TEXT,
		"created" DATETIME
	)`)
	args.dbExec(`CREATE UNIQUE INDEX appspace_domain ON appspaces (domain_name)`)
	// probably index owner_id. and maybe app_id?
	// should put a unique key constraint on location key?
	// probably index dropid_handle and domain as well.

	// remote appspaces are identified by their domain name alone
	// dropid is the id of the local user to use with that remote appspace
	args.dbExec(`CREATE TABLE "remote_appspaces" (
		"user_id" INTEGER NOT NULL,
		"domain_name" TEXT NOT NULL,
		"owner_dropid" TEXT,
		"dropid" TEXT,
		"created" DATETIME,
		PRIMARY KEY (user_id, domain_name)
	)`)
	args.dbExec(`CREATE INDEX remote_user_id ON remote_appspaces (user_id)`)
	//args.dbExec(`CREATE UNIQUE INDEX remote_appspace_domain ON remote_appspaces (domain_name)`)
	// can't help but imagine we'll need a lot more here, but for now this will do.
	// HMM, we have owner dropid in this table, but it seems that it will not be known until after an interaction
	// with said appspace.

	// contacts added by the user:
	args.dbExec(`CREATE TABLE "contacts" (
		"user_id" INTEGER NOT NULL,
		"contact_id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"name" TEXT,
		"display_name" TEXT,
		"created" DATETIME
	)`)
	args.dbExec(`CREATE INDEX contact_user_id ON contacts (user_id)`)
	// Might need a "block" flag and other controls?

	// migration jobs
	args.dbExec(`CREATE TABLE "migrationjobs" (
		"job_id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"owner_id" INTEGER NOT NULL,
		"appspace_id" INTEGER NOT NULL,
		"to_version" TEXT NOT NULL,
		"priority" INTEGER NOT NULL,
		"created" DATETIME NOT NULL,
		"started" DATETIME,
		"finished" DATETIME,
		"error" TEXT
	)`)
	// args.dbExec(`CREATE UNIQUE INDEX migrate_appspace ON migrationjobs (appspace_id)`)
	// ^^ enforce pending job uniqueness some other way.
	// Probably still need an index that helps select pending jobs
	// Also, need job key or some unique identifier? could use rowid??

	if args.dbErr != nil {
		return args.dbErr
	}
	// the other option is to just check args for errors in the caller Migrate function

	return nil
}

func freshInstallDown(args *stepArgs) error {
	// This is effectively uninstall but I don't want to implement, at least for now.
	return errors.New("can not go down from fresh install")
}

// random string
const chars36 = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand2 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomKey() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars36[seededRand2.Intn(len(chars36))]
	}
	return string(b)
}
