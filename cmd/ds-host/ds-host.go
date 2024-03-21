package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/teleclimber/DropServer/cmd/ds-host/appops"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacedb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacelogger"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacelogin"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspaceops"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacerouter"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacestatus"
	"github.com/teleclimber/DropServer/cmd/ds-host/authenticator"
	"github.com/teleclimber/DropServer/cmd/ds-host/certificatemanager.go"
	"github.com/teleclimber/DropServer/cmd/ds-host/database"
	"github.com/teleclimber/DropServer/cmd/ds-host/domaincontroller"
	"github.com/teleclimber/DropServer/cmd/ds-host/ds2ds"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacefilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacemodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/contactmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/cookiemodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/dropidmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/migrationjobmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/remoteappspacemodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/sandboxruns"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/settingsmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/userinvitationmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/usermodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/runtimeconfig"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/server"
	"github.com/teleclimber/DropServer/cmd/ds-host/twineservices"
	"github.com/teleclimber/DropServer/cmd/ds-host/userroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/views"
	"github.com/teleclimber/DropServer/cmd/ds-host/vxservices"
	"github.com/teleclimber/DropServer/internal/checkinject"
)

// cmd_version holds the version string (current git tag, etc...) and is set at build time
var cmd_version = ""

var configFlag = flag.String("config", "", "use this JSON confgiuration file")

var migrateFlag = flag.Bool("migrate", false, "Set migrate flag to migrate db as needed.")

var dumpRoutesFlag = flag.String("dump-routes", "", "dump routes in markdown format to this location")

var checkInjectOut = flag.String("checkinject-out", "", "dump checkinject data to specified file")

