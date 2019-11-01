package migrate

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

var freshInstall = migrationStep{
	up:   freshInstallUp,
	down: freshInstallDown}

// freshInstallUp means full instalation.
// This means creating a DB at the very least and creating its schema
// It may also mean things for sandboxes and ds-trusted, but we'll get to that some other time.
func freshInstallUp(args *stepArgs) domain.Error {

	args.dbExec(`CREATE TABLE "params" ( "name" TEXT, "value" TEXT )`)
	args.dbExec(`INSERT INTO "params" (name, value) VALUES("db_schema", "")`)

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
		"appspace_id" INTEGER
	)`)
	args.dbExec(`CREATE UNIQUE INDEX cookies_cookie_id ON cookies (cookie_id)`)
	// could index on user_id and appspace_id too

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
		"location_key" TEXT,
		"created" DATETIME
	)`)
	args.dbExec(`CREATE UNIQUE INDEX app_id_versions ON app_versions (app_id, version)`)

	// appspaces:
	args.dbExec(`CREATE TABLE "appspaces" (
		"appspace_id" INTEGER PRIMARY KEY ASC,
		"owner_id" INTEGER,
		"app_id" INTEGER,
		"app_version" TEXT,
		"subdomain" TEXT,
		"paused" INTEGER DEFAULT 0,
		"location_key" TEXT,
		"created" DATETIME
	)`)
	args.dbExec(`CREATE UNIQUE INDEX appspace_subdomain ON appspaces (subdomain)`)
	// probably index owner_id. and maybe app_id?
	// should put a unique key constraint on location key?

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
		return dserror.FromStandard(args.dbErr)
	}
	// the other option is to just check args for errors in the caller Migrate function

	return nil
}

func freshInstallDown(args *stepArgs) domain.Error {
	// This is effectively uninstall but I don't want to implement, at least for now.
	return dserror.New(dserror.MigrateDownNotSupported, "can not go down from fresh install")
}
