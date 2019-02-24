package main

import (
	"fmt"
	"github.com/teleclimber/DropServer/cmd/ds-host/containers"
	"github.com/teleclimber/DropServer/cmd/ds-host/trusted"
	"github.com/teleclimber/DropServer/internal/timetrack"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// TODO
// - manage containers pool
// - track when a container is "in use" by app space
// - proper shutdown of things as much as possible
//   ..including shutting down sanboxes and deleting them (for now)
// - check yourself on concurrency issues
// - start and shutdown containers
// - detect failed states in containers and quarantine them

var hostAppSpace = map[string]string{
	"as1.teleclimber.dropserver.org": "as1"}

var appSpaceApp = map[string]string{
	"as1": "app1"}

func main() {
	fmt.Println("ds-host is starting")

	trusted.Init()

	cM := containers.Manager{}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Caught signal, quitting.", sig)
		cM.StopAll()
		fmt.Println("All containers stopped")
		os.Exit(0) //temporary I suppose. need to cleanly shut down all the things.
	}()

	cM.Init()
	//cM.StartContainer()

	fmt.Println("Main after container start")

	// Proxy:
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, &cM)
	})
	if err := http.ListenAndServe(":3000", nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

/////////////////////////////////////////////////
// proxy
func handleRequest(oRes http.ResponseWriter, oReq *http.Request, cM *containers.Manager) {
	defer timetrack.Track(time.Now(), "handleRequest")
	host := oReq.Host
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

	// Here, if we need a container, then find one, commit one, recycle one, or start one.
	container, ok := cM.GetForAppSpace(app, appSpace)
	if !ok {
		fmt.Println("no containers available", appSpace)
		oRes.WriteHeader(503)
		return
	}

	// here we should start a timer
	// and we sould mark the container as processing a request.
	// OK, how? -> container holds array of ongoing requests
	// when a request ends you can remove it off the array and stash it in a log or whatever
	// ..along with requet type and return status, time taken, etc...
	// reqTask := task{}
	// container.appSpaceSession.tasks = append(container.appSpaceSession.tasks, &reqTask)
	// Here I'm thinking we can make this a method of container
	reqTask := container.StartTask()

	//container.resetTimer() // this should be managed more carfully.
	// maybe it should be paused (or stopped)
	// and restarted when the container is no longer in use
	// And in any case we shouldn't do things this way.
	// -> we should just maintain a steady pool of "Ready" containers

	header := cloneHeader(oReq.Header)
	//header["ds-user-id"] = []string{"teleclimber"}
	header["app-space-script"] = []string{"hello.js"}
	header["app-space-fn"] = []string{"hello"}

	cReq, err := http.NewRequest(oReq.Method, container.Address, oReq.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cReq.Header = header

	cRes, err := container.Transport.RoundTrip(cReq)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// futz around with headers
	copyHeader(oRes.Header(), cRes.Header)

	oRes.WriteHeader(cRes.StatusCode)

	io.Copy(oRes, cRes.Body)

	cRes.Body.Close()

	reqTask.Finished = true

	container.TouchSession()
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
