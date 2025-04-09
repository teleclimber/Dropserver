package migrate

func tsnetIntegrationUp(args *stepArgs) error {

	// alter users table to allow null email and password:
	args.dbExec(`DROP INDEX user_emails`)
	args.dbExec(`ALTER TABLE "users" ADD COLUMN email_new TEXT`)
	args.dbExec(`UPDATE users SET email_new = email`)
	args.dbExec(`ALTER TABLE "users" DROP COLUMN email`)
	args.dbExec(`ALTER TABLE "users" RENAME COLUMN email_new TO email`)
	args.dbExec(`CREATE UNIQUE INDEX user_emails ON users (email)`)

	args.dbExec(`ALTER TABLE "users" ADD COLUMN password_new TEXT`)
	args.dbExec(`UPDATE users SET password_new = password`)
	args.dbExec(`ALTER TABLE "users" DROP COLUMN password`)
	args.dbExec(`ALTER TABLE "users" RENAME COLUMN password_new TO password`)

	// add tsnet columns to user table:
	args.dbExec(`ALTER TABLE "users" ADD COLUMN tsnet_identifier TEXT`)
	args.dbExec(`ALTER TABLE "users" ADD COLUMN tsnet_extra_name TEXT`)
	args.dbExec(`CREATE UNIQUE INDEX users_tsnet_identifier ON users (tsnet_identifier)`)

	args.dbExec(`ALTER TABLE "settings" ADD COLUMN tsnet_control_url TEXT NOT NULL DEFAULT ''`)
	args.dbExec(`ALTER TABLE "settings" ADD COLUMN tsnet_hostname TEXT NOT NULL DEFAULT ''`)
	args.dbExec(`ALTER TABLE "settings" ADD COLUMN tsnet_connect INTEGER NOT NULL DEFAULT 0`)

	args.dbExec(`CREATE TABLE "appspace_tsnet" (
		"appspace_id" INTEGER PRIMARY KEY,
		"control_url" TEXT NOT NULL,
		"hostname" TEXT NOT NULL,
		"connect" INTEGER NOT NULL
	)`)
	// We also select by connect, so maybe an index on that
	// add "auto-add-users" column
	// add use-funnel col to express desire to expose appspace to internet

	///////////////////////
	// ALSO consider deleting the denosandbox code files that are at the OG location in the data-dir, for cleanliness?
	// BUT we don't have the necessary function injected here.

	return args.dbErr
}

func tsnetIntegrationDown(args *stepArgs) error {

	// after removing tsnet stuff from users,
	// it's possible you have some users with no auth.
	// It's also possible the admin no longer has an email/pw
	// Make it possible to recover from that.
	// Maybe check that one admin has and email/pw before downgrading?

	// TODO drop table appspace_tsnet

	return args.dbErr
}
