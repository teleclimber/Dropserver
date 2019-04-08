package sandbox

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/mountappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/record" // should be able to rm this import
	"github.com/teleclimber/DropServer/internal/timetrack"
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

// Sandbox holds the data necessary to interact with the container
type Sandbox struct {
	Name            string // Every property should be non-exported to guarantee use of the interface
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
	LogClient       domain.LogCLientI
}

// Stop stops the container and its associated open connections
func (s *Sandbox) Stop(wg *sync.WaitGroup) {
	defer wg.Done()

	s.recycleListener.close()
	// delete it? how do we restart?

	// reverse listener...

	lxdState := s.getLxdState()

	if lxdState.Status == "Running" {
		// stop it
		fmt.Println("Stopping Running Sandbox", s.Name)

		lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
		if err != nil {
			fmt.Println(s.Name, err)
			os.Exit(1)
		}

		reqState := lxdApi.ContainerStatePut{
			Action:  "stop",
			Timeout: -1}

		op, err := lxdConn.UpdateContainerState("ds-sandbox-"+s.Name, reqState, "")
		if err != nil {
			fmt.Println(s.Name, err)
		}

		err = op.Wait()
		if err != nil {
			fmt.Println(s.Name, err)
		}
	}
}

// basic getters

// GetName gets the name of the sandbox
func (s *Sandbox) GetName() string {
	return s.Name
}

// GetAddress gets the IP address of the sandbox
func (s *Sandbox) GetAddress() string {
	return s.Address
}

// GetTransport gets the http transport of the sandbox
func (s *Sandbox) GetTransport() http.RoundTripper {
	return s.Transport
}

// GetLogClient retuns the Logging client
func (s *Sandbox) GetLogClient() domain.LogCLientI {
	return s.LogClient
}

// TaskBegin adds a new task to session tasks and returns it
func (s *Sandbox) TaskBegin() chan bool {
	reqTask := Task{}
	s.appSpaceSession.tasks = append(s.appSpaceSession.tasks, &reqTask)
	s.appSpaceSession.lastActive = time.Now()
	s.appSpaceSession.tiedUp = true

	ch := make(chan bool)

	// go func here that blocks on chanel.
	go func() {
		<-ch
		reqTask.Finished = true
		s.appSpaceSession.lastActive = time.Now()
		s.appSpaceSession.tiedUp = s.isTiedUp()
	}()

	return ch //instead of returning a task we should return a chanel.
}

func (s *Sandbox) start() {
	s.LogClient.Log(domain.INFO, nil, "Starting sandbox")

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(s.Name, err)
		os.Exit(1)
	}

	reqState := lxdApi.ContainerStatePut{
		Action:  "start",
		Timeout: -1,
	}

	op, err := lxdConn.UpdateContainerState("ds-sandbox-"+s.Name, reqState, "")
	if err != nil {
		fmt.Println(s.Name, err)
		os.Exit(1)
	}

	// Wait for the operation to complete
	err = op.Wait()
	if err != nil {
		fmt.Println(s.Name, err)
		os.Exit(1)
	}

	// once the container is up we can launch our sandbox program
	// ugh does that put us in a difficult "run while leaving unattended"
}

func (s *Sandbox) getLxdState() *lxdApi.ContainerState {
	fmt.Println("getting sandbox LXD state", s.Name)

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(s.Name, err)
		os.Exit(1)
	}

	state, _, err := lxdConn.GetContainerState("ds-sandbox-" + s.Name)
	if err != nil {
		fmt.Println(s.Name, err)
		os.Exit(1)
	}

	return state
}
func (s *Sandbox) getIPs() {
	iface, err := net.InterfaceByName("ds-sandbox-" + s.Name)
	if err != nil {
		fmt.Println(s.Name, "unable to get interface for container", err)
		os.Exit(1)
	}

	addresses, err := iface.Addrs()
	if err != nil {
		fmt.Println(s.Name, "unable to get addresses for interface", err)
		os.Exit(1)
	}

	if len(addresses) != 1 {
		fmt.Println(s.Name, "number of IP addresses is not 1. addresses:", addresses)
		os.Exit(1)
	}

	address := addresses[0]
	ip, _, err := net.ParseCIDR(address.String())
	if err != nil {
		fmt.Println(s.Name, "error getting ip from address", address, err)
		os.Exit(1)
	}

	s.hostIP = ip

	//then use lxc info to get container side IP
	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(s.Name, err)
		os.Exit(1)
	}

	state, _, err := lxdConn.GetContainerState("ds-sandbox-" + s.Name)
	if err != nil {
		fmt.Println(s.Name, err)
		os.Exit(1)
	}

	// state.Network is map[string]ContainerStateNetwork
	// ContainerStateNetwork is { Addresses, Counter, Hwadr, HostName, ...}
	s.containerIP = state.Network["eth0"].Addresses[0].Address

	fmt.Println(s.Name, "host / container IPs:", s.hostIP, s.containerIP)
}

