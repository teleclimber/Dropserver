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
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacerouter"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacestatus"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/twineservices"
	"github.com/teleclimber/DropServer/cmd/ds-host/vxservices"
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
	migrationJobEvents := &events.MigrationJobEvents{}
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

	devAppspaceUserModel := &DevAppspaceUserModel{}
	devAppspaceUserModel.Init()

	//devAppspaceContactModel := &DevAppspaceContactModel{}

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
		Config:        runtimeConfig}
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
		RouteEvents:    appspaceRouteEvents}
	appspaceRouteModels.Init()

	// appspaceUserModels := &appspacemetadb.AppspaceUserModels{
	// 	// apparently config and validator are unused
	// 	AppspaceMetaDB: appspaceMetaDb,
	// }
	// appspaceUserModels.Init()

	devAuth := &DevAuthenticator{
		noAuth: true} // start as public

	devMigrationJobModel := &DevMigrationJobModel{
		MigrationJobEvents: migrationJobEvents,
	}

	devAppspaceModel.Appspace = domain.Appspace{
		OwnerID:     ownerID,
		AppspaceID:  appspaceID,
		AppID:       appID,
		AppVersion:  devAppModel.Ver.Version,
		DomainName:  "",
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
		SandboxManager:     devSandboxManager}

	//devAppspaceStatus := &DevAppspaceStatus{}
	appspaceStatus := &appspacestatus.AppspaceStatus{
		AppspaceModel:        devAppspaceModel,
		AppModel:             devAppModel,
		AppspaceInfoModels:   appspaceInfoModels,
		AppspacePausedEvent:  appspacePausedEvents,
		AppspaceFilesEvents:  appspaceFilesEvents,
		AppspaceRouter:       nil, //added below
		MigrationJobEvents:   migrationJobEvents,
		AppspaceStatusEvents: appspaceStatusEvents,
		AppVersionEvents:     appVersionEvents,
	}
	appspaceStatus.Init()
	migrateJobController.AppspaceStatus = appspaceStatus
	appspaceMetaDb.AppspaceStatus = appspaceStatus

	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: devSandboxManager,
		Metrics:        &m}

	appspaceRouterV0 := &appspacerouter.V0{
		AppspaceUserModel:   devAppspaceUserModel,
		AppspaceRouteModels: appspaceRouteModels,
		SandboxProxy:        sandboxProxy,
		Authenticator:       devAuth,
		RouteHitEvents:      routeHitEvents,
		Config:              runtimeConfig}

	appspaceRouter := &appspacerouter.AppspaceRouter{
		AppModel:         devAppModel,
		AppspaceModel:    devAppspaceModel,
		AppspaceStatus:   appspaceStatus,
		V0AppspaceRouter: appspaceRouterV0,
	}
	appspaceRouter.Init()
	appspaceStatus.AppspaceRouter = appspaceRouter

	appspaceDB := &appspacedb.AppspaceDB{
		Config: runtimeConfig,
	}
	appspaceDB.Init()

	// Appspace sandbox twine services:
	vxUserModels := &vxservices.VxUserModels{
		AppspaceUserModel: devAppspaceUserModel,
	}
	services := &vxservices.VXServices{
		RouteModels:  appspaceRouteModels,
		UserModels:   vxUserModels,
		V0AppspaceDB: appspaceDB.V0}

	devSandboxManager.Services = services

	devSandboxMaker := &DevSandboxMaker{
		AppspaceLogger: appspaceLogger,
		Services:       services,
		Config:         runtimeConfig}

	migrateJobController.SandboxMaker = devSandboxMaker

	migrateJobController.Start()

	appspaceStatus.Ready(appspaceID) // this puts the appspace in status map, so it gets tracked, and therefore forwards events. Not a great paradigm.

	// Ds-dev frontend twine services:
	appMetaService := &AppMetaService{
		AppVersionEvents: appVersionEvents,
		AppFilesModel:    appFilesModel,
	}
	routesService := &RoutesService{
		AppspaceRouteModels: appspaceRouteModels,
		AppspaceRouteEvents: appspaceRouteEvents,
		AppspaceFilesEvents: appspaceFilesEvents}

	userService := &UserService{
		DevAuthenticator:    devAuth,
		AppspaceUserModel:   devAppspaceUserModel,
		AppspaceFilesEvents: appspaceFilesEvents}

	routeHitService := &RouteHitService{
		RouteHitEvents:    routeHitEvents,
		AppspaceUserModel: devAppspaceUserModel}

	migrationJobTwine := &twineservices.MigrationJobService{
		AppspaceModel:      devAppspaceModel,
		MigrationJobModel:  devMigrationJobModel,
		MigrationJobEvents: migrationJobEvents,
	}

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
		AppMetaService:         appMetaService,
		RoutesService:          routesService,
		UserService:            userService,
		RouteHitService:        routeHitService,
		AppspaceStatusEvents:   appspaceStatusEvents,
		AppspaceLogEvents:      appspaceLogEvents,
		MigrationJobService:    migrationJobTwine}
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
