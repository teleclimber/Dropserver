package migrate

func tailscaleIntegrationUp(args *stepArgs) error {

	// admin / instance prefs:
	//	allow-funnel: Yes/no for tailscale funnel. applies to all ts configs and servers wihtout override.
	//					It's about exposing the instance to the public internet or not.

	// args.dbExec(`ALTER TABLE "settings" ADD COLUMN allow_funnel INTEGER`) // TODO implement in code

	args.dbExec(`CREATE TABLE "appspace_tsnet" (
		"appspace_id" INTEGER PRIMARY KEY,
		"backend_url" TEXT NOT NULL,
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

func tailscaleIntegrationDown(args *stepArgs) error {

	// TODO drop table appspace_tsnet

	return args.dbErr
}
