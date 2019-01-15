package main

import (
	"fmt"
	"github.com/teleclimber/DropServer/cmd/ds-host/mountappspace"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// TODO
// - manage containers pool
// - track when a container is "in use" by app space
// - proper shutdown of things as much as possible
// - check yourself on concurrency issues
// - start and shutdown containers
// - detect failed states in containers and quarantine them

type containersManager struct {
	containers []*container
}

type container struct {
	name            string
	status          string
	address         string
	appSpaceID      string
	recycleListener *recycleListener
	reverseListener *reverseListener
	statusSub       map[string][]chan bool
	//timer            *time.Timer //zap this
	transport       http.RoundTripper
	appSpaceSession appSpaceSession
}
type appSpaceSession struct {
	tasks      []*task
	lastActive time.Time
}
type task struct {
	finished bool //build up with start time, elapsed and any other metadata
}

type recycleListener struct {
	ln     *net.Listener
	conn   *net.Conn
	msgSub map[string]chan bool
}
type reverseListener struct { //do we really need two distinct types here?
	ln       *net.Listener
	conn     *net.Conn
	msgSub   map[string]chan bool
	sockPath string
}

var hostAppSpace = map[string]string{
	"as1.teleclimber.dropserver.org": "as1"}

var appSpaceApp = map[string]string{
	"as1": "app1"}

func main() {
	fmt.Println("ds-host is starting")

	cM := containersManager{}
	cM.startContainer()

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
func handleRequest(oRes http.ResponseWriter, oReq *http.Request, cM *containersManager) {
	defer timeTrack(time.Now(), "handleRequest")
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
	container, ok := cM.getForAppSpace(app, appSpace)
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
	reqTask := task{}
	container.appSpaceSession.tasks = append(container.appSpaceSession.tasks, &reqTask)

	//container.resetTimer() // this should be managed more carfully.
	// maybe it should be paused (or stopped)
	// and restarted when the container is no longer in use
	// And in any case we shouldn't do things this way.
	// -> we should just maintain a steady pool of "Ready" containers

	header := cloneHeader(oReq.Header)
	//header["ds-user-id"] = []string{"teleclimber"}
	header["app-space-script"] = []string{"hello.js"}
	header["app-space-fn"] = []string{"hello"}

	cReq, err := http.NewRequest(oReq.Method, container.address, oReq.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cReq.Header = header

	cRes, err := container.transport.RoundTrip(cReq)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// futz around with headers
	copyHeader(oRes.Header(), cRes.Header)

	oRes.WriteHeader(cRes.StatusCode)

	io.Copy(oRes, cRes.Body)

	cRes.Body.Close()

	reqTask.finished = true

	container.appSpaceSession.lastActive = time.Now()
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

/////////////////////////////////////////
// container management
func (cM *containersManager) startContainer() {
	newContainer := container{
		name:       "c7",
		status:     "starting",
		address:    "http://10.140.177.203:3030",
		appSpaceID: "",
		statusSub:  make(map[string][]chan bool)}

	cM.containers = append(cM.containers, &newContainer)
	// ^^ you want it in there early so that you don't start another one?

	newContainer.recycleListener = newRecycleListener("c7", newContainer.onRecyclerMsg)

	newContainer.recycle()
}

func (cM *containersManager) getForAppSpace(app string, appSpace string) (retContainer *container, ok bool) {
	// first look to see if there is a container that is already commited
	for _, c := range cM.containers {
		if c.appSpaceID == appSpace {
			retContainer = c
			ok = true //note that it might not be ready!
			c.waitFor("committed")
			break
		}
	}

	if !ok {
		// now see if there is a container we can commit
		for _, c := range cM.containers {
			if c.status == "ready" && c.appSpaceID == "" {
				c.commit(app, appSpace)
				retContainer = c
				ok = true
				break
			}
		}
	}

	if !ok {
		// now see if there is a container we can commit
		for _, c := range cM.containers {
			if c.status == "starting" || c.status == "recycling" {
				// can we have a c.reserve?
				fmt.Println("reserving container that is starting or recycling")
				c.appSpaceID = appSpace
				c.waitFor("ready")
				c.commit(app, appSpace)
				retContainer = c
				ok = true
				break
			}
		}
	}

	// next we can also try to recycle a container or start a new one
	if !ok {
		// now see if there is a container we can recycle
		var candidate *container
		for _, c := range cM.containers {
			if c.status == "committed" && !c.isTiedUp() {
				if candidate == nil {
					candidate = c
				} else if candidate.appSpaceSession.lastActive.After(c.appSpaceSession.lastActive) {
					candidate = c
				}
				// loop over tasks and see the latest finish time
				// recycle the container with the longest idle state
				// or other option is to keep a running tally for each container?

				fmt.Println("reserving container that is starting or recycling")
				c.appSpaceID = appSpace
				c.waitFor("ready")
				c.commit(app, appSpace)
				retContainer = c
				ok = true
				break
			}
		}

		if candidate != nil {
			// go ahead and recycle this one
			candidate.recycle()
			candidate.commit(app, appSpace)
			retContainer = candidate
			ok = true
		}
	}

	return
}

// manager methods like getForAppSpace, ...

// containers:
func (c *container) recycle() {
	fmt.Println("starting recycle")
	defer timeTrack(time.Now(), "recycle")

	c.status = "recycling"
	c.appSpaceID = ""
	c.appSpaceSession = appSpaceSession{lastActive: time.Now()}

	// close all connections (they should all be idle if we are recycling)
	transport, ok := c.transport.(*http.Transport)
	if !ok {
		fmt.Println("did not find transport, sorry")
	} else {
		transport.CloseIdleConnections()
	}

	// stop reverse channel? Or will it stop itself with kill?
	if c.reverseListener != nil {
		c.reverseListener.close()
	}

	c.recycleListener.send("kill")
	c.recycleListener.waitFor("kild")

	mountappspace.UnMount(c.name)

	c.reverseListener = newReverseListener("c7", c.onReverseMsg)
	c.recycleListener.send("run")
	c.reverseListener.waitFor("hi")

	c.status = "ready"

	c.waitForDone("ready")
}
func (c *container) commit(app, appSpace string) {
	defer timeTrack(time.Now(), "commit")

	c.appSpaceID = appSpace

	c.status = "committing"

	mountappspace.Mount(app, appSpace, c.name)

	c.transport = http.DefaultTransport

	c.status = "committed"
	c.waitForDone("commited")

	// ^^ I suspect we are going to get random glitches due to concurrency.
	// Probably need to lock something somewhere. Not sure what though.

	// duration, err := time.ParseDuration("5s")
	// if err != nil {
	// 	fmt.Println("error parsing duration")
	// }
	// c.timer = time.AfterFunc(duration, c.recycle)
}

func (c *container) isTiedUp() (tiedUp bool) {
	for _, task := range c.appSpaceSession.tasks {
		if !task.finished {
			tiedUp = true
			break
		}
	}
	return
}

// func (c *container) resetTimer() {
// 	// basically just reset the timer before self-recycle
// 	duration, err := time.ParseDuration("5s")
// 	if err != nil {
// 		fmt.Println("error parsing duration")
// 	}
// 	t := c.timer
// 	if !t.Stop() {
// 		<-t.C
// 	}
// 	t.Reset(duration)
// }
func (c *container) waitFor(status string) {
	if c.status == status {
		return
	}
	fmt.Println("waiting for container status", status)

	if _, ok := c.statusSub[status]; !ok {
		c.statusSub[status] = []chan bool{}
	}
	statusMet := make(chan bool)
	c.statusSub[status] = append(c.statusSub[status], statusMet)
	<-statusMet
	delete(c.statusSub, status)
}
func (c *container) waitForDone(status string) {
	if subs, ok := c.statusSub[status]; ok {
		for _, wCh := range subs {
			wCh <- true
		}
		c.statusSub[status] = []chan bool{}
	}
	// then gotta empty / reset the channel.
	// though probably lock the array?
}
func (c *container) onRecyclerMsg(msg string) {
	//fmt.Println("onRecyclerMsg", msg, c.name)
}
func (c *container) onReverseMsg(msg string) {
	//fmt.Println("onReverseMsg", msg, c.name)
}

// close function which terminates all the things?

func newRecycleListener(containerName string, msgCb func(msg string)) *recycleListener {
	recyclerSockPath := "/home/developer/container_sockets/" + containerName + "/recycle.sock"
	err := os.Remove(recyclerSockPath)
	if err != nil {
		fmt.Println(err)
		//os.Exit(1)	// don't exit. if file didn't exist it errs.
	}
	listener, err := net.Listen("unix", recyclerSockPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//defer revListener.Close()	// this looks like it will close the server at end of function
	// ^^ so instead do it when we kill the server

	err = os.Chmod(recyclerSockPath, 0777) //temporary until we figure out our users scenario
	if err != nil {
		fmt.Println(err)
	}

	rl := recycleListener{ln: &listener, msgSub: make(map[string]chan bool)}

	recConn, err := listener.Accept() // I think this blocks until aconn shows up?
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//fmt.Println("recycle connection accepted")

	rl.conn = &recConn

	hi := make(chan bool)
	rl.msgSub["hi"] = hi

	go func() {
		p := make([]byte, 4)
		for {
			n, err := recConn.Read(p)
			if err != nil {
				if err == io.EOF {
					fmt.Println("got EOF from recycle conn", string(p[:n]))
					break
				}
				fmt.Println(err)
				os.Exit(1)
			}
			command := string(p[:n])
			//fmt.Println("recycle listener got message", command)
			if subChan, ok := rl.msgSub[command]; ok {
				subChan <- true
			}

			// not sure if this is useful after all
			msgCb(command)
		}
	}()

	<-hi
	delete(rl.msgSub, "hi")

	return &rl
}
func (rl *recycleListener) send(msg string) { // return err?
	fmt.Println("Sending message", msg)
	_, err := (*rl.conn).Write([]byte(msg))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func (rl *recycleListener) waitFor(msg string) {
	fmt.Println("waiting for", msg)
	done := make(chan bool)
	rl.msgSub[msg] = done
	<-done
	//fmt.Println("DONE waiting for", msg)
	delete(rl.msgSub, msg)
}
func (rl *recycleListener) close() {
	//conn.end() or some such
}

// reverse Listener
func newReverseListener(containerName string, msgCb func(msg string)) *reverseListener {
	sockPath := "/home/developer/container_mount/" + containerName + "/reverse.sock"
	err := os.Remove(sockPath)
	if err != nil {
		fmt.Println(err)
		//os.Exit(1)	// don't exit. if file didn't exist it errs.
	}
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rl := reverseListener{ln: &listener, msgSub: make(map[string]chan bool), sockPath: sockPath}

	go func(ln net.Listener) {
		revConn, err := ln.Accept() // I think this blocks until aconn shows up?
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("reverse connection accepted")

		rl.conn = &revConn

		go func(rc net.Conn) {
			p := make([]byte, 4)
			for {
				n, err := rc.Read(p)
				if err != nil {
					if err == io.EOF {
						fmt.Println("got EOF from reverse conn, closing this side", string(p[:n]))
						err := rc.Close()
						if err != nil {
							fmt.Println("error clsing rev conn after EOF")
						}
						break
					}
					fmt.Println(err)
					os.Exit(1)
				}
				command := string(p[:n])
				//fmt.Println("reverse listener got message", command)
				if subChan, ok := rl.msgSub[command]; ok {
					subChan <- true
				}
				msgCb(command)
			}
		}(revConn)

	}(listener)

	return &rl
}
func (rl *reverseListener) send(msg string) { // return err?
	fmt.Println("Sending to reverse message", msg)
	//c := *rl.conn
	_, err := (*rl.conn).Write([]byte(msg))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func (rl *reverseListener) waitFor(msg string) {
	fmt.Println("rev waiting for", msg)
	done := make(chan bool)
	rl.msgSub[msg] = done
	<-done
	delete(rl.msgSub, msg)
}
func (rl reverseListener) close() {
	//conn.end() or some such
	err := os.Remove(rl.sockPath)
	if err != nil {
		fmt.Println(err)
		//os.Exit(1)	// don't exit. if file didn't exist it errs.
	}
}

// // utilities for container management
// func mount(app, appSpace, containerName string) { // later pass app space data so we can get user and app version
// 	dsAsMounter([]string{app, appSpace, containerName})
// }
// func unMount(containerName string) {
// 	dsAsMounter([]string{containerName})
// }
// func dsAsMounter(args []string) {
// 	cmd := exec.Command("/home/developer/ds-as-mounter", args...)
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	err := cmd.Run()
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	fmt.Println("done running ds-as-mounter command", args)
// }

///////general utils
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Printf("%s took %s\n", name, elapsed)
}
