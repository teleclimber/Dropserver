package containers

import (
	"container/list"
	"fmt"
	"github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/teleclimber/DropServer/cmd/ds-host/mountappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"
)

const lxdUnixSocket = "/var/snap/lxd/common/lxd/unix.socket"

// Manager manages containers
type Manager struct {
	containers          []*Container
	nextID              int
	poolMux             sync.Mutex
	requests            map[string]request
	committedContainers map[string]*Container
	readyContainers     list.List //[]*Container //make that a FIFO list?
	readyCh             chan *Container
	readyStop           chan bool
}

type request struct {
	appSpace        string
	app             string
	sandboxChannels []chan *Container
}

// Init zaps existing sandboxes and creates fresh ones
func (cM *Manager) Init(initWg *sync.WaitGroup) {
	defer initWg.Done()

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	lxdContainers, err := lxdConn.GetContainersFull()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	isSandbox := regexp.MustCompile(`ds-sandbox-[0-9]+$`).MatchString

	var delWg sync.WaitGroup

	for _, lxdC := range lxdContainers {

		if isSandbox(lxdC.Name) {
			// since we are at init, then nothing should be connected to this process.
			// so just turn it off.

			delWg.Add(1)

			go func(lxdContainer lxdApi.ContainerFull, wg *sync.WaitGroup) {
				defer wg.Done()

				containerID := lxdContainer.Name[11:]

				fmt.Println("Container Status", lxdContainer.Status)

				if lxdContainer.Status == "Running" {
					fmt.Println("Stopping Container", lxdContainer.Name)

					reqState := lxdApi.ContainerStatePut{
						Action:  "stop",
						Timeout: -1}

					op, err := lxdConn.UpdateContainerState(lxdContainer.Name, reqState, "")
					if err != nil {
						fmt.Println(err)
					}

					err = op.Wait()
					if err != nil {
						fmt.Println(err)
					}
				}

				// unmount or delete container will fail
				mountappspace.UnMount(containerID)

				//then delete it.
				fmt.Println("Deleting Container", lxdContainer.Name)

				op, err := lxdConn.DeleteContainer(lxdContainer.Name)
				if err != nil {
					fmt.Println(err)
				}

				err = op.Wait()
				if err != nil {
					fmt.Println(err)
				}
			}(lxdC, &delWg)
		}
	}

	delWg.Wait()

	cM.requests = make(map[string]request)
	cM.committedContainers = make(map[string]*Container)

	cM.readyCh = make(chan *Container)
	go cM.readyIn()

	cM.nextID = 1

	// now create a handful of containers
	var wg sync.WaitGroup
	num := 9
	wg.Add(num)
	for i := 0; i < num; i++ {
		containerID := strconv.Itoa(cM.nextID)
		cM.nextID++
		go cM.launchNewSandbox(containerID, &wg)
	}

	wg.Wait()
}

// StopAll takes all known containers and stops them
func (cM *Manager) StopAll() {
	var stopWg sync.WaitGroup
	for _, c := range cM.containers {
		// If we get to this point assume the connection from the host http proxy has been stopped
		// so it should be safe to shut things down
		// ..barring anything "waiting for"...
		stopWg.Add(1)
		go c.Stop(&stopWg)
	}

	stopWg.Wait()
}

// launchNewSandbox creates a new container from sandbox image and starts it.
func (cM *Manager) launchNewSandbox(containerID string, wg *sync.WaitGroup) {
	// get a next id, by taking current nextId and checking to be sure there is nothing there in dir.
	// ..AND checking to make sure there is no container by that name ?

	defer wg.Done()

	fmt.Println("Creating new Sandbox", containerID)

	newContainer := Container{
		Name:       containerID, // change that key please
		Status:     "starting",
		appSpaceID: "",
		statusSub:  make(map[string][]chan bool),
		LogClient:  record.NewSandboxLogClient(containerID)}

	cM.containers = append(cM.containers, &newContainer)

	cM.recordContainerStatusMetric()

	newContainer.recycleListener = newRecycleListener(containerID, newContainer.onRecyclerMsg)

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(containerID, err)
		os.Exit(1)
	}

	// add a unix socket proxy
	dev := map[string]map[string]string{
		"cmd-proxy": {
			"type":    "proxy",
			"bind":    "container",
			"connect": "unix:/home/developer/ds-socket-proxies/recycler-" + containerID,
			"listen":  "unix:/mnt/cmd-socket"},
		"eth0": {
			"type":      "nic",
			"nictype":   "p2p",
			"name":      "eth0",
			"host_name": "ds-sandbox-" + containerID}}

	req := lxdApi.ContainersPost{
		Name: "ds-sandbox-" + containerID,
		Source: lxdApi.ContainerSource{
			Type:  "image",
			Alias: "ds-sandbox",
		},
		ContainerPut: lxdApi.ContainerPut{
			Profiles: []string{"ds-sandbox-profile"},
			Devices:  dev}}

	op, err := lxdConn.CreateContainer(req)
	if err != nil {
		fmt.Println(containerID, err)
		os.Exit(1)
	}

	err = op.Wait()
	if err != nil {
		fmt.Println(containerID, err)
		os.Exit(1)
	}

	go newContainer.start()

	newContainer.recycleListener.waitFor("hi")

	newContainer.getIPs()

	newContainer.Address = "http://[" + newContainer.containerIP + "%25ds-sandbox-" + containerID + "]:3030"
	// ^ %25 is % escaped

	newContainer.reverseListener = newReverseListener(newContainer.Name, newContainer.hostIP, newContainer.onReverseMsg)

	fmt.Println(containerID, "container started, recycling")
	newContainer.recycle(cM.readyCh)
}

