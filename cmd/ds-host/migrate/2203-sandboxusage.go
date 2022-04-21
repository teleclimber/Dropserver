package migrate

// sandboxUsageUp adds the table needed to track sandbox resource usage
// This supports tracking and limiting or charging for resource usage.
// columns: owner, app or appspace id,
// start / end date time, sandbox label (for cross-referencing)
// Remember this is purely for accounting purposes
// Performance diagnostics should be left out.
func sandboxUsageUp(args *stepArgs) error {
	args.dbExec(`CREATE TABLE "sandbox_runs" (
		"sandbox_id" INTEGER PRIMARY KEY,
		"instance" TEXT NOT NULL,
		"local_id" INTEGER NOT NULL,
		"owner_id" INTEGER NOT NULL,
		"app_id" INTEGER NOT NULL,
		"version" TEXT NOT NULL,
		"appspace_id" INTEGER,
		"operation" TEXT NOT NULL,
		"cgroup" TEXT NOT NULL,
		"start" DATETIME NOT NULL,
		"end" DATETIME,
		"tied_up_ms" INTEGER NOT NULL DEFAULT 0,
		"cpu_usec" INTEGER NOT NULL DEFAULT 0,
		"memory_byte_sec" INTEGER NOT NULL DEFAULT 0
	)`)
	// sandbox_id is an alias for sqilte's rowid.
	args.dbExec(`CREATE INDEX sandbox_runs_owner ON sandbox_runs ( owner_id )`)
	args.dbExec(`CREATE INDEX sandbox_runs_app ON sandbox_runs ( app_id )`) // not sure we should include version in this index
	args.dbExec(`CREATE INDEX sandbox_runs_appspace ON sandbox_runs ( appspace_id )`)
	args.dbExec(`CREATE INDEX sandbox_runs_start ON sandbox_runs ( start )`)

	return args.dbErr
}

func sandboxUsageDown(args *stepArgs) error {
	args.dbExec(`DROP TABLE "sandbox_runs"`)
	return args.dbErr
}
