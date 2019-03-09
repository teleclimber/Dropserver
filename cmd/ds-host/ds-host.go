package main

import (
	"flag"
	"fmt"
	"github.com/teleclimber/DropServer/cmd/ds-host/containers"
	"github.com/teleclimber/DropServer/cmd/ds-host/trusted"
	"github.com/teleclimber/DropServer/internal/timetrack"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

// TODO
// - check yourself on concurrency issues
// - detect failed states in containers and quarantine them

var hostAppSpace = map[string]string{}
var appSpaceApp = map[string]string{}

func main() {
	flag.Parse()

	fmt.Println("ds-host is starting")

	generateHostAppSpaces(100)
	fmt.Println(hostAppSpace, appSpaceApp)

	var initWg sync.WaitGroup
	initWg.Add(2)

	go trusted.Init(&initWg)

	cM := containers.Manager{}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Caught signal, quitting.", sig)
		pprof.StopCPUProfile()
		cM.StopAll()
		fmt.Println("All containers stopped")
		os.Exit(0) //temporary I suppose. need to cleanly shut down all the things.
	}()

	go cM.Init(&initWg)

	initWg.Wait()

	fmt.Println("Main after container start")

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

	// Proxy:
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, &cM)
	})
	if err := http.ListenAndServe(":3000", nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Leaving main func")
}

func generateHostAppSpaces(n int) {
	var host, appSpace, app string
	for i := 1; i <= n; i++ {
		host = fmt.Sprintf("as%d.teleclimber.dropserver.develop", i)
		appSpace = fmt.Sprintf("as%d", i)
		app = fmt.Sprintf("app%d", i)
		hostAppSpace[host] = appSpace
		appSpaceApp[appSpace] = app
	}
}

/////////////////////////////////////////////////
// proxy
func handleRequest(oRes http.ResponseWriter, oReq *http.Request, cM *containers.Manager) {
	defer timetrack.Track(time.Now(), "handleRequest")

	host := strings.Split(oReq.Host, ":")[0] //in case the port was included in host
	appSpace, ok := hostAppSpace[host]
	if !ok {
		//this is a request error
		fmt.Println("app space not found for host", host)
		oRes.WriteHeader(404)
		return
	}
	app, ok := appSpaceApp[appSpace]
	if !ok {
		fmt.Println("app not found for app space", appSpace)
		oRes.WriteHeader(500)
		return
	}

	// then gotta run through auth and route...

	fmt.Println("in request handler", host, appSpace, app)

	getTime := time.Now()

	sandboxChan := cM.GetForAppSpace(app, appSpace)
	container := <-sandboxChan

	timetrack.Track(getTime, "getting sandbox "+appSpace+" c"+container.Name)

	reqTask := container.TaskBegin()

	header := cloneHeader(oReq.Header)
	//header["ds-user-id"] = []string{"teleclimber"}
	header["app-space-script"] = []string{"hello.js"}
	header["app-space-fn"] = []string{"hello"}

	cReq, err := http.NewRequest(oReq.Method, container.Address, oReq.Body)
	if err != nil {
		fmt.Println("http.NewRequest error", container.Name, oReq.Method, container.Address, err)
		os.Exit(1)
	}

	cReq.Header = header

	cRes, err := container.Transport.RoundTrip(cReq)
	if err != nil {
		fmt.Println("container.Transport.RoundTrip(cReq) error", container.Name, err)
		os.Exit(1)
	}

	// futz around with headers
	copyHeader(oRes.Header(), cRes.Header)

	oRes.WriteHeader(cRes.StatusCode)

	io.Copy(oRes, cRes.Body)

	cRes.Body.Close()

	container.TaskEnd(reqTask)
}

// From https://golang.org/src/net/http/httputil/reverseproxy.go
// ..later we can pick and choose the headers and values we forward to the app
func cloneHeader(h http.Header) http.Header {
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
