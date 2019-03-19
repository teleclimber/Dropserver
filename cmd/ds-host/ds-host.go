package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
	"github.com/teleclimber/DropServer/cmd/ds-host/trusted"
	"github.com/teleclimber/DropServer/internal/timetrack"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

// TODO
// - check yourself on concurrency issues
// - detect failed states in sandbox and quarantine them

var hostAppSpace = map[string]string{}
var appSpaceApp = map[string]string{}

func main() {
	flag.Parse()

	record.Init()

	record.Log(record.INFO, nil, "ds-host is starting")

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

	// Proxy:
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, &sM)
	})
	if err := http.ListenAndServe(":3000", nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Leaving main func")
}

func generateHostAppSpaces(n int) {
	record.Log(record.WARN, nil, "Generating app spaces and apps:"+strconv.Itoa(n))
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
func handleRequest(oRes http.ResponseWriter, oReq *http.Request, sM *sandbox.Manager) {
	defer record.HostHandleReq(time.Now())

	host := strings.Split(oReq.Host, ":")[0] //in case the port was included in host
	appSpace, ok := hostAppSpace[host]
	if !ok {
		//this is a request error
		fmt.Println("app space not found for host", host) //request id, host
		oRes.WriteHeader(404)
		return
	}
	app, ok := appSpaceApp[appSpace]
	if !ok {
		fmt.Println("app not found for app space", appSpace) //request id, appspace
		oRes.WriteHeader(500)
		return
	}

	// then gotta run through auth and route...

	fmt.Println("in request handler", host, appSpace, app)

	getTime := time.Now()

	sandboxChan := sM.GetForAppSpace(app, appSpace)
	sb := <-sandboxChan

	timetrack.Track(getTime, "getting sandbox "+appSpace+" c"+sb.Name)

	reqTask := sb.TaskBegin()

	header := cloneHeader(oReq.Header)
	//header["ds-user-id"] = []string{"teleclimber"}
	header["app-space-script"] = []string{"hello.js"}
	header["app-space-fn"] = []string{"hello"}

	cReq, err := http.NewRequest(oReq.Method, sb.Address, oReq.Body)
	if err != nil {
		fmt.Println("http.NewRequest error", sb.Name, oReq.Method, sb.Address, err)
		os.Exit(1)
	}

	cReq.Header = header

	cRes, err := sb.Transport.RoundTrip(cReq)
	if err != nil {
		fmt.Println("sb.Transport.RoundTrip(cReq) error", sb.Name, err)
		os.Exit(1)
	}

	// futz around with headers
	copyHeader(oRes.Header(), cRes.Header)

	oRes.WriteHeader(cRes.StatusCode)

	io.Copy(oRes, cRes.Body)

	cRes.Body.Close()

	sb.TaskEnd(reqTask)

	sb.LogClient.Log(record.INFO, map[string]string{
		"app-space": appSpace, "app": app},
		"Request handled")
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
