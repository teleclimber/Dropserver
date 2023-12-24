package migrate

func appsFromURLsUp(args *stepArgs) error {
	args.dbExec(`CREATE TABLE "app_urls" (
		"app_id" INTEGER PRIMARY KEY,
		"url" TEXT NOT NULL,
		"automatic" INTEGER NOT NULL,
		"last_dt" DATETIME,
		"last_result" TEXT,
		"new_url" TEXT,
		"new_url_dt" DATETIME,
		"listing" JSON,
		"listing_dt" DATETIME,
		"etag" TEXT,
		"latest_version" TEXT
	)`)
	// create an idex that facilitaes fetching the listings that need refreshing:
	args.dbExec(`CREATE UNIQUE INDEX app_urls_auto ON app_urls (automatic, last_dt)`)

	return args.dbErr
}

func appsFromURLsDown(args *stepArgs) error {

	args.dbExec(`ALTER TABLE apps DROP COLUMN url`)

	return args.dbErr
}
