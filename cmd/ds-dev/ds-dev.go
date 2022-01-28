package main

import (
	"embed"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/appops"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacedb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacelogger"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspaceops"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacerouter"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacestatus"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/twineservices"
	"github.com/teleclimber/DropServer/cmd/ds-host/vxservices"
	"github.com/teleclimber/DropServer/denosandboxcode"
	"github.com/teleclimber/DropServer/internal/checkinject"
	"github.com/teleclimber/DropServer/internal/embedutils"
)

// cmd_version holds the version string (current git tag, etc...) and is set at build time
var cmd_version = "unspecified"

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

//go:embed avatars
var avatarsFS embed.FS

var appDirFlag = flag.String("app", "", "specify root directory of app code")
var appspaceDirFlag = flag.String("appspace", "", "specify root directory of appspace data")

var checkInjectOut = flag.String("checkinject-out", "", "dump checkinject data to specified file")

const ownerID = domain.UserID(7)
const appID = domain.AppID(11)
const appspaceID = domain.AppspaceID(15)

func main() {
	fmt.Println("ds-dev version: " + cmd_version)

	flag.Parse()

	if *appDirFlag == "" {
		fmt.Println("Please specify app dir")
		os.Exit(1)
	}

	appspaceSourceDir := *appspaceDirFlag
	if appspaceSourceDir != "" && !filepath.IsAbs(*appspaceDirFlag) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		appspaceSourceDir = filepath.Join(wd, *appspaceDirFlag)
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Println("Temp dir: " + tempDir)

	runtimeConfig := GetConfig(*appDirFlag, tempDir)

	// in ds-host app meta is in the folder above actual app code
	// In ds-dev, since we read app files directly, have to stash app meta elsewhere.
	appMetaDir := filepath.Join(tempDir, "app-meta")

	// make all the dirs
	dirs := []string{
		runtimeConfig.Exec.AppspacesPath,
		runtimeConfig.Exec.SandboxCodePath,
		runtimeConfig.Sandbox.SocketsDir,
		appMetaDir}
	for _, d := range dirs {
		err = os.MkdirAll(d, 0744)
		if err != nil {
			panic(err)
		}
	}

	err = embedutils.DirToDisk(denosandboxcode.SandboxCode, ".", runtimeConfig.Exec.SandboxCodePath)
	if err != nil {
		panic(err)
	}

	// dev-only events:
	appVersionEvents := &DevAppVersionEvents{}
	inspectSandboxEvents := &InspectSandboxEvents{}
	// events:
	appspaceFilesEvents := &events.AppspaceFilesEvents{}
	appspacePausedEvents := &events.AppspacePausedEvents{}
	//appspaceLogEvents := &events.AppspaceLogEvents{}
	migrationJobEvents := &events.MigrationJobEvents{}
	appspaceStatusEvents := &events.AppspaceStatusEvents{}
	routeHitEvents := &events.AppspaceRouteHitEvents{}

	location2path := &Location2Path{
		AppMetaDir: appMetaDir,
		Config:     runtimeConfig}

	appFilesModel := &appfilesmodel.AppFilesModel{
		Location2Path: location2path,
		Config:        runtimeConfig,
	}
	devAppFilesModel := &DevAppFilesModel{
		*appFilesModel,
		nil,
		nil,
	}

	devAppModel := &DevAppModel{}
	devSingleAppModel := &DevSingleAppModel{}

	devAppspaceModel := &DevAppspaceModel{
		AsPausedEvent: appspacePausedEvents}

	//devAppspaceContactModel := &DevAppspaceContactModel{}

	v0AppRoutes := &appspacerouter.V0AppRoutes{
		AppModel:      devAppModel,
		AppFilesModel: devAppFilesModel,
		Config:        runtimeConfig,
	}

	appLogger := &appspacelogger.AppLogger{
		Location2Path: location2path,
	}
	appLogger.Init()

	appGetter := &appops.AppGetter{
		AppFilesModel: devAppFilesModel,
		AppModel:      devAppModel,
		V0AppRoutes:   v0AppRoutes,
		AppLogger:     appLogger,
		//SandboxMaker: ,	// added below
	}
	appGetter.Init()

	appRoutesService := &AppRoutesService{
		AppFilesModel:    devAppFilesModel,
		AppVersionEvents: appVersionEvents,
	}

	devAppWatcher := &DevAppWatcher{
		AppGetter:        appGetter,
		DevAppModel:      devAppModel,
		DevAppspaceModel: devAppspaceModel,
		AppVersionEvents: appVersionEvents,
	}

	// Now read appspace metadata.
	appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
		AppspaceModel: devAppspaceModel,
		Config:        runtimeConfig}
	appspaceMetaDb.Init()

	appspaceFiles := &DevAppspaceFiles{
		AppspaceMetaDb:      appspaceMetaDb,
		AppspaceFilesEvents: appspaceFilesEvents,
		sourceDir:           appspaceSourceDir,
		destDir:             runtimeConfig.Exec.AppspacesPath,
	}
	appspaceFiles.Reset()

	avatars := &appspaceops.Avatars{
		Config: runtimeConfig,
	}

	appspaceInfoModel := &appspacemetadb.InfoModel{
		AppspaceMetaDB: appspaceMetaDb}

	appspaceUsersModelV0 := &appspacemetadb.UsersV0{
		AppspaceMetaDB: appspaceMetaDb,
	}

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
		AppspaceModel: devAppspaceModel,
		//AppspaceStatus: see below
		Config: runtimeConfig}
	appspaceLogger.Init()

	devSandboxManager := &DevSandboxManager{
		AppspaceLogger:   appspaceLogger,
		Config:           runtimeConfig,
		AppVersionEvents: appVersionEvents,
		Location2Path:    location2path,
	}
	devSandboxManager.Init()

	migrateJobController := &appspaceops.MigrationJobController{
		MigrationJobModel: devMigrationJobModel,
		AppModel:          devAppModel,
		AppspaceInfoModel: appspaceInfoModel,
		AppspaceModel:     devAppspaceModel,
		AppspaceLogger:    appspaceLogger,
		AppspaceStatus:    nil, //set below
		BackupAppspace:    nil, // TODO going to need something like this!
		RestoreAppspace:   nil,
		SandboxMaker:      nil, // added below
		SandboxManager:    devSandboxManager}

	//devAppspaceStatus := &DevAppspaceStatus{}
	appspaceStatus := &appspacestatus.AppspaceStatus{
		AppspaceModel:        devAppspaceModel,
		AppModel:             devAppModel,
		AppspaceInfoModel:    appspaceInfoModel,
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
	appspaceLogger.AppspaceStatus = appspaceStatus

	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: devSandboxManager}

	appspaceRouterV0 := &appspacerouter.V0{
		AppspaceUsersModelV0: appspaceUsersModelV0,
		V0AppRoutes:          v0AppRoutes,
		SandboxProxy:         sandboxProxy,
		Authenticator:        devAuth,
		RouteHitEvents:       routeHitEvents,
		Location2Path:        location2path,
		Config:               runtimeConfig}
	appspaceRouterV0.Init()

	v0dropserverRoutes := &appspacerouter.V0DropserverRoutes{
		AppspaceModel: devAppspaceModel,
		Authenticator: devAuth,
	}
	dropserverRoutes := &appspacerouter.DropserverRoutes{
		V0DropServerRoutes: v0dropserverRoutes,
	}

	appspaceRouter := &appspacerouter.AppspaceRouter{
		Authenticator:    devAuth,
		AppModel:         devAppModel,
		AppspaceModel:    devAppspaceModel,
		AppspaceStatus:   appspaceStatus,
		V0AppspaceRouter: appspaceRouterV0,
		DropserverRoutes: dropserverRoutes,
	}
	appspaceRouter.Init()
	appspaceStatus.AppspaceRouter = appspaceRouter

	appspaceDB := &appspacedb.AppspaceDB{
		Config: runtimeConfig,
	}
	appspaceDB.Init()

	services := &vxservices.VXServices{
		AppspaceUsersV0: appspaceUsersModelV0,
		V0AppspaceDB:    appspaceDB.V0}

	devSandboxManager.Services = services

	devSandboxMaker := &DevSandboxMaker{
		AppspaceLogger: appspaceLogger,
		AppLogger:      appLogger,
		Services:       services,
		Location2Path:  location2path,
		Config:         runtimeConfig}

	migrateJobController.SandboxMaker = devSandboxMaker
	appGetter.SandboxMaker = devSandboxMaker

	devAppWatcher.Start(*appDirFlag)

	migrateJobController.Start()

	// Ds-dev frontend twine services:
	appsaceStatusService := &AppspaceStatusService{
		AppspaceStatus:       appspaceStatus,
		AppspaceStatusEvents: appspaceStatusEvents,
	}
	sandboxControlService := &SandboxControlService{
		DevSandboxMaker:      devSandboxMaker,
		DevSandboxManager:    devSandboxManager,
		InspectSandboxEvents: inspectSandboxEvents,
	}
	appMetaService := &AppMetaService{
		DevAppModel:      devAppModel,
		AppVersionEvents: appVersionEvents,
		AppFilesModel:    devAppFilesModel,
		AppGetter:        appGetter,
	}
	userService := &UserService{
		DevAuthenticator:     devAuth,
		AppspaceUsersModelV0: appspaceUsersModelV0,
		Avatars:              avatars,
		AppspaceFilesEvents:  appspaceFilesEvents}

	routeHitService := &RouteHitService{
		RouteHitEvents:       routeHitEvents,
		AppspaceUsersModelV0: appspaceUsersModelV0}

	migrationJobTwine := &twineservices.MigrationJobService{
		AppspaceModel:      devAppspaceModel,
		MigrationJobModel:  devMigrationJobModel,
		MigrationJobEvents: migrationJobEvents,
	}
	appspaceLogTwine := &twineservices.AppspaceLogService{
		AppspaceModel:  devAppspaceModel,
		AppModel:       devSingleAppModel,
		AppspaceLogger: appspaceLogger,
		AppLogger:      appLogger,
	}

	dsDevHandler := &DropserverDevServer{
		DevAppModel:            devAppModel,
		AppGetter:              appGetter,
		AppspaceFiles:          appspaceFiles,
		DevAppspaceModel:       devAppspaceModel,
		AppspaceMetaDB:         appspaceMetaDb,
		AppspaceDB:             appspaceDB,
		AppspaceInfoModel:      appspaceInfoModel,
		AppspaceLogger:         appspaceLogger,
		DevSandboxManager:      devSandboxManager,
		MigrationJobModel:      devMigrationJobModel,
		MigrationJobController: migrateJobController,
		AppspaceStatus:         appspaceStatus,
		AppspaceStatusService:  appsaceStatusService,
		SandboxControlService:  sandboxControlService,
		AppMetaService:         appMetaService,
		AppRoutesService:       appRoutesService,
		UserService:            userService,
		RouteHitService:        routeHitService,
		AppspaceLogService:     appspaceLogTwine,
		MigrationJobService:    migrationJobTwine}
	dsDevHandler.SetPaths(*appDirFlag, *appspaceDirFlag)

	// Create server.
	server := &Server{
		Config:               runtimeConfig,
		DropserverDevHandler: dsDevHandler,
		AppspaceRouter:       appspaceRouter}

	// experimental:
	if os.Getenv("DEBUG") != "" || *checkInjectOut != "" {
		depGraph := checkinject.Collect(*server)
		if *checkInjectOut != "" {
			depGraph.GenerateDotFile(*checkInjectOut, []interface{}{runtimeConfig, location2path})
		}
		depGraph.CheckMissing()
	}

	// open the appspace log.
	appspaceLogger.Open(appspaceID)

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

}
