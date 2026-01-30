package migrate

func removeDropIDsUp(args *stepArgs) error {
	args.dbExec(`ALTER TABLE appspaces DROP COLUMN dropid`)

	args.dbExec(`DROP TABLE remote_appspaces`)

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

	return args.dbErr
}
