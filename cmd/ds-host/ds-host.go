package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/runtimeconfig"
	"github.com/teleclimber/DropServer/cmd/ds-host/database"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
	"github.com/teleclimber/DropServer/cmd/ds-host/authenticator"
	"github.com/teleclimber/DropServer/cmd/ds-host/views"
	"github.com/teleclimber/DropServer/cmd/ds-host/server"
	"github.com/teleclimber/DropServer/cmd/ds-host/userroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspaceroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacemodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/asroutesmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/usermodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/cookiemodel"
	"github.com/teleclimber/DropServer/internal/validator"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

var configFlag = flag.String("config", "", "use this JSON confgiuration file")

var migrateFlag = flag.Bool("migrate", false, "Set migrate flag to migrate db as needed.")

func main() {
	flag.Parse()

	runtimeConfig := runtimeconfig.Load(*configFlag)

	dbManager := &database.Manager{
		Config: runtimeConfig }

	db, err := dbManager.Open()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	migrator := &migrate.Migrator{
		OrderedSteps: migrate.OrderedSteps,
		StringSteps: migrate.StringSteps,
		Config: runtimeConfig,
		DBManager: dbManager }

	if *migrateFlag {
		
		dsErr := migrator.Migrate("")
		if dsErr != nil {
			fmt.Println("Error Migrating", dsErr.PublicString(), dsErr.ExtraMessage())
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

	record.Init(runtimeConfig)	// ok, but that's not how we should do it.
	// ^^ preserve this for metrics, but get rid of it eventually

	logger := record.NewLogClient(runtimeConfig)

	logger.Log(domain.INFO, nil, "ds-host is starting")

	validator := &validator.Validator{}
	validator.Init()

	// models
	userModel := &usermodel.UserModel{
		DB: db,
		Logger: logger }
	userModel.PrepareStatements()

	cookieModel := &cookiemodel.CookieModel{
		DB: db,
		Logger: logger }
	cookieModel.PrepareStatements()

	appFilesModel := &appfilesmodel.AppFilesModel{
		Config: runtimeConfig,
		Logger: logger}

	appModel := &appmodel.AppModel{
		DB: db,
		Logger: logger }
	appModel.PrepareStatements()

	appspaceModel := &appspacemodel.AppspaceModel{
		DB: db,
		Logger: logger }
	appspaceModel.PrepareStatements()

	// appspaceroutesmodel is questionable because it loads the routes from the files, yet we have a model that reads from there?
	asRoutesModel := &asroutesmodel.ASRoutesModel{
		AppFilesModel: appFilesModel,	// temporary!
		Logger: logger }

	sM := sandbox.Manager{
		Config: runtimeConfig,
		Logger: logger }

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Caught signal, quitting.", sig)
		pprof.StopCPUProfile()

		sM.StopAll()
		fmt.Println("All sandbox stopped")

		os.Exit(0)
	}()

	sM.Init()

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

	// Create proxy
	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: &sM,
		Logger: logger,
		Metrics: &m	}

	// auth
	authenticator := &authenticator.Authenticator{
		CookieModel: cookieModel,
		Config: runtimeConfig }

	// Views 
	views := &views.Views{
		Logger: logger,
		Config: runtimeConfig }
	views.PrepareTemplates()

	// Create routes
	authRoutes := &userroutes.AuthRoutes{
		Views: views,
		UserModel: userModel,
		Authenticator: authenticator,
		Validator: validator}

	adminRoutes := &userroutes.AdminRoutes{
		UserModel: userModel,
		Logger: logger}

	applicationRoutes := &userroutes.ApplicationRoutes{
		AppFilesModel: appFilesModel,
		AppModel: appModel,
		AppspaceModel: appspaceModel,
		Logger: logger }

	appspaceUserRoutes := &userroutes.AppspaceRoutes{
		AppspaceModel: appspaceModel,
		AppModel: appModel,
		Logger: logger }
		
	userRoutes := &userroutes.UserRoutes{
		Authenticator: authenticator,
		AuthRoutes: authRoutes,
		AdminRoutes: adminRoutes,
		ApplicationRoutes: applicationRoutes,
		AppspaceRoutes: appspaceUserRoutes,
		UserModel: userModel,
		Views: views,
		Validator: validator,
		Logger: logger }

	dropserverASRoutes := &appspaceroutes.DropserverRoutes{}
	appspaceRoutes := &appspaceroutes.AppspaceRoutes{
		AppModel:	appModel,
		AppspaceModel: appspaceModel,
		ASRoutesModel: asRoutesModel,
		DropserverRoutes: dropserverASRoutes,
		SandboxProxy: sandboxProxy,
		Logger: logger }

	// Create server.
	server := &server.Server{
		Config: runtimeConfig,
		UserRoutes: userRoutes,
		AppspaceRoutes: appspaceRoutes,
		Metrics: &m,
		Logger: logger }

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

