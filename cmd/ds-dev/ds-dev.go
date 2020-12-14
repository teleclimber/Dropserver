package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/appspacedb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacelogger"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacelogin"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacerouter"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacestatus"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/vxservices"
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

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Println("Temp dir: " + tempDir)

	appspaceWorkingDir := filepath.Join(tempDir, "appspace")
	err = os.MkdirAll(appspaceWorkingDir, 0744)
	if err != nil {
		panic(err)
	}

	socketsDir := filepath.Join(tempDir, "sockets")
	err = os.MkdirAll(appspaceWorkingDir, 0744)
	if err != nil {
		panic(err)
	}

	// events:
	appspaceFilesEvents := &events.AppspaceFilesEvents{}
	appVersionEvents := &AppVersionEvents{}
	appspacePausedEvents := &events.AppspacePausedEvents{}
	appspaceRouteEvents := &AppspaceRouteEvents{}
	appspaceLogEvents := &events.AppspaceLogEvents{}
	migrationJobEvents := &events.MigrationJobStatusEvents{}
	appspaceStatusEvents := &events.AppspaceStatusEvents{}
	routeHitEvents := &events.AppspaceRouteHitEvents{}

	runtimeConfig := GetConfig(*execPathFlag, *appDirFlag, appspaceWorkingDir)
	runtimeConfig.Sandbox.SocketsDir = socketsDir

	appFilesModel := &appfilesmodel.AppFilesModel{
		Config: runtimeConfig,
	}
	devAppModel := &DevAppModel{}

	devAppspaceModel := &DevAppspaceModel{
		AsPausedEvent: appspacePausedEvents}

	devAppspaceContactModel := &DevAppspaceContactModel{}

	devAppWatcher := &DevAppWatcher{
		AppFilesModel:    appFilesModel,
		DevAppModel:      devAppModel,
		DevAppspaceModel: devAppspaceModel,
		AppVersionEvents: appVersionEvents,
	}
	devAppWatcher.Start(*appDirFlag)

	// Now read appspace metadata.
	appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
		AppspaceModel: devAppspaceModel,
		Config:        runtimeConfig,
		Validator:     validator}
	appspaceMetaDb.Init()

	appspaceFiles := &DevAppspaceFiles{
		AppspaceMetaDb:      appspaceMetaDb,
		AppspaceFilesEvents: appspaceFilesEvents,
		sourceDir:           *appspaceDirFlag,
		destDir:             appspaceWorkingDir,
	}
	appspaceFiles.Reset()

	appspaceInfoModels := &appspacemetadb.AppspaceInfoModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb}
	appspaceInfoModels.Init()

	appspaceRouteModels := &appspacemetadb.AppspaceRouteModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb,
		RouteEvents:    appspaceRouteEvents,
		Validator:      validator}
	appspaceRouteModels.Init()

	appspaceUserModels := &appspacemetadb.AppspaceUserModels{
		// apparently config and validator are unused
		AppspaceMetaDB:       appspaceMetaDb,
		AppspaceContactModel: devAppspaceContactModel,
	}
	appspaceUserModels.Init()

	devAuth := &DevAuthenticator{
		noAuth: true} // start as public

	devMigrationJobModel := &DevMigrationJobModel{}

	devAppspaceModel.Appspace = domain.Appspace{
		OwnerID:     ownerID,
		AppspaceID:  appspaceID,
		AppID:       appID,
		AppVersion:  devAppModel.Ver.Version,
		Subdomain:   "",
		Created:     time.Now(),
		LocationKey: "",
		Paused:      false}

	appspaceLogger := &appspacelogger.AppspaceLogger{
		AppspaceLogEvents: appspaceLogEvents,
		AppspaceModel:     devAppspaceModel,
		Config:            runtimeConfig}
	appspaceLogger.Init()

	devSandboxManager := &DevSandboxManager{
		AppspaceLogger:   appspaceLogger,
		Config:           runtimeConfig,
		AppVersionEvents: appVersionEvents,
	}
	devSandboxManager.Init()

	migrateJobController := &migrateappspace.JobController{
		MigrationJobModel:  devMigrationJobModel,
		AppModel:           devAppModel,
		AppspaceInfoModels: appspaceInfoModels,
		AppspaceModel:      devAppspaceModel,
		AppspaceStatus:     nil, //set below
		SandboxMaker:       nil, // added below
		SandboxManager:     devSandboxManager,
		MigrationEvents:    migrationJobEvents}

	//devAppspaceStatus := &DevAppspaceStatus{}
	appspaceStatus := &appspacestatus.AppspaceStatus{
		AppspaceModel:        devAppspaceModel,
		AppModel:             devAppModel,
		AppspaceInfoModels:   appspaceInfoModels,
		AppspacePausedEvent:  appspacePausedEvents,
		AppspaceFilesEvents:  appspaceFilesEvents,
		AppspaceRouter:       nil, //added below
		MigrationJobs:        migrateJobController,
		MigrationJobsEvents:  migrationJobEvents,
		AppspaceStatusEvents: appspaceStatusEvents,
		AppVersionEvents:     appVersionEvents,
	}
	appspaceStatus.Init()
	appspaceStatus.Ready(appspaceID) // this puts the appspace in status map, so it gets tracked, and therefore forwards events. Not a great paradigm.
	migrateJobController.AppspaceStatus = appspaceStatus

	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: devSandboxManager,
		Metrics:        &m}

	appspaceLogin := &appspacelogin.AppspaceLogin{}
	appspaceLogin.Start()

	appspaceRouterV0 := &appspacerouter.V0{
		AppspaceRouteModels:  appspaceRouteModels,
		AppspaceContactModel: devAppspaceContactModel,
		DropserverRoutes:     &appspacerouter.DropserverRoutesV0{},
		SandboxProxy:         sandboxProxy,
		Authenticator:        devAuth,
		VxUserModels:         appspaceUserModels,
		RouteHitEvents:       routeHitEvents,
		AppspaceLogin:        appspaceLogin,
		Config:               runtimeConfig}

	appspaceRouter := &appspacerouter.AppspaceRouter{
		AppModel:       devAppModel,
		AppspaceModel:  devAppspaceModel,
		AppspaceStatus: appspaceStatus,
		V0:             appspaceRouterV0}
	appspaceRouter.Init()
	appspaceStatus.AppspaceRouter = appspaceRouter

	appspaceDB := &appspacedb.AppspaceDB{
		Config: runtimeConfig,
	}
	appspaceDB.Init()

	services := &vxservices.VXServices{
		RouteModels:  appspaceRouteModels,
		UserModels:   appspaceUserModels,
		V0AppspaceDB: appspaceDB.V0}

	devSandboxManager.Services = services

	devSandboxMaker := &DevSandboxMaker{
		AppspaceLogger: appspaceLogger,
		Services:       services,
		Config:         runtimeConfig}

	migrateJobController.SandboxMaker = devSandboxMaker

	migrateJobController.Start()

	// twine services:
	routesService := &RoutesService{
		AppspaceRouteModels: appspaceRouteModels,
		AppspaceRouteEvents: appspaceRouteEvents,
		AppspaceFilesEvents: appspaceFilesEvents}

	userService := &UserService{
		DevAuthenticator:        devAuth,
		AppspaceUserModels:      appspaceUserModels,
		DevAppspaceContactModel: devAppspaceContactModel,
		AppspaceFilesEvents:     appspaceFilesEvents}

	routeHitService := &RouteHitService{
		RouteHitEvents:          routeHitEvents,
		AppspaceUserModels:      appspaceUserModels,
		DevAppspaceContactModel: devAppspaceContactModel}

	dsDevHandler := &DropserverDevServer{
		DevAppModel:            devAppModel,
		AppFilesModel:          appFilesModel,
		AppspaceFiles:          appspaceFiles,
		DevAppspaceModel:       devAppspaceModel,
		AppspaceMetaDB:         appspaceMetaDb,
		AppspaceDB:             appspaceDB,
		AppspaceInfoModels:     appspaceInfoModels,
		DevSandboxManager:      devSandboxManager,
		MigrationJobModel:      devMigrationJobModel,
		MigrationJobController: migrateJobController,
		DevSandboxMaker:        devSandboxMaker,
		AppspaceStatus:         appspaceStatus,
		Config:                 runtimeConfig,
		RoutesService:          routesService,
		UserService:            userService,
		RouteHitService:        routeHitService,
		AppVersionEvents:       appVersionEvents,
		AppspaceStatusEvents:   appspaceStatusEvents,
		AppspaceLogEvents:      appspaceLogEvents,
		MigrationJobsEvents:    migrationJobEvents}
	dsDevHandler.SetPaths(*appDirFlag, *appspaceDirFlag)

	// Create server.
	server := &Server{
		Authenticator:        devAuth,
		Config:               runtimeConfig,
		DropserverDevHandler: dsDevHandler,
		AppspaceRouter:       appspaceRouter}

	fmt.Println("starting server")

	// start things up
	//migrationJobCtl.Start()

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

}
