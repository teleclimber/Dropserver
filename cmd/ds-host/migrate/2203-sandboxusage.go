package migrate

// sandboxUsageUp adds the table needed to track sandbox resource usage
// This supports tracking and limiting or charging for resource usage.
// columns: owner, app or appspace id,
// start / end date time, sandbox label (for cross-referencing)
// cpu seconds, memory.high
// Remember this is purely for accounting purposes
// Performance diagnostics should be left out.
func sandboxUsageUp(args *stepArgs) error {
	args.dbExec(`CREATE TABLE "sandbox_runs" (
		"sandbox_id" TEXT NOT NULL,
		"owner_id" INTEGER NOT NULL,
		"app_id" INTEGER NOT NULL,
		"version" TEXT NOT NULL,
		"appspace_id" INTEGER,
		"operation" TEXT NOT NULL,
		"cgroup" TEXT NOT NULL,
		"start" DATETIME NOT NULL,
		"end" DATETIME,
		"cpu_seconds" REAL NOT NULL DEFAULT 0,
		"memory" INTEGER NOT NULL DEFAULT 0
	)`)
	// indices? owner id, appspace_id, app_id, start
	args.dbExec(`CREATE UNIQUE INDEX sandbox_runs_id ON sandbox_runs ( sandbox_id )`)
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