// GetForAppSpace records the need for a sandbox and returns a channel
func (cM *Manager) GetForAppSpace(app string, appSpace string) chan *Container {
	ch := make(chan *Container)

	cM.poolMux.Lock()
	defer cM.poolMux.Unlock()

	c, ok := cM.committedContainers[appSpace]
	if ok {
		//OK, but is the container ready yet?
		// it may have *just* been mark for commit, so it'll get there but have to wait
		fmt.Println("GFAS found container already committed", appSpace)
		go func() {
			c.waitFor("committed")           // I don't like this. If something si going to wait, I'd rather it get put in requests or something.
			ch <- c                          // do this in goroutine because it will block
			cM.recordContainerStatusMetric() // really?
		}()
	} else {
		req, ok := cM.requests[appSpace]
		if !ok {
			req = request{
				appSpace:        appSpace,
				app:             app,
				sandboxChannels: make([]chan *Container, 0)}
		}

		req.sandboxChannels = append(req.sandboxChannels, ch)

		cM.requests[appSpace] = req

		go cM.dispatchPool()
	}

	return ch
}

func (cM *Manager) dispatchPool() {
	// assigns sandboxes to  app-spaces.
	// if a container needs to be committed, then subsequently call recyclePool
	// this has to be fast. No waiting!

	// there may be requests but no sandboxes
	// or no requests but sandboxes
	// or both,
	// or none.

	fmt.Println("dispatchPool")
	cM.poolMux.Lock()
	defer cM.poolMux.Unlock()

	for appSpace := range cM.requests {
		// maybe check that all requests are still active first..
		front := cM.readyContainers.Front()
		if front == nil {
			record.Log(record.WARN, map[string]string{"app-space": appSpace},
				"dispatch pool: no sandboxes left for app-space")
			break
		} else {
			r := cM.requests[appSpace]

			c := front.Value.(*Container)
			c.appSpaceID = r.appSpace
			c.Status = "committing"

			cM.readyContainers.Remove(front)
			delete(cM.requests, appSpace)
			cM.committedContainers[r.appSpace] = c

			go cM.commit(c, r)

			go cM.recyclePool()
		}
	}

	go cM.recordContainerStatusMetric()
}
func (cM *Manager) commit(container *Container, request request) {
	fmt.Println("cmcommit for ", container.Name, request.appSpace)

	container.commit(request.app, request.appSpace)

	for _, ch := range request.sandboxChannels {
		ch <- container // will panic if ch is closed! Will block if nobody at the other end
		// -> requires precise management of the channel.
	}

	cM.recordContainerStatusMetric()
}

func (cM *Manager) recyclePool() {
	cM.poolMux.Lock()
	defer cM.poolMux.Unlock()

	numC := len(cM.containers)
	maxCommitted := int(math.Round(float64(numC) * 2 / 3))
	numRecyc := len(cM.committedContainers) - maxCommitted

	if numRecyc > 0 {
		var sortedAppSpaces []string

		for appSpace := range cM.committedContainers {
			c := cM.committedContainers[appSpace]
			if c.Status == "committed" && !c.appSpaceSession.tiedUp {
				duration := time.Since(c.appSpaceSession.lastActive)
				c.recycleScore = duration.Seconds()
				sortedAppSpaces = append(sortedAppSpaces, appSpace)
			}
		}

		sort.Slice(sortedAppSpaces, func(i, j int) bool {
			asi := sortedAppSpaces[i]
			asj := sortedAppSpaces[j]
			return cM.committedContainers[asi].recycleScore > cM.committedContainers[asj].recycleScore
		})

		for i := 0; i < numRecyc && i < len(sortedAppSpaces); i++ {
			appSpace := sortedAppSpaces[i]
			c := cM.committedContainers[appSpace]
			c.Status = "recycling"
			c.appSpaceID = ""
			delete(cM.committedContainers, appSpace)
			go c.recycle(cM.readyCh)

		}
	}

	go cM.recordContainerStatusMetric()
}

func (cM *Manager) readyIn() { //deliberately bad name
	for {
		select {
		case readyC := <-cM.readyCh:
			cM.poolMux.Lock()
			cM.readyContainers.PushBack(readyC)
			cM.poolMux.Unlock()
			cM.dispatchPool()
		case <-cM.readyStop:
			break
		}
	}
}

func (cM *Manager) recordContainerStatusMetric() {
	var s = &record.SandboxStatuses{
		Starting:   0,
		Ready:      0,
		Committing: 0,
		Committed:  0,
		Recycling:  0}
	for _, c := range cM.containers {
		switch c.Status {
		case "starting":
			s.Starting++
		case "ready":
			s.Ready++
		case "committing":
			s.Committing++
		case "committed":
			s.Committed++
		case "recycling":
			s.Recycling++
		}
	}
	record.SandboxStatusCounts(s)
}

// PrintContainers outputs containersa and status
func (cM *Manager) PrintContainers() {
	var readys []string
	for rc := cM.readyContainers.Front(); rc != nil; rc = rc.Next() {
		readys = append(readys, rc.Value.(*Container).Name)
	}

	fmt.Println("Ready Containers", readys)

	fmt.Println("Committed containers", cM.committedContainers)
	for _, c := range cM.containers {
		tiedUp := "not-tied"
		if c.appSpaceSession.tiedUp {
			tiedUp = "tied-up"
		}
		fmt.Println(c.Name, c.Status, c.appSpaceID, tiedUp, c.recycleScore)
	}
}
