package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspaceroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/authenticator"
	"github.com/teleclimber/DropServer/cmd/ds-host/clihandlers"
	"github.com/teleclimber/DropServer/cmd/ds-host/database"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrateappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacefilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacemodel"
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
	"github.com/teleclimber/DropServer/cmd/ds-host/userroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/views"
	"github.com/teleclimber/DropServer/internal/stdinput"
	"github.com/teleclimber/DropServer/internal/validator"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

var configFlag = flag.String("config", "", "use this JSON confgiuration file")

var migrateFlag = flag.Bool("migrate", false, "Set migrate flag to migrate db as needed.")

var addAdminFlag = flag.Bool("add-admin", false, "add an admin")

func main() {
	//startServer := true	// currnetly actually not used.

	flag.Parse()

	runtimeConfig := runtimeconfig.Load(*configFlag)

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

	logger := record.NewLogClient(runtimeConfig) // we should start logger before migration step, and log migrations

	validator := &validator.Validator{}
	validator.Init()

	stdInput := &stdinput.StdInput{}

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

	logger.Log(domain.INFO, nil, "ds-host is starting")

	cookieModel := &cookiemodel.CookieModel{
		DB: db}
	cookieModel.PrepareStatements()

	appFilesModel := &appfilesmodel.AppFilesModel{
		Config: runtimeConfig}

	appModel := &appmodel.AppModel{
		DB: db}
	appModel.PrepareStatements()

	appspaceFilesModel := &appspacefilesmodel.AppspaceFilesModel{
		Config: runtimeConfig}

	appspaceModel := &appspacemodel.AppspaceModel{
		DB: db}
	appspaceModel.PrepareStatements()

	migrationJobModel := &migrationjobmodel.MigrationJobModel{
		DB: db}
	migrationJobModel.PrepareStatements()

	sandboxManager := &sandbox.Manager{
		Config: runtimeConfig}

	migrationJobCtl := &migrateappspace.JobController{
		AppspaceModel:     appspaceModel,
		AppModel:          appModel,
		SandboxManager:    sandboxManager,
		MigrationJobModel: migrationJobModel,
		Config:            runtimeConfig,
		Logger:            logger}

	// auth
	authenticator := &authenticator.Authenticator{
		CookieModel: cookieModel,
		Config:      runtimeConfig}

	liveDataRoutes := &userroutes.LiveDataRoutes{
		JobController:     migrationJobCtl,
		MigrationJobModel: migrationJobModel,
		Authenticator:     authenticator,
		Logger:            logger}
	liveDataRoutes.Init()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Caught signal, quitting.", sig)
		pprof.StopCPUProfile()

		sandboxManager.StopAll()
		fmt.Println("All sandbox stopped")

		migrationJobCtl.Stop() // We should make all stop things async and have a waitgroup for them.

		liveDataRoutes.Stop()

		os.Exit(0)
	}()

	sandboxManager.Init()

	fmt.Println("Main after sandbox manager start")

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

	migrationJobCtl.Start() // TODO: add delay, maybe set in runtimeconfig for first job to run

	// Create proxy
	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: sandboxManager,
		Logger:         logger,
		Metrics:        &m}

	// Views
	views := &views.Views{
		Logger: logger,
		Config: runtimeConfig}
	views.PrepareTemplates()

	// Create routes
	authRoutes := &userroutes.AuthRoutes{
		Views:               views,
		SettingsModel:       settingsModel,
		UserModel:           userModel,
		UserInvitationModel: userInvitationModel,
		Authenticator:       authenticator,
		Validator:           validator}

	adminRoutes := &userroutes.AdminRoutes{
		UserModel:           userModel,
		SettingsModel:       settingsModel,
		UserInvitationModel: userInvitationModel,
		Validator:           validator,
		Logger:              logger}

	applicationRoutes := &userroutes.ApplicationRoutes{
		AppFilesModel: appFilesModel,
		AppModel:      appModel,
		AppspaceModel: appspaceModel,
		Logger:        logger}

	appspaceUserRoutes := &userroutes.AppspaceRoutes{
		AppspaceFilesModel:     appspaceFilesModel,
		AppspaceModel:          appspaceModel,
		MigrationJobModel:      migrationJobModel,
		MigrationJobController: migrationJobCtl,
		AppModel:               appModel}

	userRoutes := &userroutes.UserRoutes{
		Authenticator:     authenticator,
		AuthRoutes:        authRoutes,
		AdminRoutes:       adminRoutes,
		ApplicationRoutes: applicationRoutes,
		AppspaceRoutes:    appspaceUserRoutes,
		LiveDataRoutes:    liveDataRoutes,
		UserModel:         userModel,
		Views:             views,
		Validator:         validator,
		Logger:            logger}

	appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
		Config:    runtimeConfig,
		Validator: validator}
	appspaceRouteModels := &appspacemetadb.AppspaceRouteModels{
		Config:         runtimeConfig,
		AppspaceMetaDB: appspaceMetaDb,
		Validator:      validator}

	appspaceRoutesV0 := &appspaceroutes.V0{
		AppspaceRouteModels: appspaceRouteModels,
		DropserverRoutes:    &appspaceroutes.DropserverRoutesV0{},
		SandboxProxy:        sandboxProxy,
		Logger:              logger}

	appspaceRoutes := &appspaceroutes.AppspaceRoutes{
		AppModel:      appModel,
		AppspaceModel: appspaceModel,
		V0:            appspaceRoutesV0}

	revServices := &domain.ReverseServices{
		Routes: appspaceRouteModels,
	}
	sandboxManager.Services = revServices

	// Create server.
	server := &server.Server{
		Config:         runtimeConfig,
		UserRoutes:     userRoutes,
		AppspaceRoutes: appspaceRoutes,
		Metrics:        &m,
		Logger:         logger}

	fmt.Println("starting server")

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

	fmt.Println("Leaving main func")
}

// func generateHostAppSpaces(n int, am domain.AppModel, asm domain.AppspaceModel, logger domain.LogCLientI) {
// 	logger.Log(domain.WARN, nil, "Generating app spaces and apps:"+strconv.Itoa(n))
// 	var appSpace, app string
// 	for i := 1; i <= n; i++ {
// 		appSpace = fmt.Sprintf("as%d", i)
// 		app = fmt.Sprintf("app%d", i)
// 		//am.Create( &domain.App{Name:app})
// 		//asm.Create( &domain.Appspace{Name:appSpace, AppName: app})
// 	}
// }
