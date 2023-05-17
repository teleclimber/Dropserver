package main

import (
	"embed"
	"flag"
	"fmt"
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

//go:embed avatars
var avatarsFS embed.FS

var appFlag = flag.String("app", "", "specify root directory of app code or location of packaged app") // "... or URL"
var createPackageFlag = flag.String("create-package", "", "create package and output at directory")
var appspaceDirFlag = flag.String("appspace", "", "specify root directory of appspace data")
var importMapFlag = flag.String("import-map-extras", "", "specify JSON file with additional import mappings")

var checkInjectOut = flag.String("checkinject-out", "", "dump checkinject data to specified file")

const ownerID = domain.UserID(7)
const appID = domain.AppID(11)
const appspaceID = domain.AppspaceID(15)
const appspaceLocationKey = "as12345"

func main() {
	fmt.Println("ds-dev version: " + cmd_version)

	flag.Parse()

	appOrigin := makeAbsolute(*appFlag) // assumes this is not a URL!
	appOriginType := ResolveAppOrigin(*appFlag)

	appspaceSourceDir := makeAbsolute(*appspaceDirFlag)

	checkFlags(appOriginType)

	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Println("Temp dir: " + tempDir)

	appDir := appOrigin
	if appOriginType == Package {
		appDir = extractPackage(appOrigin, tempDir)
	}

	runtimeConfig := GetConfig(appDir, tempDir)

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
	appProcessingEvents := &DevAppProcessingEvents{}
	inspectSandboxEvents := &InspectSandboxEvents{}
	sandboxStatusEvents := &SandboxStatusEvents{}
	// events:
	appspaceFilesEvents := &events.AppspaceFilesEvents{}
	migrationJobEvents := &events.MigrationJobEvents{}
	appspaceStatusEvents := &events.AppspaceStatusEvents{}
	routeHitEvents := &events.AppspaceRouteHitEvents{}

	appLocation2Path := &AppLocation2Path{
		AppMetaDir: appMetaDir,
		Config:     runtimeConfig}

	appspaceLocation2Path := &AppspaceLocation2Path{
		Config: runtimeConfig}

	appFilesModel := &appfilesmodel.AppFilesModel{
		AppLocation2Path: appLocation2Path,
		Config:           runtimeConfig,
	}
	devAppFilesModel := &DevAppFilesModel{
		*appFilesModel,
		nil,
		nil,
	}

	devAppModel := &DevAppModel{}
	devSingleAppModel := &DevSingleAppModel{}

	devAppspaceModel := &DevAppspaceModel{}

	devSandboxRunsModel := &DevSandboxRunsModel{}

	v0AppRoutes := &appspacerouter.V0AppRoutes{
		AppModel:      devAppModel,
		AppFilesModel: devAppFilesModel,
		Config:        runtimeConfig,
	}

	appLogger := &appspacelogger.AppLogger{
		AppLocation2Path: appLocation2Path,
	}
	appLogger.Init()

	appGetter := &appops.AppGetter{
		AppFilesModel: devAppFilesModel,
		AppModel:      devAppModel,
		V0AppRoutes:   v0AppRoutes,
		AppLogger:     appLogger,
		//SandboxManager: ,	// added below
	}
	appGetter.Init()

	appRoutesService := &AppRoutesService{
		AppFilesModel:    devAppFilesModel,
		AppVersionEvents: appVersionEvents,
	}

	devAppWatcher := &DevAppWatcher{
		AppGetter:           appGetter,
		DevAppModel:         devAppModel,
		DevAppspaceModel:    devAppspaceModel,
		DevAppProcessEvents: appProcessingEvents,
		AppVersionEvents:    appVersionEvents,
	}
	if appOriginType == Directory {
		devAppWatcher.AddDir(appOrigin)
	}

	devSandboxManager := &DevSandboxManager{
		SandboxRuns:           devSandboxRunsModel,
		AppLogger:             appLogger,
		AppspaceLogger:        nil,
		Config:                runtimeConfig,
		AppVersionEvents:      appVersionEvents,
		SandboxStatusEvents:   sandboxStatusEvents,
		AppLocation2Path:      appLocation2Path,
		AppspaceLocation2Path: appspaceLocation2Path,
	}
	devSandboxManager.Init()
	appGetter.SandboxManager = devSandboxManager

	if *createPackageFlag != "" {
		packager := &AppPackager{
			AppGetter: appGetter}
		packager.PackageApp(appOrigin, *createPackageFlag)
		os.Exit(0)
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
		destDir:             filepath.Join(runtimeConfig.Exec.AppspacesPath, appspaceLocationKey),
	}

	avatars := &appspaceops.Avatars{
		Config:                runtimeConfig,
		AppspaceLocation2Path: appspaceLocation2Path}

	appspaceInfoModel := &appspacemetadb.InfoModel{
		AppspaceMetaDB: appspaceMetaDb}

	appspaceUsersModelV0 := &appspacemetadb.UsersV0{
		AppspaceMetaDB: appspaceMetaDb,
	}

	devAuth := &DevAuthenticator{
		noAuth: true} // start as public

	devMigrationJobModel := &DevMigrationJobModel{
		DevAppModel:            devAppModel,
		AppspaceInfoModel:      appspaceInfoModel,
		MigrationJobController: nil, // see below
		MigrationJobEvents:     migrationJobEvents,
	}

	devAppspaceModel.Appspace = domain.Appspace{
		OwnerID:     ownerID,
		AppspaceID:  appspaceID,
		AppID:       appID,
		AppVersion:  devAppModel.Ver.Version,
		DomainName:  "",
		Created:     time.Now(),
		LocationKey: appspaceLocationKey,
		Paused:      false}

	appspaceLogger := &appspacelogger.AppspaceLogger{
		AppspaceModel: devAppspaceModel,
		//AppspaceStatus: see below
		Config: runtimeConfig}
	appspaceLogger.Init()
	devSandboxManager.AppspaceLogger = appspaceLogger

	importMapExtras := &ImportMapExtras{
		SandboxManager: devSandboxManager,
		AppWatcher:     devAppWatcher,
	}
	importMapExtras.Init(*importMapFlag)

	appspaceFiles.Reset()

	// We can start files watcher after import map extras have been registered.
	devAppWatcher.Start()

	pauseAppspace := &appspaceops.PauseAppspace{
		AppspaceModel:  devAppspaceModel,
		AppspaceStatus: nil, // see below
		SandboxManager: devSandboxManager,
		AppspaceLogger: appspaceLogger,
	}
	migrationJobController := &appspaceops.MigrationJobController{
		MigrationJobModel: devMigrationJobModel,
		AppModel:          devAppModel,
		AppspaceInfoModel: appspaceInfoModel,
		AppspaceModel:     devAppspaceModel,
		AppspaceLogger:    appspaceLogger,
		AppspaceStatus:    nil, //set below
		BackupAppspace:    nil, // TODO going to need something like this!
		RestoreAppspace:   nil,
		SandboxManager:    devSandboxManager}
	devMigrationJobModel.MigrationJobController = migrationJobController

	appspaceStatus := &appspacestatus.AppspaceStatus{
		AppspaceModel:        devAppspaceModel,
		AppModel:             devAppModel,
		AppspaceInfoModel:    appspaceInfoModel,
		AppspaceFilesEvents:  appspaceFilesEvents,
		AppspaceRouter:       nil, //added below
		MigrationJobEvents:   migrationJobEvents,
		AppspaceStatusEvents: appspaceStatusEvents,
		AppVersionEvents:     appVersionEvents,
	}
	appspaceStatus.Init()
	pauseAppspace.AppspaceStatus = appspaceStatus
	migrationJobController.AppspaceStatus = appspaceStatus
	appspaceMetaDb.AppspaceStatus = appspaceStatus
	appspaceLogger.AppspaceStatus = appspaceStatus

	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: devSandboxManager}

	appspaceRouterV0 := &appspacerouter.V0{
		AppspaceUsersModelV0:  appspaceUsersModelV0,
		V0AppRoutes:           v0AppRoutes,
		SandboxProxy:          sandboxProxy,
		Authenticator:         devAuth,
		RouteHitEvents:        routeHitEvents,
		Config:                runtimeConfig,
		AppLocation2Path:      appLocation2Path,
		AppspaceLocation2Path: appspaceLocation2Path}
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

	migrationJobController.Start()

	// Ds-dev frontend twine services:
	appsaceStatusService := &AppspaceStatusService{
		AppspaceStatus:       appspaceStatus,
		AppspaceStatusEvents: appspaceStatusEvents,
	}
	sandboxControlService := &SandboxControlService{
		DevSandboxManager:    devSandboxManager,
		InspectSandboxEvents: inspectSandboxEvents,
		SandboxStatusEvents:  sandboxStatusEvents,
	}
	appMetaService := &AppMetaService{
		DevAppModel:         devAppModel,
		DevAppProcessEvents: appProcessingEvents,
		AppVersionEvents:    appVersionEvents,
		AppFilesModel:       devAppFilesModel,
		AppGetter:           appGetter,
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
		Config:                 runtimeConfig,
		DevAppModel:            devAppModel,
		AppGetter:              appGetter,
		AppspaceFiles:          appspaceFiles,
		PauseAppspace:          pauseAppspace,
		AppspaceMetaDB:         appspaceMetaDb,
		AppspaceDB:             appspaceDB,
		AppspaceLogger:         appspaceLogger,
		DevSandboxManager:      devSandboxManager,
		MigrationJobModel:      devMigrationJobModel,
		MigrationJobController: migrationJobController,
		AppspaceStatus:         appspaceStatus,
		AppspaceStatusService:  appsaceStatusService,
		SandboxControlService:  sandboxControlService,
		AppMetaService:         appMetaService,
		AppRoutesService:       appRoutesService,
		UserService:            userService,
		RouteHitService:        routeHitService,
		AppspaceLogService:     appspaceLogTwine,
		MigrationJobService:    migrationJobTwine}
	dsDevHandler.SetPaths(appOrigin, appspaceSourceDir)

	// Create server.
	server := &Server{
		Config:                runtimeConfig,
		DropserverDevHandler:  dsDevHandler,
		AppspaceRouter:        appspaceRouter,
		AppspaceLocation2Path: appspaceLocation2Path}

	// experimental:
	if os.Getenv("DEBUG") != "" || *checkInjectOut != "" {
		depGraph := checkinject.Collect(*server)
		if *checkInjectOut != "" {
			depGraph.GenerateDotFile(*checkInjectOut, []interface{}{runtimeConfig, appLocation2Path, appspaceLocation2Path})
		}
		depGraph.CheckMissing()
	}

	// open the appspace log.
	appspaceLogger.Open(appspaceID)

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

}

func checkFlags(appOriginType AppSourceType) {
	if *appFlag == "" {
		fmt.Println("Please specify app")
		os.Exit(1)
	}

	if *createPackageFlag != "" {
		// rule out other flags:
		if *appspaceDirFlag != "" {
			fmt.Println("Do not specify an appspace dir when creating an app package")
			os.Exit(1)
		}
		if *importMapFlag != "" {
			fmt.Println("Do not specify import map extras when creating a package")
			os.Exit(1)
		}
		if appOriginType != Directory {
			fmt.Println("Unable to package: app should be a directory")
			os.Exit(1)
		}
	}
}