func main() {
	flag.Parse()

	// serve pprof routes if DEBUG is on
	if os.Getenv("DEBUG") != "" {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
		go func() {
			fmt.Println("Starting server for pprof")
			log.Println(http.ListenAndServe("localhost:6060", nil)) // makes pprof routes available
		}()
	}

	runtimeConfig := runtimeconfig.Load(*configFlag)

	if cmd_version == "" {
		cmd_version = "unspecified"
	}
	runtimeConfig.Exec.CmdVersion = cmd_version

	record.InitDsLogger()
	err := record.SetLogOutput(runtimeConfig.Log)
	if err != nil {
		panic(err)
	}

	record.NewDsLogger().Log("ds-host version: " + cmd_version)

	if runtimeConfig.Prometheus.Enable && !*migrateFlag {
		record.ExposePromMetrics(*runtimeConfig)
	}

	copyEmbeddedFiles(*runtimeConfig)

	appLocation2Path := &runtimeconfig.AppLocation2Path{
		Config: runtimeConfig}
	appspaceLocation2Path := &runtimeconfig.AppspaceLocation2Path{
		Config: runtimeConfig}

	dbManager := &database.Manager{
		Config: runtimeConfig}

	db, err := dbManager.Open()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	migrator := &migrate.Migrator{
		Steps:     migrate.MigrationSteps,
		Config:    runtimeConfig,
		DBManager: dbManager}

	if *migrateFlag {
		err := migrator.Migrate("")
		if err == migrate.ErrNoMigrationNeeded {
			fmt.Println("Schema matches desired schema, no migration needed")
			os.Exit(0)
		}
		if err != nil {
			fmt.Println("Error Migrating", err.Error())
			os.Exit(1)
		}

		sc := dbManager.GetSchema()
		fmt.Println("schema after migration:", sc)
		os.Exit(0)
	}

	// now check schema?
	if dbManager.GetSchema() != migrator.LastStepName() {
		fmt.Println("gotta migrate:", dbManager.GetSchema(), "->", migrator.LastStepName())
		os.Exit(1)
	}

	// events
	appspaceFilesEvents := &events.AppspaceFilesEvents{}
	appspaceStatusEvents := &events.AppspaceStatusEvents{}
	migrationJobEvents := &events.MigrationJobEvents{}
	appUrlDataEvents := &events.AppUrlDataEvents{}

	// models
	settingsModel := &settingsmodel.SettingsModel{
		DB: db}
	settingsModel.PrepareStatements()

	userInvitationModel := &userinvitationmodel.UserInvitationModel{
		DB: db}
	userInvitationModel.PrepareStatements()

	userModel := &usermodel.UserModel{
		DB: db}
	userModel.PrepareStatements()

	cookieModel := &cookiemodel.CookieModel{
		DB: db}
	cookieModel.PrepareStatements()

	contactModel := &contactmodel.ContactModel{
		DB: db}
	contactModel.PrepareStatements()

	dropIDModel := &dropidmodel.DropIDModel{
		DB: db}
	dropIDModel.PrepareStatements()

	appFilesModel := &appfilesmodel.AppFilesModel{
		AppLocation2Path: appLocation2Path,
		Config:           runtimeConfig}

	appModel := &appmodel.AppModel{
		DB:               db,
		AppUrlDataEvents: appUrlDataEvents}
	appModel.PrepareStatements()

	appspaceFilesModel := &appspacefilesmodel.AppspaceFilesModel{
		Config:              runtimeConfig,
		AppspaceFilesEvents: appspaceFilesEvents}

	appspaceModel := &appspacemodel.AppspaceModel{
		DB: db}
	appspaceModel.PrepareStatements()

	remoteAppspaceModel := &remoteappspacemodel.RemoteAppspaceModel{
		DB: db,
	}
	remoteAppspaceModel.PrepareStatements()

	sandboxRunsModel := &sandboxruns.SandboxRunsModel{
		DB: db}
	sandboxRunsModel.PrepareStatements()

	appLogger := &appspacelogger.AppLogger{
		AppLocation2Path: appLocation2Path}
	appLogger.Init()

	appspaceLogger := &appspacelogger.AppspaceLogger{
		AppspaceModel: appspaceModel,
		//AppspaceStatus: see below,
		Config: runtimeConfig}
	appspaceLogger.Init()

	appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
		Config:        runtimeConfig,
		AppspaceModel: appspaceModel}
	appspaceMetaDb.Init()

	appspaceInfoModel := &appspacemetadb.InfoModel{
		AppspaceMetaDB: appspaceMetaDb}

	appspaceUsersModelV0 := &appspacemetadb.UsersV0{
		AppspaceMetaDB: appspaceMetaDb,
	}

	appspaceDB := &appspacedb.AppspaceDB{
		Config: runtimeConfig,
	}
	appspaceDB.Init()

	v0AppRoutes := &appspacerouter.V0AppRoutes{
		AppModel:      appModel,
		AppFilesModel: appFilesModel,
		Config:        runtimeConfig,
	}
	v0AppRoutes.Init()

	migrationJobModel := &migrationjobmodel.MigrationJobModel{
		MigrationJobEvents: migrationJobEvents,
		DB:                 db}
	migrationJobModel.PrepareStatements()
	migrationJobModel.StartupFinishStartedJobs("Job found unfinished at startup")

	var cGroups *sandbox.CGroups
	if runtimeConfig.Sandbox.UseCGroups {
		cGroups = &sandbox.CGroups{
			Config: runtimeConfig,
		}
		err = cGroups.Init()
		if err != nil {
			panic(err)
		}
	}

	sandboxManager := &sandbox.Manager{
		SandboxRuns:           sandboxRunsModel,
		CGroups:               cGroups,
		AppLogger:             appLogger,
		AppspaceLogger:        appspaceLogger,
		AppLocation2Path:      appLocation2Path,
		AppspaceLocation2Path: appspaceLocation2Path,
		Config:                runtimeConfig,
	}

	domainController := &domaincontroller.DomainController{
		Config:        runtimeConfig,
		AppspaceModel: appspaceModel,
	}

	pauseAppspace := &appspaceops.PauseAppspace{
		AppspaceModel:  appspaceModel,
		AppspaceStatus: nil, // see below
		SandboxManager: sandboxManager,
		AppspaceLogger: appspaceLogger,
	}
	backupAppspace := &appspaceops.BackupAppspace{
		AppspaceModel:         appspaceModel,
		SandboxManager:        sandboxManager,
		AppspaceStatus:        nil,
		AppspaceMetaDB:        appspaceMetaDb,
		AppspaceDB:            appspaceDB,
		AppspaceLogger:        appspaceLogger,
		AppspaceLocation2Path: appspaceLocation2Path,
	}
	restoreAppspace := &appspaceops.RestoreAppspace{
		InfoModel:             appspaceInfoModel,
		AppspaceModel:         appspaceModel,
		AppspaceFilesModel:    appspaceFilesModel,
		AppspaceStatus:        nil,
		AppspaceMetaDB:        appspaceMetaDb,
		AppspaceDB:            appspaceDB,
		AppspaceLogger:        appspaceLogger,
		AppspaceLocation2Path: appspaceLocation2Path,
	}
	restoreAppspace.Init()

	migrationJobCtl := &appspaceops.MigrationJobController{
		AppspaceModel:     appspaceModel,
		AppModel:          appModel,
		AppspaceInfoModel: appspaceInfoModel,
		SandboxManager:    sandboxManager,
		BackupAppspace:    backupAppspace,
		RestoreAppspace:   restoreAppspace,
		AppspaceLogger:    appspaceLogger,
		AppspaceStatus:    nil, // added below
		MigrationJobModel: migrationJobModel}

	createAppspace := &appspaceops.CreateAppspace{
		AppspaceModel:          appspaceModel,
		AppspaceFilesModel:     appspaceFilesModel,
		AppspaceMetaDB:         appspaceMetaDb,
		AppspaceUsersModelV0:   appspaceUsersModelV0,
		DomainController:       domainController,
		MigrationJobModel:      migrationJobModel,
		MigrationJobController: migrationJobCtl}

	deleteAppspace := &appspaceops.DeleteAppspace{
		AppspaceStatus:     nil,
		AppspaceModel:      appspaceModel,
		AppspaceFilesModel: appspaceFilesModel,
		DomainController:   domainController,
		MigrationJobModel:  migrationJobModel,
		SandboxManager:     sandboxManager,
		AppspaceLogger:     appspaceLogger,
	}

	remoteAppGetter := &appops.RemoteAppGetter{
		Config:        runtimeConfig,
		AppFilesModel: appFilesModel,
		AppModel:      appModel,
	}
	remoteAppGetter.Init()

	appGetter := &appops.AppGetter{
		AppFilesModel:    appFilesModel,
		AppLocation2Path: appLocation2Path,
		AppModel:         appModel,
		AppLogger:        appLogger,
		RemoteAppGetter:  remoteAppGetter,
		SandboxManager:   sandboxManager,
		V0AppRoutes:      v0AppRoutes,
	}
	appGetter.Init()

	// auth
	authenticator := &authenticator.Authenticator{
		CookieModel: cookieModel,
		Config:      runtimeConfig}

	ds2ds := &ds2ds.DS2DS{
		Config: runtimeConfig,
	}
	ds2ds.Init()

	v0tokenManager := &appspacelogin.V0TokenManager{
		Config:               *runtimeConfig,
		DS2DS:                ds2ds,
		AppspaceModel:        appspaceModel,
		AppspaceUsersModelV0: appspaceUsersModelV0,
	}
	v0tokenManager.Start()

	v0requestToken := &appspacelogin.V0RequestToken{
		Config:              *runtimeConfig,
		DS2DS:               ds2ds,
		RemoteAppspaceModel: remoteAppspaceModel,
	}

	sandboxManager.Init()

	// controllers:

	setupKey := &runtimeconfig.SetupKey{
		Config:    runtimeConfig,
		DBManager: dbManager,
		UserModel: userModel,
	}

	deleteApp := &appops.DeleteApp{
		AppFilesModel: appFilesModel,
		AppModel:      appModel,
		AppspaceModel: appspaceModel,
		AppLogger:     appLogger,
	}

	appspaceStatus := &appspacestatus.AppspaceStatus{
		AppspaceModel:        appspaceModel,
		AppModel:             appModel,
		AppspaceInfoModel:    appspaceInfoModel,
		MigrationJobEvents:   migrationJobEvents,
		AppspaceFilesEvents:  appspaceFilesEvents,
		AppspaceStatusEvents: appspaceStatusEvents,
		//AppspaceRouter: see below
	}
	appspaceStatus.Init()
	pauseAppspace.AppspaceStatus = appspaceStatus
	backupAppspace.AppspaceStatus = appspaceStatus
	restoreAppspace.AppspaceStatus = appspaceStatus
	migrationJobCtl.AppspaceStatus = appspaceStatus
	appspaceMetaDb.AppspaceStatus = appspaceStatus
	appspaceLogger.AppspaceStatus = appspaceStatus
	deleteAppspace.AppspaceStatus = appspaceStatus

	migrationMinder := &appspacestatus.MigrationMinder{
		AppModel: appModel,
	}

	appspaceAvatars := &appspaceops.Avatars{
		Config:                runtimeConfig,
		AppspaceLocation2Path: appspaceLocation2Path}

	var certificateManager *certificatemanager.CertficateManager
	if runtimeConfig.ManageTLSCertificates.Enable {
		certificateManager = &certificatemanager.CertficateManager{
			Config: runtimeConfig,
		}
		certificateManager.Init()
		domainController.CertificateManager = certificateManager
	}

	// Create proxy
	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: sandboxManager}

	// Views
	views := &views.Views{
		Config: runtimeConfig}
	views.PrepareTemplates()

	// Create routes
	authRoutes := &userroutes.AuthRoutes{
		Views:               views,
		SettingsModel:       settingsModel,
		UserModel:           userModel,
		UserInvitationModel: userInvitationModel,
		Authenticator:       authenticator,
		SetupKey:            setupKey}

	appspaceLoginRoutes := &userroutes.AppspaceLoginRoutes{
		Config:              runtimeConfig,
		AppspaceModel:       appspaceModel,
		RemoteAppspaceModel: remoteAppspaceModel,
		DS2DS:               ds2ds,
		V0RequestToken:      v0requestToken,
		V0TokenManager:      v0tokenManager,
	}

	adminRoutes := &userroutes.AdminRoutes{
		UserModel:           userModel,
		SettingsModel:       settingsModel,
		UserInvitationModel: userInvitationModel}

	applicationRoutes := &userroutes.ApplicationRoutes{
		AppGetter:       appGetter,
		RemoteAppGetter: remoteAppGetter,
		DeleteApp:       deleteApp,
		AppFilesModel:   appFilesModel,
		AppModel:        appModel,
		AppLogger:       appLogger}

	userAppspaceUserRoutes := &userroutes.AppspaceUserRoutes{
		AppspaceUsersModelV0:  appspaceUsersModelV0,
		Avatars:               appspaceAvatars,
		Config:                runtimeConfig,
		AppspaceLocation2Path: appspaceLocation2Path,
	}
	exportAppspaceRoutes := &userroutes.AppspaceBackupRoutes{
		AppspaceFilesModel:    appspaceFilesModel,
		BackupAppspace:        backupAppspace,
		AppspaceLocation2Path: appspaceLocation2Path,
	}
	restoreAppspaceRoutes := &userroutes.AppspaceRestoreRoutes{
		RestoreAppspace: restoreAppspace,
	}
	userAppspaceRoutes := &userroutes.AppspaceRoutes{
		Config:                *runtimeConfig,
		AppspaceUserRoutes:    userAppspaceUserRoutes,
		AppspaceModel:         appspaceModel,
		AppspaceExportRoutes:  exportAppspaceRoutes,
		AppspaceRestoreRoutes: restoreAppspaceRoutes,
		DropIDModel:           dropIDModel,
		MigrationMinder:       migrationMinder,
		AppspaceMetaDB:        appspaceMetaDb,
		CreateAppspace:        createAppspace,
		PauseAppspace:         pauseAppspace,
		DeleteAppspace:        deleteAppspace,
		AppspaceLogger:        appspaceLogger,
		SandboxRunsModel:      sandboxRunsModel,
		AppModel:              appModel}

	remoteAppspaceRoutes := &userroutes.RemoteAppspaceRoutes{
		RemoteAppspaceModel: remoteAppspaceModel,
		AppspaceModel:       appspaceModel,
		DropIDModel:         dropIDModel,
	}

	contactRoutes := &userroutes.ContactRoutes{
		ContactModel: contactModel,
	}

	domainNameRoutes := &userroutes.DomainNameRoutes{
		DomainController: domainController,
	}

	dropIDRoutes := &userroutes.DropIDRoutes{
		DomainController: domainController,
		DropIDModel:      dropIDModel,
	}

	migrationJobRoutes := &userroutes.MigrationJobRoutes{
		AppModel:               appModel,
		AppspaceModel:          appspaceModel,
		MigrationJobModel:      migrationJobModel,
		MigrationJobController: migrationJobCtl,
	}

	appspaceStatusTwine := &twineservices.AppspaceStatusService{
		AppspaceModel:        appspaceModel,
		AppspaceStatus:       appspaceStatus,
		AppspaceStatusEvents: appspaceStatusEvents,
	}
	migrationJobTwine := &twineservices.MigrationJobService{
		AppspaceModel:      appspaceModel,
		MigrationJobModel:  migrationJobModel,
		MigrationJobEvents: migrationJobEvents,
	}
	appGetterTwine := &twineservices.AppGetterService{
		AppGetter: appGetter,
	}

	userRoutes := &userroutes.UserRoutes{
		Config:               runtimeConfig,
		Authenticator:        authenticator,
		AuthRoutes:           authRoutes,
		AppspaceLoginRoutes:  appspaceLoginRoutes,
		AdminRoutes:          adminRoutes,
		ApplicationRoutes:    applicationRoutes,
		AppspaceRoutes:       userAppspaceRoutes,
		RemoteAppspaceRoutes: remoteAppspaceRoutes,
		ContactRoutes:        contactRoutes,
		DomainRoutes:         domainNameRoutes,
		DropIDRoutes:         dropIDRoutes,
		MigrationJobRoutes:   migrationJobRoutes,
		AppspaceStatusTwine:  appspaceStatusTwine,
		MigrationJobTwine:    migrationJobTwine,
		AppGetterTwine:       appGetterTwine,
		UserModel:            userModel,
		Views:                views}
	userRoutes.Init()
	userRoutes.DumpRoutes(*dumpRoutesFlag)

	v0dropserverRoutes := &appspacerouter.V0DropserverRoutes{
		AppspaceModel:  appspaceModel,
		Authenticator:  authenticator,
		V0RequestToken: v0requestToken,
		V0TokenManager: v0tokenManager,
	}
	dropserverRoutes := &appspacerouter.DropserverRoutes{
		V0DropServerRoutes: v0dropserverRoutes,
	}

	v0appspaceRouter := &appspacerouter.V0{
		V0AppRoutes:           v0AppRoutes,
		AppspaceUsersModelV0:  appspaceUsersModelV0,
		SandboxProxy:          sandboxProxy,
		Authenticator:         authenticator,
		V0TokenManager:        v0tokenManager,
		Config:                runtimeConfig,
		AppLocation2Path:      appLocation2Path,
		AppspaceLocation2Path: appspaceLocation2Path}
	v0appspaceRouter.Init()

	appspaceRouter := &appspacerouter.AppspaceRouter{
		Authenticator:    authenticator,
		AppModel:         appModel,
		AppspaceModel:    appspaceModel,
		AppspaceStatus:   appspaceStatus,
		DropserverRoutes: dropserverRoutes,
		V0AppspaceRouter: v0appspaceRouter}
	appspaceRouter.Init()
	appspaceStatus.AppspaceRouter = appspaceRouter

	services := &vxservices.VXServices{
		AppspaceUsersV0: appspaceUsersModelV0,
		V0AppspaceDB:    appspaceDB.V0}
	sandboxManager.Services = services

	// Create server.
	server := &server.Server{
		Config:             runtimeConfig,
		CertificateManager: certificateManager,
		UserRoutes:         userRoutes,
		AppspaceRouter:     appspaceRouter}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	exit := make(chan struct{})
	go func() {
		sig := <-sigs
		record.Log(fmt.Sprintf("Caught signal %v, quitting.", sig))

		sandboxManager.StopAll()
		record.Debug("All sandbox stopped")

		v0tokenManager.Stop()

		migrationJobCtl.Stop() // We should make all stop things async and have a waitgroup for them.

		restoreAppspace.DeleteAll()

		remoteAppGetter.Stop()
		appGetter.Stop()

		server.Shutdown()

		record.StopPromMetrics()

		err = record.CloseLogOutput()
		if err != nil {
			panic(err)
		}

		exit <- struct{}{}
	}()

	if os.Getenv("DEBUG") != "" || *checkInjectOut != "" {
		depGraph := checkinject.Collect(*server)
		if *checkInjectOut != "" {
			depGraph.GenerateDotFile(*checkInjectOut, []interface{}{runtimeConfig, appLocation2Path, appspaceLocation2Path})
		}
		depGraph.CheckMissing()
	}

	// start things up
	migrationJobCtl.Start() // TODO: add delay, maybe set in runtimeconfig for first job to run

	server.Start()

	go domainController.ResumeManagingCertificates()

	// Reveal the setup key in the log
	setupKey.RevealKey()

	<-exit

}
