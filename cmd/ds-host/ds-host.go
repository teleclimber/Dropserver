package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"sync"
	"syscall"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/runtimeconfig"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
	"github.com/teleclimber/DropServer/cmd/ds-host/trusted"
	"github.com/teleclimber/DropServer/cmd/ds-host/server"
	"github.com/teleclimber/DropServer/cmd/ds-host/userroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspaceroutes"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appmodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacemodel"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

var configFlag = flag.String("config", "", "use this JSON confgiuration file")

var hostAppSpace = map[string]string{}	// this stuff is DB model
var appSpaceApp = map[string]string{}

func main() {
	flag.Parse()

	runtimeConfig := runtimeconfig.Load(*configFlag)

	record.Init(runtimeConfig)	// ok, but that's not how we should do it.
	// ^^ preserve this for metrics, but get rid of it eventually

	logger := record.NewLogClient(runtimeConfig)

	logger.Log(domain.INFO, nil, "ds-host is starting")

	// models
	appModel := appmodel.NewAppModel()
	appspaceModel := appspacemodel.NewAppspaceModel()

	generateHostAppSpaces(100, appModel, appspaceModel, logger)

	var initWg sync.WaitGroup
	initWg.Add(2)

	trustedClient := trusted.RPCClient{}

	trustedManager := trusted.Trusted{
		RPCClient: &trustedClient }

	trustedManager.Init(&initWg)

	sM := sandbox.Manager{
		Config: runtimeConfig,
		Logger: logger }

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Caught signal, quitting.", sig)
		pprof.StopCPUProfile()

		var stopWg sync.WaitGroup
		stopWg.Add(1)
		trustedManager.Stop(&stopWg)
		sM.StopAll()	//should take a waitgroup
		stopWg.Wait()
		
		fmt.Println("All sandbox stopped")
		os.Exit(0) //temporary I suppose. need to cleanly shut down all the things.
	}()

	go sM.Init(&initWg)

	initWg.Wait()

	fmt.Println("Main after constainers start")

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

	// Create routes
	applicationRoutes := &userroutes.ApplicationRoutes{
		TrustedClient: &trustedClient,
		AppModel: appModel,
		Logger: logger }
	userRoutes := &userroutes.UserRoutes{
		ApplicationRoutes: applicationRoutes,
		Logger: logger }

	dropserverASRoutes := &appspaceroutes.DropserverRoutes{}
	appspaceRoutes := &appspaceroutes.AppspaceRoutes{
		AppModel:	appModel,
		AppspaceModel: appspaceModel,
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

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

	fmt.Println("Leaving main func")
}

func generateHostAppSpaces(n int, am domain.AppModel, asm domain.AppspaceModel, logger domain.LogCLientI) {
	logger.Log(domain.WARN, nil, "Generating app spaces and apps:"+strconv.Itoa(n))
	var appSpace, app string
	for i := 1; i <= n; i++ {
		appSpace = fmt.Sprintf("as%d", i)
		app = fmt.Sprintf("app%d", i)
		am.Create( &domain.App{Name:app})
		asm.Create( &domain.Appspace{Name:appSpace, AppName: app})
	}
}

