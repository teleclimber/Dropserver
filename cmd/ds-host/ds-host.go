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
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
	"github.com/teleclimber/DropServer/cmd/ds-host/trusted"
	"github.com/teleclimber/DropServer/cmd/ds-host/server"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

// TODO
// - check yourself on concurrency issues
// - detect failed states in sandbox and quarantine them

var hostAppSpace = map[string]string{}	// this stuff is DB model
var appSpaceApp = map[string]string{}

func main() {
	flag.Parse()

	record.Init()

	record.Log(domain.INFO, nil, "ds-host is starting")

	generateHostAppSpaces(100)

	var initWg sync.WaitGroup
	initWg.Add(2)

	go trusted.Init(&initWg)

	sM := sandbox.Manager{}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Caught signal, quitting.", sig)
		pprof.StopCPUProfile()
		sM.StopAll()
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

	// Create server. pass it sandbox manager I suppose?
	// Or you actually want to set these dependencies by setting them directly on the struct.
	server := &server.Server{
		SandboxManager: &sM,
		Metrics: &m,
		HostAppSpace: &hostAppSpace,
		AppSpaceApp: &appSpaceApp}

	server.Start()
	// ^^ this blocks as it is. Obviously not what what we want.

	fmt.Println("Leaving main func")
}

func generateHostAppSpaces(n int) {
	record.Log(domain.WARN, nil, "Generating app spaces and apps:"+strconv.Itoa(n))
	var host, appSpace, app string
	for i := 1; i <= n; i++ {
		host = fmt.Sprintf("as%d.teleclimber.dropserver.develop", i)
		appSpace = fmt.Sprintf("as%d", i)
		app = fmt.Sprintf("app%d", i)
		hostAppSpace[host] = appSpace
		appSpaceApp[appSpace] = app
	}
}

