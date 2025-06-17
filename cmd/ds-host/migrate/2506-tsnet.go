package migrate

import "errors"

func tsnetIntegrationUp(args *stepArgs) error {

	// add tsnet columns to user table:
	args.dbExec(`ALTER TABLE "users" ADD COLUMN tsnet_identifier TEXT`)
	args.dbExec(`ALTER TABLE "users" ADD COLUMN tsnet_extra_name TEXT`)
	args.dbExec(`CREATE UNIQUE INDEX users_tsnet_identifier ON users (tsnet_identifier)`)

	// add tsnet columns to instance settings table
	args.dbExec(`ALTER TABLE "settings" ADD COLUMN tsnet_control_url TEXT NOT NULL DEFAULT ''`)
	args.dbExec(`ALTER TABLE "settings" ADD COLUMN tsnet_hostname TEXT NOT NULL DEFAULT ''`)
	args.dbExec(`ALTER TABLE "settings" ADD COLUMN tsnet_connect INTEGER NOT NULL DEFAULT 0`)

	// add table for appspace tsnet settings
	args.dbExec(`CREATE TABLE "appspace_tsnet" (
		"appspace_id" INTEGER PRIMARY KEY,
		"control_url" TEXT NOT NULL,
		"hostname" TEXT NOT NULL,
		"connect" INTEGER NOT NULL
	)`)

	return args.dbErr
}

type getCount struct {
	Count int `db:"count"`
}

func tsnetIntegrationDown(args *stepArgs) error {
	var c getCount
	args.db.Handle.Get(&c, "SELECT COUNT(*) as count FROM users WHERE email IS NULL OR password IS NULL")
	if c.Count != 0 {
		return errors.New("to downgrade from 2506-tsnet all users must have an email and password")
	}

	args.dbExec(`DROP INDEX users_tsnet_identifier`)
	args.dbExec(`ALTER TABLE users DROP COLUMN tsnet_identifier`)
	args.dbExec(`ALTER TABLE users DROP COLUMN tsnet_extra_name`)

	args.dbExec(`ALTER TABLE "settings" DROP COLUMN tsnet_control_url`)
	args.dbExec(`ALTER TABLE "settings" DROP COLUMN tsnet_hostname`)
	args.dbExec(`ALTER TABLE "settings" DROP COLUMN tsnet_connect`)

	args.dbExec(`DROP TABLE appspace_tsnet`)

	return args.dbErr
}
