package migrate

func tailscaleIntegrationUp(args *stepArgs) error {
	// TODO tbd

	// ALSO consider deleting the denosandbox code files that are at the OG location in the data-dir, for cleanliness?
	// BUT we don't have the necessary function injected here.

	return args.dbErr
}

func tailscaleIntegrationDown(args *stepArgs) error {

	return args.dbErr
}