func (s *Sandbox) recycle(readyCh chan *Sandbox) {
	s.LogClient.Log(domain.INFO, nil, "recycling start")

	defer timetrack.Track(time.Now(), "recycle")
	defer record.SandboxRecycleTime(time.Now())

	s.Status = "recycling"
	s.appSpaceID = ""
	s.appSpaceSession = appSpaceSession{lastActive: time.Now()} //?? why?
	// ^^ appSpaceSession isn't really relevant until committed?

	// close all connections (they should all be idle if we are recycling)
	transport, ok := s.Transport.(*http.Transport)
	if !ok {
		fmt.Println(s.Name, "did not find transport, sorry")
	} else {
		transport.CloseIdleConnections()
	}

	// stop reverse channel? Or will it stop itself with kill?
	// if s.reverseListener != nil {
	// 	s.reverseListener.close()
	// }
	// ^^ since host is the server, it can just kep listening and wait for another connection?

	s.recycleListener.send("kill")
	s.recycleListener.waitFor("kild")
	// ^^ here we should wait for either "kild" or "fail", and act in consequence

	mountappspace.UnMount(s.Name)

	go s.reverseListener.waitForConn()

	s.recycleListener.send("run " + s.hostIP.String())
	s.reverseListener.waitFor("hi")
	// ^ wait for ready or a timeout, otherwise this blocks forever in case of problem

	s.Status = "ready"

	s.waitForDone("ready") // it's "thing is done so you can stop waiting". urg  bad name.

	readyCh <- s

	s.LogClient.Log(domain.INFO, nil, "recycling complete") // include time in tehre for good measure?
}
func (s *Sandbox) commit(app, appSpace string) {
	s.LogClient.Log(domain.INFO, map[string]string{
		"app": app, "app-space": appSpace}, "commit start")

	defer timetrack.Track(time.Now(), "commit")
	defer record.SandboxCommitTime(time.Now())

	s.appSpaceID = appSpace

	s.Status = "committing"

	mountappspace.Mount(app, appSpace, s.Name)

	s.Transport = http.DefaultTransport

	s.Status = "committed"
	s.waitForDone("commited")

	s.LogClient.Log(domain.INFO, map[string]string{
		"app": app, "app-space": appSpace}, "commit complete")
}

func (s *Sandbox) isTiedUp() (tiedUp bool) {
	for _, task := range s.appSpaceSession.tasks {
		if !task.Finished {
			tiedUp = true
			break
		}
	}
	return
}

func (s *Sandbox) waitFor(status string) {
	if s.Status == status {
		return
	}
	fmt.Println(s.Name, "waiting for container status", status)

	if _, ok := s.statusSub[status]; !ok {
		s.statusSub[status] = []chan bool{}
	}
	statusMet := make(chan bool)
	s.statusSub[status] = append(s.statusSub[status], statusMet)
	<-statusMet
	delete(s.statusSub, status)
}
func (s *Sandbox) waitForDone(status string) {
	if subs, ok := s.statusSub[status]; ok {
		for _, wCh := range subs {
			wCh <- true
		}
		s.statusSub[status] = []chan bool{}
	}
	// then gotta empty / reset the channel.
	// though probably lock the array?
}
func (s *Sandbox) onRecyclerMsg(msg string) {
	fmt.Println("onRecyclerMsg", msg, s.Name)
}
func (s *Sandbox) onReverseMsg(msg string) {
	fmt.Println("onReverseMsg", msg, s.Name)
}
