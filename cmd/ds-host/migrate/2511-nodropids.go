package migrate

func removeDropIDsUp(args *stepArgs) error {
	args.dbExec(`ALTER TABLE appspaces DROP COLUMN dropid`)

	return args.dbErr
}

func removeDropIDsDown(args *stepArgs) error {
	args.dbExec(`ALTER TABLE appspaces ADD COLUMN "dropid" TEXT`)

	return args.dbErr
}
