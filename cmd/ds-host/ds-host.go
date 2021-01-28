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
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacefilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacemodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/contactmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/cookiemodel"
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
	"github.com/teleclimber/DropServer/internal/validator"
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

		dsErr := migrator.Migrate("")
		if dsErr != nil {
			fmt.Println("Error Migrating", dsErr.PublicString(), dsErr.ExtraMessage())
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

	validator := &validator.Validator{}
	validator.Init()

	stdInput := &stdinput.StdInput{}

	// events
	appspaceFilesEvents := &events.AppspaceFilesEvents{}
	appspacePausedEvent := &events.AppspacePausedEvents{}
	appspaceStatusEvents := &events.AppspaceStatusEvents{}
	appspaceLogEvents := &events.AppspaceLogEvents{}
	migrationStatusEvents := &events.MigrationJobStatusEvents{}

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
		Validator: validator,
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
		_, dsErr := cliHandlers.AddAdmin()
		if dsErr != nil {
			fmt.Println(dsErr)
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

	contactModel := contactmodel.ContactModel{
		DB: db}
	contactModel.PrepareStatements()

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

	appspaceLogger := &appspacelogger.AppspaceLogger{
		AppspaceModel:     appspaceModel,
		AppspaceLogEvents: appspaceLogEvents,
		Config:            runtimeConfig}
	appspaceLogger.Init()

	appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
		Config:        runtimeConfig,
		Validator:     validator,
		AppspaceModel: appspaceModel}
	appspaceMetaDb.Init()

	appspaceInfoModels := &appspacemetadb.AppspaceInfoModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb}
	appspaceInfoModels.Init()

	migrationJobModel := &migrationjobmodel.MigrationJobModel{
		DB: db}
	migrationJobModel.PrepareStatements()

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
		MigrationJobModel:  migrationJobModel,
		MigrationEvents:    migrationStatusEvents}

	// auth
	authenticator := &authenticator.Authenticator{
		CookieModel: cookieModel,
		Config:      runtimeConfig}

	appspaceLogin := &appspacelogin.AppspaceLogin{}
	appspaceLogin.Start()

	liveDataRoutes := &userroutes.LiveDataRoutes{
		//JobController:     migrationJobCtl,
		MigrationJobModel: migrationJobModel,
		Authenticator:     authenticator}
	liveDataRoutes.Init()

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

		liveDataRoutes.Stop()

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
		MigrationJobs:        migrationJobCtl,
		MigrationJobsEvents:  migrationStatusEvents,
		AppspaceFilesEvents:  appspaceFilesEvents,
		AppspacePausedEvent:  appspacePausedEvent,
		AppspaceStatusEvents: appspaceStatusEvents,
	}
	appspaceStatus.Init()

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
		AppspaceLogin:       appspaceLogin,
		Validator:           validator}

	adminRoutes := &userroutes.AdminRoutes{
		UserModel:           userModel,
		SettingsModel:       settingsModel,
		UserInvitationModel: userInvitationModel,
		Validator:           validator}

	applicationRoutes := &userroutes.ApplicationRoutes{
		AppGetter:     appGetter,
		AppFilesModel: appFilesModel,
		AppModel:      appModel,
		AppspaceModel: appspaceModel}

	appspaceUserRoutes := &userroutes.AppspaceRoutes{
		AppspaceFilesModel:     appspaceFilesModel,
		AppspaceModel:          appspaceModel,
		AppspaceMetaDB:         appspaceMetaDb,
		MigrationJobModel:      migrationJobModel,
		MigrationJobController: migrationJobCtl,
		AppModel:               appModel}

	appspaceStatusTwine := &twineservices.AppspaceStatusService{
		AppspaceModel:        appspaceModel,
		AppspaceStatus:       appspaceStatus,
		AppspaceStatusEvents: appspaceStatusEvents,
	}

	userRoutes := &userroutes.UserRoutes{
		AuthRoutes:          authRoutes,
		AdminRoutes:         adminRoutes,
		ApplicationRoutes:   applicationRoutes,
		AppspaceRoutes:      appspaceUserRoutes,
		LiveDataRoutes:      liveDataRoutes,
		AppspaceStatusTwine: appspaceStatusTwine,
		UserModel:           userModel,
		Views:               views,
		Validator:           validator}

	appspaceDB := &appspacedb.AppspaceDB{
		Config: runtimeConfig,
	}
	appspaceDB.Init()

	appspaceRouteModels := &appspacemetadb.AppspaceRouteModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb,
		Validator:      validator}
	appspaceRouteModels.Init()

	appspaceUserModels := &appspacemetadb.AppspaceUserModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb,
		Validator:      validator,
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
		UserRoutes:     userRoutes,
		AppspaceRouter: appspaceRouter,
		Metrics:        &m}

	fmt.Println("starting server")

	// start things up
	migrationJobCtl.Start() // TODO: add delay, maybe set in runtimeconfig for first job to run

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

	fmt.Println("Leaving main func")
}
