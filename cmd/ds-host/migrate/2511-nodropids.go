package migrate

func removeDropIDsUp(args *stepArgs) error {
	args.dbExec(`ALTER TABLE appspaces DROP COLUMN dropid`)

	args.dbExec(`DROP TABLE remote_appspaces`)

	// add display name and image to users table.
	args.dbExec(`ALTER TABLE "users" ADD COLUMN display_name TEXT NOT NULL DEFAULT ""`)
	args.dbExec(`ALTER TABLE "users" ADD COLUMN display_image TEXT NOT NULL DEFAULT ""`)

	// Add table that associates an instance user with an appspace user
	args.dbExec(`CREATE TABLE "instance_appspace_users" (
		"user_id" INTEGER NOT NULL,	
		"appspace_id" INTEGER NOT NULL,
		"proxy_id" TEXT NOT NULL
	)`)

	args.dbExec(`CREATE UNIQUE INDEX instance_appspace_ids ON instance_appspace_users (user_id, appspace_id)`)
	args.dbExec(`CREATE UNIQUE INDEX instance_appspace_all_cols ON instance_appspace_users (user_id, appspace_id, proxy_id)`)

	// add single-column indexes for faster lookups
	args.dbExec(`CREATE INDEX instance_appspace_user_id_idx ON instance_appspace_users (user_id)`)
	args.dbExec(`CREATE INDEX instance_appspace_appspace_id_idx ON instance_appspace_users (appspace_id)`)

	return args.dbErr
}

func removeDropIDsDown(args *stepArgs) error {
	args.dbExec(`ALTER TABLE appspaces ADD COLUMN "dropid" TEXT`)

	// recreate remote appspaces columns.
	// Data is lost, but at least the code won't error due to missingtable.
	args.dbExec(`CREATE TABLE "remote_appspaces" (
		"user_id" INTEGER NOT NULL,
		"domain_name" TEXT NOT NULL,
		"owner_dropid" TEXT,
		"dropid" TEXT,
		"created" DATETIME,
		PRIMARY KEY (user_id, domain_name)
	)`)
	args.dbExec(`CREATE INDEX remote_user_id ON remote_appspaces (user_id)`)

	// drop single-column indexes first
	args.dbExec(`DROP INDEX instance_appspace_user_id_idx`)
	args.dbExec(`DROP INDEX instance_appspace_appspace_id_idx`)

	// drop the composite unique indexes and table
	args.dbExec(`DROP INDEX instance_appspace_all_cols`)
	args.dbExec(`DROP INDEX instance_appspace_ids`)
	args.dbExec(`DROP TABLE "instance_appspace_users"`)

	return args.dbErr
}
