package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspaceroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/internal/validator"
)

var appDirFlag = flag.String("app", "", "specify root directory of app code")
var appspaceDirFlag = flag.String("appspace", "", "specify root directory of appspace data")

var execPathFlag = flag.String("exec-path", "", "specify where the exec path is so resources can be loaded")

func main() {

	ownerID := domain.UserID(7)
	appID := domain.AppID(11)
	appspaceID := domain.AppspaceID(15)

	m := record.Metrics{}

	validator := &validator.Validator{}
	validator.Init()

	flag.Parse()

	runtimeConfig := GetConfig(*execPathFlag, *appDirFlag, *appspaceDirFlag)

	if *appDirFlag == "" {
		fmt.Println("Please specify app dir")
		os.Exit(1)
	}
	if *appspaceDirFlag == "" {
		fmt.Println("Please specify appspace dir")
		os.Exit(1)
	}

	appFilesModel := appfilesmodel.AppFilesModel{
		Config: runtimeConfig,
	}

	appFilesMeta, dsErr := appFilesModel.ReadMeta("")
	if dsErr != nil {
		fmt.Println("Failed to read app metadata: " + dsErr.PublicString())
	}

	// Now read appspace metadata.
	devAppspaceModel := &DevAppspaceModel{}

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
		fmt.Printf("Schema mismatch: app: %v <> appspace: %v \n", appFilesMeta.SchemaVersion, appspaceSchema)
	}

	devAuth := &DevAuthenticator{}
	devAuth.Set(domain.Authentication{
		HasUserID:  true,
		UserID:     ownerID,
		AppspaceID: appspaceID,
	})
	devSandboxManager := &DevSandboxManager{}

	devAppModel := &DevAppModel{}
	devAppModel.Set(
		domain.App{
			OwnerID: ownerID,
			AppID:   appID,
			Created: time.Now(),
			Name:    appFilesMeta.AppName},
		domain.AppVersion{
			AppID:       appID,
			AppName:     appFilesMeta.AppName,
			Version:     appFilesMeta.AppVersion,
			Schema:      appFilesMeta.SchemaVersion,
			Created:     time.Now(),
			LocationKey: "",
		})

	devAppspaceModel.Set(domain.Appspace{
		OwnerID:     ownerID,
		AppspaceID:  appspaceID,
		AppID:       appID,
		AppVersion:  domain.Version("0.0.0"), // This isn't written in appspace meta. It may not matter. It's schema that makes a difference.
		Subdomain:   "",
		Created:     time.Now(),
		LocationKey: "",
		Paused:      false,
	})

	devAppspaceStatus := &DevAppspaceStatus{}

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
		AppspaceStatus: devAppspaceStatus,
		V0:             appspaceRoutesV0}
	appspaceRoutes.Init()
	//devAppspaceStatus.AppspaceRoutes = appspaceRoutes

	revServices := &domain.ReverseServices{
		Routes: appspaceRouteModels,
	}
	devSandboxManager.Services = revServices
	//migrationSandboxMaker.ReverseServices = revServices

	dsDevHandler := &DropserverDevServer{
		Config:      runtimeConfig,
		RouteEvents: routeEvents}
	dsDevHandler.SetBaseData(BaseData{
		AppPath:        *appDirFlag,
		AppName:        appFilesMeta.AppName,
		AppVersion:     string(appFilesMeta.AppVersion),
		AppSchema:      appFilesMeta.SchemaVersion,
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
