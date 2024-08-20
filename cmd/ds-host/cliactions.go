package main

import (
	"fmt"
	"os"

	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/database"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacemodel"
)

func migrateData(config *domain.RuntimeConfig) {

	dbManager := &database.Manager{
		Config: config}

	db, err := dbManager.Open()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	migrator := &migrate.Migrator{
		Steps:     migrate.MigrationSteps,
		Config:    config,
		DBManager: dbManager}

	migrateAppspaces, err := migrator.AppspaceMigrationRequired("")
	if err != nil {
		fmt.Println("Error getting migration information. No changes have been made to the data directory.", err.Error())
		os.Exit(1)
	}
	err = migrator.Migrate("")
	if err == migrate.ErrNoMigrationNeeded {
		fmt.Println("Schema matches desired schema, no migration needed")
		os.Exit(0)
	}
	if err != nil {
		fmt.Println("Error Migrating", err.Error())
		os.Exit(1)
	}

	if migrateAppspaces {
		appspaceModel := &appspacemodel.AppspaceModel{
			DB: db}
		appspaceModel.PrepareStatements()
		appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
			Config:        config,
			AppspaceModel: appspaceModel,
		}
		appspaceMetaDb.Init()

		appspaces, err := appspaceModel.GetAll()
		if err != nil {
			fmt.Println("Error getting all appspaces for migration:", err.Error())
			os.Exit(1)
		}
		for _, a := range appspaces {
			err = appspaceMetaDb.OfflineMigrate(a.AppspaceID)
			if err != nil {
				fmt.Printf("Aborting due to error migrating appspace at %v (%v): %v\n", a.LocationKey, a.DomainName, err.Error())
				os.Exit(1)
			}
		}
		fmt.Printf("Migrated %v appspaces\n", len(appspaces))
	}

	sc := dbManager.GetSchema()
	fmt.Println("Migration complete. Schema after migration:", sc)
	os.Exit(0)
}
