package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/otiai10/copy"

	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspaceroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacestatus"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/internal/validator"
)

// Some lifecycle sequences:
// For migrations:
// - stop everything, wait until status relflects all stopped and closed
// - copy appspace files (really? not always. only if a reset is desired.)
// - run migration

// For resetting appspace files:
// - stop everything (including meta db files, and close them), wait for status to reflect that
// - copy appspace files

// For entering Migration mode
// - everything should be stopped, but this is as built in migration runner/whatever.

// Detect Schema mismatch:
// - reflect "stopping -- switching to migration mode" in UI
// - stop the appspace completely
// - wait until fully stopped
// - enter "migration" mode in UI: show migrate buttons, hide route hits etc...

var appDirFlag = flag.String("app", "", "specify root directory of app code")
var appspaceDirFlag = flag.String("appspace", "", "specify root directory of appspace data")

var execPathFlag = flag.String("exec-path", "", "specify where the exec path is so resources can be loaded")

const ownerID = domain.UserID(7)
const appID = domain.AppID(11)
const appspaceID = domain.AppspaceID(15)

func main() {

	m := record.Metrics{}

	validator := &validator.Validator{}
	validator.Init()

	flag.Parse()

	if *appDirFlag == "" {
		fmt.Println("Please specify app dir")
		os.Exit(1)
	}
	if *appspaceDirFlag == "" {
		fmt.Println("Please specify appspace dir")
		os.Exit(1)
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	appspaceWorkingDir := filepath.Join(tempDir, "appspace")
	err = os.MkdirAll(appspaceWorkingDir, 0744)
	if err != nil {
		panic(err)
	}

	// events:
	appspacePausedEvents := &events.AppspacePausedEvents{}
	appspaceStatusEvents := &events.AppspaceStatusEvents{}

	runtimeConfig := GetConfig(*execPathFlag, *appDirFlag, appspaceWorkingDir)

	appFilesModel := appfilesmodel.AppFilesModel{
		Config: runtimeConfig,
	}

	appFilesMeta, dsErr := appFilesModel.ReadMeta("")
	if dsErr != nil {
		fmt.Println("Failed to read app metadata: " + dsErr.PublicString())
	}

	// Copy appspace files
	err = copy.Copy(*appspaceDirFlag, appspaceWorkingDir)
	if err != nil {
		panic(err)
	}

	// Now read appspace metadata.
	devAppspaceModel := &DevAppspaceModel{
		AsPausedEvent: appspacePausedEvents}

	appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
		AppspaceModel: devAppspaceModel,
		Config:        runtimeConfig,
		Validator:     validator}
	appspaceMetaDb.Init()

	appspaceInfoModels := &appspacemetadb.AppspaceInfoModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb}
	appspaceInfoModels.Init()

	appspaceRouteModels := &appspacemetadb.AppspaceRouteModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb,
		Validator:      validator}
	appspaceRouteModels.Init()

	appspaceSchema, err := appspaceInfoModels.GetSchema(appspaceID)
	if err != nil {
		fmt.Println("failed to get appspace schema: " + err.Error())
	}

	if appspaceSchema != appFilesMeta.SchemaVersion {
		// TODO: here we are in some sort of migration mode.
		fmt.Printf("Schema mismatch: app: %v <> appspace: %v \n", appFilesMeta.SchemaVersion, appspaceSchema)
	}

	devAuth := &DevAuthenticator{}
	devAuth.Set(domain.Authentication{
		HasUserID:  true,
		UserID:     ownerID,
		AppspaceID: appspaceID,
	})
	devSandboxManager := &DevSandboxManager{}

	devMigrationJobModel := &DevMigrationJobModel{}

	devAppModel := &DevAppModel{}
	devAppModel.App = domain.App{
		OwnerID: ownerID,
		AppID:   appID,
		Created: time.Now(),
		Name:    appFilesMeta.AppName}
	devAppModel.Ver = domain.AppVersion{
		AppID:       appID,
		AppName:     appFilesMeta.AppName,
		Version:     appFilesMeta.AppVersion,
		Schema:      appFilesMeta.SchemaVersion,
		Created:     time.Now(),
		LocationKey: ""}

	devAppspaceModel.Appspace = domain.Appspace{
		OwnerID:     ownerID,
		AppspaceID:  appspaceID,
		AppID:       appID,
		AppVersion:  appFilesMeta.AppVersion, // assume it's the all we are working on.
		Subdomain:   "",
		Created:     time.Now(),
		LocationKey: "",
		Paused:      false}

	migrateJobController := &migrateappspace.JobController{
		MigrationJobModel:  devMigrationJobModel,
		AppModel:           devAppModel,
		AppspaceInfoModels: appspaceInfoModels,
		AppspaceModel:      devAppspaceModel,
		AppspaceStatus:     nil, //set below
		SandboxMaker:       nil, // added below
		SandboxManager:     devSandboxManager,
	}

	//devAppspaceStatus := &DevAppspaceStatus{}
	appspaceStatus := &appspacestatus.AppspaceStatus{
		AppspaceModel:        devAppspaceModel,
		AppModel:             devAppModel,
		AppspaceInfoModels:   appspaceInfoModels,
		AppspacePausedEvent:  appspacePausedEvents,
		AppspaceRoutes:       nil, //added below
		MigrationJobs:        migrateJobController,
		MigrationJobsEvents:  migrateJobController,
		AppspaceStatusEvents: appspaceStatusEvents,
	}
	appspaceStatus.Init()
	migrateJobController.AppspaceStatus = appspaceStatus

	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: devSandboxManager,
		Metrics:        &m}

	routeEvents := &events.AppspaceRouteHitEvents{}
	appspaceRoutesV0 := &appspaceroutes.V0{
		AppspaceRouteModels: appspaceRouteModels,
		DropserverRoutes:    &appspaceroutes.DropserverRoutesV0{},
		SandboxProxy:        sandboxProxy,
		Authenticator:       devAuth,
		RouteHitEvents:      routeEvents,
		//AppspaceLogin:       appspaceLogin,	// should never happen, leave nil. It will crash if we made a mistake.
		Config: runtimeConfig}

	appspaceRoutes := &appspaceroutes.AppspaceRoutes{
		AppModel:       devAppModel,
		AppspaceModel:  devAppspaceModel,
		AppspaceStatus: appspaceStatus,
		V0:             appspaceRoutesV0}
	appspaceRoutes.Init()
	appspaceStatus.AppspaceRoutes = appspaceRoutes

	revServices := &domain.ReverseServices{
		Routes: appspaceRouteModels,
	}
	devSandboxManager.Services = revServices

	devSandboxMaker := &DevSandboxMaker{
		ReverseServices: revServices,
		Config:          runtimeConfig}

	migrateJobController.SandboxMaker = devSandboxMaker

	migrateJobController.Start()

	dsDevHandler := &DropserverDevServer{
		DevAppModel:            devAppModel,
		DevAppspaceModel:       devAppspaceModel,
		MigrationJobModel:      devMigrationJobModel,
		MigrationJobController: migrateJobController,
		Config:                 runtimeConfig,
		AppspaceStatusEvents:   appspaceStatusEvents,
		RouteEvents:            routeEvents}
	dsDevHandler.SetBaseData(BaseData{
		AppPath:        *appDirFlag,
		AppName:        appFilesMeta.AppName,
		AppVersion:     string(appFilesMeta.AppVersion),
		AppSchema:      appFilesMeta.SchemaVersion,
		AppMigrations:  appFilesMeta.Migrations,
		AppspacePath:   *appspaceDirFlag,
		AppspaceSchema: appspaceSchema})

	// Create server.
	server := &Server{
		Authenticator:        devAuth,
		Config:               runtimeConfig,
		DropserverDevHandler: dsDevHandler,
		AppspaceRoutes:       appspaceRoutes}

	fmt.Println("starting server")

	// start things up
	//migrationJobCtl.Start()

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

}
