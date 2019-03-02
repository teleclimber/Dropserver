package containers

import (
	"fmt"
	"github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/teleclimber/DropServer/cmd/ds-host/mountappspace"
	"github.com/teleclimber/DropServer/internal/timetrack"
	"net"
	"net/http"
	"os"
	"time"
)

type appSpaceSession struct {
	tasks      []*Task
	lastActive time.Time
	tiedUp     bool
}

// Task tracks the container being tied up for one request
type Task struct {
	Finished bool //build up with start time, elapsed and any other metadata
}

// Container holds the data necessary to interact with the container
type Container struct {
	Name            string
	Status          string
	Address         string
	hostIP          net.IP
	containerIP     string
	appSpaceID      string
	recycleListener *recycleListener
	reverseListener *reverseListener
	statusSub       map[string][]chan bool
	Transport       http.RoundTripper
	appSpaceSession appSpaceSession
	recycleScore    float64
}

// Stop stops the container and its associated open connections
func (c *Container) Stop() {
	c.recycleListener.close()
	// delete it? how do we restart?

	// reverse listener...

	lxdState := c.getLxdState()

	if lxdState.Status == "Running" {
		// stop it
		fmt.Println("Stopping Running Container", c.Name)

		lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		reqState := lxdApi.ContainerStatePut{
			Action:  "stop",
			Timeout: -1}

		op, err := lxdConn.UpdateContainerState("ds-sandbox-"+c.Name, reqState, "")
		if err != nil {
			fmt.Println(err)
		}

		err = op.Wait()
		if err != nil {
			fmt.Println(err)
		}
	}
}

// TaskBegin adds a new task to session tasks and returns it
func (c *Container) TaskBegin() *Task {
	reqTask := Task{}
	c.appSpaceSession.tasks = append(c.appSpaceSession.tasks, &reqTask)
	c.appSpaceSession.lastActive = time.Now()
	c.appSpaceSession.tiedUp = true
	return &reqTask
}

// TaskEnd stops the task and evaluates container tie up and last active
func (c *Container) TaskEnd(task *Task) {
	task.Finished = true
	c.appSpaceSession.lastActive = time.Now()
	c.appSpaceSession.tiedUp = c.isTiedUp()
}

func (c *Container) start() {
	fmt.Println("starting sandbox", c.Name)

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	reqState := lxdApi.ContainerStatePut{
		Action:  "start",
		Timeout: -1,
	}

	op, err := lxdConn.UpdateContainerState("ds-sandbox-"+c.Name, reqState, "")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for the operation to complete
	err = op.Wait()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// once the container is up we can launch our sandbox program
	// ugh does that put us in a difficult "run while leaving unattended"
}

func (c *Container) getLxdState() *lxdApi.ContainerState {
	fmt.Println("getting sandbox LXD state", c.Name)

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	state, _, err := lxdConn.GetContainerState("ds-sandbox-" + c.Name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return state
}
func (c *Container) getIPs() {
	iface, err := net.InterfaceByName("ds-sandbox-" + c.Name)
	if err != nil {
		fmt.Println("unable to get interface for container", err)
		os.Exit(1)
	}

	addresses, err := iface.Addrs()
	if err != nil {
		fmt.Println("unable to get addresses for interface", err)
		os.Exit(1)
	}

	if len(addresses) != 1 {
		fmt.Println("number of IP addresses is not 1. addresses:", addresses)
		os.Exit(1)
	}

	address := addresses[0]
	ip, _, err := net.ParseCIDR(address.String())
	if err != nil {
		fmt.Println("error getting ip from address", address, err)
		os.Exit(1)
	}

	c.hostIP = ip

	//then use lxc info to get container side IP
	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	state, _, err := lxdConn.GetContainerState("ds-sandbox-" + c.Name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// state.Network is map[string]ContainerStateNetwork
	// ContainerStateNetwork is { Addresses, Counter, Hwadr, HostName, ...}
	c.containerIP = state.Network["eth0"].Addresses[0].Address

	fmt.Println("host / container IPs:", c.hostIP, c.containerIP)
}

func (c *Container) recycle() {
	fmt.Println("starting recycle")
	defer timetrack.Track(time.Now(), "recycle")

	c.Status = "recycling"
	c.appSpaceID = ""
	c.appSpaceSession = appSpaceSession{lastActive: time.Now()} //?? why?
	// ^^ appSpaceSession isn't really relevant until committed?

	// close all connections (they should all be idle if we are recycling)
	transport, ok := c.Transport.(*http.Transport)
	if !ok {
		fmt.Println("did not find transport, sorry")
	} else {
		transport.CloseIdleConnections()
	}

	// stop reverse channel? Or will it stop itself with kill?
	// if c.reverseListener != nil {
	// 	c.reverseListener.close()
	// }
	// ^^ since host is the server, it can just kep listening and wait for another connection?

	c.recycleListener.send("kill")
	c.recycleListener.waitFor("kild")

	mountappspace.UnMount(c.Name)

	go c.reverseListener.waitForConn()

	c.recycleListener.send("run " + c.hostIP.String())
	c.reverseListener.waitFor("hi")

	c.Status = "ready"

	c.waitForDone("ready") // it's "wait for is done". urg  bad name.
}
func (c *Container) commit(app, appSpace string) {
	defer timetrack.Track(time.Now(), "commit")

	c.appSpaceID = appSpace

	c.Status = "committing"

	mountappspace.Mount(app, appSpace, c.Name)

	c.Transport = http.DefaultTransport

	c.Status = "committed"
	c.waitForDone("commited")

	// ^^ I suspect we are going to get random glitches due to concurrency.
	// Probably need to lock something somewhere. Not sure what though.
}

func (c *Container) isTiedUp() (tiedUp bool) {
	for _, task := range c.appSpaceSession.tasks {
		if !task.Finished {
			tiedUp = true
			break
		}
	}
	return
}

func (c *Container) waitFor(status string) {
	if c.Status == status {
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
func (c *Container) waitForDone(status string) {
	if subs, ok := c.statusSub[status]; ok {
		for _, wCh := range subs {
			wCh <- true
		}
		c.statusSub[status] = []chan bool{}
	}
	// then gotta empty / reset the channel.
	// though probably lock the array?
}
func (c *Container) onRecyclerMsg(msg string) {
	fmt.Println("onRecyclerMsg", msg, c.Name)
}
func (c *Container) onReverseMsg(msg string) {
	fmt.Println("onReverseMsg", msg, c.Name)
}
