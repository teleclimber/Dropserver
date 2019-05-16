package migrate

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

var freshInstall = migrationStep{
	up:   freshInstallUp,
	down: freshInstallDown}

// freshInstallUp means full instalation.
// This means creating a DB at the very least and creating its schema
// It may also mean things for sandboxes and ds-trusted, but we'll get to that some other time.
func freshInstallUp(args *stepArgs) domain.Error {

	args.dbExec(`CREATE TABLE "params" ( "name" TEXT, "value" TEXT )`)
	args.dbExec(`INSERT INTO "params" (name, value) VALUES("db_schema", "")`)

	//... skipping a bunch of tables for the moment

	args.dbExec(`CREATE TABLE "apps" ("name" TEXT, "location_key" TEXT)`)

	if args.dbErr != nil {
		return dserror.FromStandard(args.dbErr)
	}
	// the other option is to just check args for errors in the caller Migrate function

	return nil
}

func freshInstallDown(args *stepArgs) domain.Error {
	// This is effectively uninstall but I don't want to implement, at least for now.
	return dserror.New(dserror.MigrateDownNotSupported, "can not go down from fresh install")
}
