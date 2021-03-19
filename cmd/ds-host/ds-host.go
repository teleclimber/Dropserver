package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/teleclimber/DropServer/cmd/ds-host/appgetter"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacedb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacelogger"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacelogin"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacerouter"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacestatus"
	"github.com/teleclimber/DropServer/cmd/ds-host/authenticator"
	"github.com/teleclimber/DropServer/cmd/ds-host/clihandlers"
	"github.com/teleclimber/DropServer/cmd/ds-host/database"
	"github.com/teleclimber/DropServer/cmd/ds-host/domaincontroller"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacefilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacemodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspaceusermodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/contactmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/cookiemodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/dropidmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/migrationjobmodel"
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
	"github.com/teleclimber/DropServer/internal/stdinput"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

var configFlag = flag.String("config", "", "use this JSON confgiuration file")

var migrateFlag = flag.Bool("migrate", false, "Set migrate flag to migrate db as needed.")

var addAdminFlag = flag.Bool("add-admin", false, "add an admin")

var execPathFlag = flag.String("exec-path", "", "specify where the exec path is so resources can be loaded")

var noSslFlag = flag.Bool("no-ssl", false, "Disable SSL (for dev use only)")

func main() {
	//startServer := true	// currnetly actually not used.

	flag.Parse()

	runtimeConfig := runtimeconfig.Load(*configFlag, *execPathFlag, *noSslFlag)

	dbManager := &database.Manager{
		Config: runtimeConfig}

	db, err := dbManager.Open()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	migrator := &migrate.Migrator{
		OrderedSteps: migrate.OrderedSteps,
		StringSteps:  migrate.StringSteps,
		Config:       runtimeConfig,
		DBManager:    dbManager}

	if *migrateFlag {
		//startServer = false

		err := migrator.Migrate("")
		if err != nil {
			fmt.Println("Error Migrating", err.Error())
			os.Exit(1)
		}

		sc := dbManager.GetSchema()
		fmt.Println("schema after migration:", sc)
	}

	// now check schema?
	if dbManager.GetSchema() != migrator.LastStepName() {
		fmt.Println("gotta migrate:", dbManager.GetSchema(), "->", migrator.LastStepName())
		os.Exit(1)
	}

	record.Init(runtimeConfig) // ok, but that's not how we should do it.
	// ^^ preserve this for metrics, but get rid of it eventually

	stdInput := &stdinput.StdInput{}

	// events
	appspaceFilesEvents := &events.AppspaceFilesEvents{}
	appspacePausedEvent := &events.AppspacePausedEvents{}
	appspaceStatusEvents := &events.AppspaceStatusEvents{}
	appspaceLogEvents := &events.AppspaceLogEvents{}
	migrationJobEvents := &events.MigrationJobEvents{}

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

	cliHandlers := clihandlers.CliHandlers{
		UserModel: userModel,
		StdInput:  stdInput}

	// Check we have admins before going further.
	admins, dsErr := userModel.GetAllAdmins()
	if dsErr != nil {
		fmt.Println(dsErr)
		os.Exit(1)
	}
	if len(admins) == 0 {
		fmt.Println("There are currently no admin users, please create one.")
	}

	if *addAdminFlag || len(admins) == 0 {
		//startServer = false
		err := cliHandlers.AddAdmin()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *addAdminFlag {
		os.Exit(0)
	}

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
		Config: runtimeConfig}

	appModel := &appmodel.AppModel{
		DB: db}
	appModel.PrepareStatements()

	appspaceFilesModel := &appspacefilesmodel.AppspaceFilesModel{
		Config: runtimeConfig}

	appspaceModel := &appspacemodel.AppspaceModel{
		DB:            db,
		AsPausedEvent: appspacePausedEvent}
	appspaceModel.PrepareStatements()

	appspaceUserModel := &appspaceusermodel.AppspaceUserModel{
		DB: db}
	appspaceUserModel.PrepareStatements()

	appspaceLogger := &appspacelogger.AppspaceLogger{
		AppspaceModel:     appspaceModel,
		AppspaceLogEvents: appspaceLogEvents,
		Config:            runtimeConfig}
	appspaceLogger.Init()

	appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
		Config:        runtimeConfig,
		AppspaceModel: appspaceModel}
	appspaceMetaDb.Init()

	appspaceInfoModels := &appspacemetadb.AppspaceInfoModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb}
	appspaceInfoModels.Init()

	migrationJobModel := &migrationjobmodel.MigrationJobModel{
		MigrationJobEvents: migrationJobEvents,
		DB:                 db}
	migrationJobModel.PrepareStatements()
	migrationJobModel.StartupFinishStartedJobs("Job found unfinished at startup")

	sandboxManager := &sandbox.Manager{
		AppspaceLogger: appspaceLogger,
		Config:         runtimeConfig}

	migrationSandboxMaker := &migrateappspace.SandboxMaker{
		AppspaceLogger: appspaceLogger,
		Config:         runtimeConfig}

	migrationJobCtl := &migrateappspace.JobController{
		AppspaceModel:      appspaceModel,
		AppModel:           appModel,
		AppspaceInfoModels: appspaceInfoModels,
		SandboxManager:     sandboxManager,
		SandboxMaker:       migrationSandboxMaker,
		MigrationJobModel:  migrationJobModel}

	// auth
	authenticator := &authenticator.Authenticator{
		CookieModel: cookieModel,
		Config:      runtimeConfig}

	appspaceLogin := &appspacelogin.AppspaceLogin{}
	appspaceLogin.Start()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Caught signal, quitting.", sig)
		pprof.StopCPUProfile()

		sandboxManager.StopAll()
		fmt.Println("All sandbox stopped")

		appspaceLogin.Stop()

		migrationJobCtl.Stop() // We should make all stop things async and have a waitgroup for them.

		// TODO server stop

		os.Exit(0)
	}()

	sandboxManager.Init()

	record.Debug("Main after sandbox manager start")

	// maybe we can start profiler here?
	if *cpuprofile != "" {
		fmt.Println("Starting CPU Profile")
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			fmt.Println("failed to start cpu profiler", err)
			os.Exit(1)
		}
		//defer pprof.StopCPUProfile()
	}

	m := record.Metrics{}

	// controllers:
	domainController := &domaincontroller.DomainController{
		Config:        runtimeConfig,
		AppspaceModel: appspaceModel,
	}

	appGetter := &appgetter.AppGetter{
		AppFilesModel: appFilesModel,
		AppModel:      appModel,
	}
	appGetter.Init()

	appspaceStatus := &appspacestatus.AppspaceStatus{
		AppspaceModel:      appspaceModel,
		AppModel:           appModel,
		AppspaceInfoModels: appspaceInfoModels,
		//AppspaceRouter: see below
		MigrationJobModel:    migrationJobModel,
		MigrationJobEvents:   migrationJobEvents,
		AppspaceFilesEvents:  appspaceFilesEvents,
		AppspacePausedEvent:  appspacePausedEvent,
		AppspaceStatusEvents: appspaceStatusEvents,
	}
	appspaceStatus.Init()

	migrationMinder := &appspacestatus.MigrationMinder{
		AppModel:      appModel,
		AppspaceModel: appspaceModel,
	}

	migrationJobCtl.AppspaceStatus = appspaceStatus

	// Create proxy
	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: sandboxManager,
		Metrics:        &m}

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
		AppspaceLogin:       appspaceLogin}

	adminRoutes := &userroutes.AdminRoutes{
		UserModel:           userModel,
		SettingsModel:       settingsModel,
		UserInvitationModel: userInvitationModel}

	applicationRoutes := &userroutes.ApplicationRoutes{
		AppGetter:     appGetter,
		AppFilesModel: appFilesModel,
		AppModel:      appModel,
		AppspaceModel: appspaceModel}

	userAppspaceUserRoutes := &userroutes.AppspaceUserRoutes{
		AppspaceUserModel: appspaceUserModel,
	}
	userAppspaceRoutes := &userroutes.AppspaceRoutes{
		AppspaceUserRoutes:     userAppspaceUserRoutes,
		AppspaceFilesModel:     appspaceFilesModel,
		AppspaceModel:          appspaceModel,
		DropIDModel:            dropIDModel,
		MigrationMinder:        migrationMinder,
		AppspaceMetaDB:         appspaceMetaDb,
		DomainController:       domainController,
		MigrationJobModel:      migrationJobModel,
		MigrationJobController: migrationJobCtl,
		AppModel:               appModel}

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

	userRoutes := &userroutes.UserRoutes{
		AuthRoutes:          authRoutes,
		AdminRoutes:         adminRoutes,
		ApplicationRoutes:   applicationRoutes,
		AppspaceRoutes:      userAppspaceRoutes,
		ContactRoutes:       contactRoutes,
		DomainRoutes:        domainNameRoutes,
		DropIDRoutes:        dropIDRoutes,
		MigrationJobRoutes:  migrationJobRoutes,
		AppspaceStatusTwine: appspaceStatusTwine,
		MigrationJobTwine:   migrationJobTwine,
		UserModel:           userModel}

	appspaceDB := &appspacedb.AppspaceDB{
		Config: runtimeConfig,
	}
	appspaceDB.Init()

	appspaceRouteModels := &appspacemetadb.AppspaceRouteModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb}
	appspaceRouteModels.Init()

	appspaceUserModels := &appspacemetadb.AppspaceUserModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb,
	}
	appspaceUserModels.Init()

	v0appspaceRouter := &appspacerouter.V0{
		AppspaceRouteModels: appspaceRouteModels,
		VxUserModels:        appspaceUserModels,
		DropserverRoutes:    &appspacerouter.DropserverRoutesV0{},
		SandboxProxy:        sandboxProxy,
		Authenticator:       authenticator,
		AppspaceLogin:       appspaceLogin,
		Config:              runtimeConfig}

	appspaceRouter := &appspacerouter.AppspaceRouter{
		AppModel:       appModel,
		AppspaceModel:  appspaceModel,
		AppspaceStatus: appspaceStatus,
		V0:             v0appspaceRouter}
	appspaceRouter.Init()
	appspaceStatus.AppspaceRouter = appspaceRouter

	services := &vxservices.VXServices{
		RouteModels:  appspaceRouteModels,
		UserModels:   appspaceUserModels,
		V0AppspaceDB: appspaceDB.V0}
	sandboxManager.Services = services
	migrationSandboxMaker.Services = services

	// Create server.
	server := &server.Server{
		Authenticator:  authenticator,
		Config:         runtimeConfig,
		Views:          views,
		UserRoutes:     userRoutes,
		AppspaceRouter: appspaceRouter,
		Metrics:        &m}
	server.Init()

	// start things up
	migrationJobCtl.Start() // TODO: add delay, maybe set in runtimeconfig for first job to run

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

	fmt.Println("Leaving main func")
}
