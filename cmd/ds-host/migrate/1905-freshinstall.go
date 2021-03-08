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
		"proxy_id" TEXT
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

	// then add auth tables
	// CREATE TABLE contact_ds_auth (
	//	"contact_id" INTEGER NOT NULL,
	// 	"username" TEXT,
	// 	"url" TEXT,
	// 	"token" TEXT,
	// )
	// plus other things like datetime established,
	// Whether contact is 2-way..

	// Other tables: contact_email_auth,

	// appspace_users linkes contacts (or owner) to proxy ids.
	args.dbExec(`CREATE TABLE "appspace_contacts" (
		"appspace_id" INTEGER NOT NULL,
		"contact_id" INTEGER,
		"proxy_id" TEXT
	)`)
	args.dbExec(`CREATE UNIQUE INDEX appspace_proxy_id ON appspace_contacts (appspace_id, proxy_id)`)
	args.dbExec(`CREATE INDEX user_contact_id ON appspace_contacts (contact_id)`)

	// Do we need a "block" flag? We'd need it on appspaces (kind of like a "pause" but for a user)
	// Also would need a block flag at the contact level, which blocks contact from all appspaces.
	// The per-appspace block would be in the appspace meta data itself, so that non-contacts can be blocked.

	// We may need to separatecontact urls and auth stuff into separate tables to enable multiple ways for contacts to log in, etc..
	// Also need a "mutual_contacts" or something like that? For when a contact is an actual user's contact.

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
		return dserror.FromStandard(args.dbErr)
	}
	// the other option is to just check args for errors in the caller Migrate function

	return nil
}

func freshInstallDown(args *stepArgs) domain.Error {
	// This is effectively uninstall but I don't want to implement, at least for now.
	return dserror.New(dserror.MigrateDownNotSupported, "can not go down from fresh install")
}
