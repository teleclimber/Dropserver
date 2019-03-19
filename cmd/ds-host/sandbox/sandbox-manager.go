package sandbox

import (
	"container/list"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/teleclimber/DropServer/cmd/ds-host/mountappspace"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

const lxdUnixSocket = "/var/snap/lxd/common/lxd/unix.socket"

// Manager manages sandboxes
type Manager struct {
	sandboxes          []*Sandbox
	nextID             int
	poolMux            sync.Mutex
	requests           map[string]request
	committedSandboxes map[string]*Sandbox
	readySandboxes     list.List //[]*Sandbox //make that a FIFO list?
	readyCh            chan *Sandbox
	readyStop          chan bool
}

type request struct {
	appSpace        string
	app             string
	sandboxChannels []chan *Sandbox
}

// Init zaps existing sandboxes and creates fresh ones
func (sM *Manager) Init(initWg *sync.WaitGroup) {
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

				fmt.Println("Sandbox Status", lxdContainer.Status)

				if lxdContainer.Status == "Running" {
					fmt.Println("Stopping Sandbox", lxdContainer.Name)

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
				fmt.Println("Deleting Sandbox", lxdContainer.Name)

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

	sM.requests = make(map[string]request)
	sM.committedSandboxes = make(map[string]*Sandbox)

	sM.readyCh = make(chan *Sandbox)
	go sM.readyIn()

	sM.nextID = 1

	// now create a handful of sandboxes
	var wg sync.WaitGroup
	num := 9
	wg.Add(num)
	for i := 0; i < num; i++ {
		sandboxID := strconv.Itoa(sM.nextID)
		sM.nextID++
		go sM.launchNewSandbox(sandboxID, &wg)
	}

	wg.Wait()
}

// StopAll takes all known sandboxes and stops them
func (sM *Manager) StopAll() {
	var stopWg sync.WaitGroup
	for _, c := range sM.sandboxes {
		// If we get to this point assume the connection from the host http proxy has been stopped
		// so it should be safe to shut things down
		// ..barring anything "waiting for"...
		stopWg.Add(1)
		go c.Stop(&stopWg)
	}

	stopWg.Wait()
}

// launchNewSandbox creates a new container from sandbox image and starts it.
func (sM *Manager) launchNewSandbox(sandboxID string, wg *sync.WaitGroup) {
	// get a next id, by taking current nextId and checking to be sure there is nothing there in dir.
	// ..AND checking to make sure there is no container by that name ?

	defer wg.Done()

	fmt.Println("Creating new Sandbox", sandboxID)

	newSandbox := Sandbox{
		Name:       sandboxID, // change that key please
		Status:     "starting",
		appSpaceID: "",
		statusSub:  make(map[string][]chan bool),
		LogClient:  record.NewSandboxLogClient(sandboxID)}

	sM.sandboxes = append(sM.sandboxes, &newSandbox)

	sM.recordSandboxStatusMetric()

	newSandbox.recycleListener = newRecycleListener(sandboxID, newSandbox.LogClient, newSandbox.onRecyclerMsg)

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(sandboxID, err)
		os.Exit(1)
	}

	// add a unix socket proxy
	dev := map[string]map[string]string{
		"cmd-proxy": {
			"type":    "proxy",
			"bind":    "container",
			"connect": "unix:/home/developer/ds-socket-proxies/recycler-" + sandboxID,
			"listen":  "unix:/mnt/cmd-socket"},
		"eth0": {
			"type":      "nic",
			"nictype":   "p2p",
			"name":      "eth0",
			"host_name": "ds-sandbox-" + sandboxID}}

	req := lxdApi.ContainersPost{
		Name: "ds-sandbox-" + sandboxID,
		Source: lxdApi.ContainerSource{
			Type:  "image",
			Alias: "ds-sandbox",
		},
		ContainerPut: lxdApi.ContainerPut{
			Profiles: []string{"ds-sandbox-profile"},
			Devices:  dev}}

	op, err := lxdConn.CreateContainer(req)
	if err != nil {
		fmt.Println(sandboxID, err)
		os.Exit(1)
	}

	err = op.Wait()
	if err != nil {
		fmt.Println(sandboxID, err)
		os.Exit(1)
	}

	go newSandbox.start()

	newSandbox.recycleListener.waitFor("hi")

	newSandbox.getIPs()

	newSandbox.Address = "http://[" + newSandbox.containerIP + "%25ds-sandbox-" + sandboxID + "]:3030"
	// ^ %25 is % escaped

	newSandbox.reverseListener = newReverseListener(newSandbox.Name, newSandbox.hostIP, newSandbox.onReverseMsg)

	fmt.Println(sandboxID, "container started, recycling")
	newSandbox.recycle(sM.readyCh)
}

// GetForAppSpace records the need for a sandbox and returns a channel
func (sM *Manager) GetForAppSpace(app string, appSpace string) chan *Sandbox {
	ch := make(chan *Sandbox)

	sM.poolMux.Lock()
	defer sM.poolMux.Unlock()

	c, ok := sM.committedSandboxes[appSpace]
	if ok {
		//OK, but is the container ready yet?
		// it may have *just* been mark for commit, so it'll get there but have to wait
		fmt.Println("GFAS found sandbox already committed", appSpace)
		go func() {
			c.waitFor("committed")         // I don't like this. If something si going to wait, I'd rather it get put in requests or something.
			ch <- c                        // do this in goroutine because it will block
			sM.recordSandboxStatusMetric() // really?
		}()
	} else {
		req, ok := sM.requests[appSpace]
		if !ok {
			req = request{
				appSpace:        appSpace,
				app:             app,
				sandboxChannels: make([]chan *Sandbox, 0)}
		}

		req.sandboxChannels = append(req.sandboxChannels, ch)

		sM.requests[appSpace] = req

		go sM.dispatchPool()
	}

	return ch
}

func (sM *Manager) dispatchPool() {
	// assigns sandboxes to  app-spaces.
	// if a container needs to be committed, then subsequently call recyclePool
	// this has to be fast. No waiting!

	// there may be requests but no sandboxes
	// or no requests but sandboxes
	// or both,
	// or none.

	fmt.Println("dispatchPool")
	sM.poolMux.Lock()
	defer sM.poolMux.Unlock()

	for appSpace := range sM.requests {
		// maybe check that all requests are still active first..
		front := sM.readySandboxes.Front()
		if front == nil {
			record.Log(record.WARN, map[string]string{"app-space": appSpace},
				"dispatch pool: no sandboxes left for app-space")
			break
		} else {
			r := sM.requests[appSpace]

			c := front.Value.(*Sandbox)
			c.appSpaceID = r.appSpace
			c.Status = "committing"

			sM.readySandboxes.Remove(front)
			delete(sM.requests, appSpace)
			sM.committedSandboxes[r.appSpace] = c

			go sM.commit(c, r)

			go sM.recyclePool()
		}
	}

	go sM.recordSandboxStatusMetric()
}
func (sM *Manager) commit(sandbox *Sandbox, request request) {
	fmt.Println("cmcommit for ", sandbox.Name, request.appSpace)

	sandbox.commit(request.app, request.appSpace)

	for _, ch := range request.sandboxChannels {
		ch <- sandbox // will panic if ch is closed! Will block if nobody at the other end
		// -> requires precise management of the channel.
	}

	sM.recordSandboxStatusMetric()
}

func (sM *Manager) recyclePool() {
	sM.poolMux.Lock()
	defer sM.poolMux.Unlock()

	numC := len(sM.sandboxes)
	maxCommitted := int(math.Round(float64(numC) * 2 / 3))
	numRecyc := len(sM.committedSandboxes) - maxCommitted

	if numRecyc > 0 {
		var sortedAppSpaces []string

		for appSpace := range sM.committedSandboxes {
			c := sM.committedSandboxes[appSpace]
			if c.Status == "committed" && !c.appSpaceSession.tiedUp {
				duration := time.Since(c.appSpaceSession.lastActive)
				c.recycleScore = duration.Seconds()
				sortedAppSpaces = append(sortedAppSpaces, appSpace)
			}
		}

		sort.Slice(sortedAppSpaces, func(i, j int) bool {
			asi := sortedAppSpaces[i]
			asj := sortedAppSpaces[j]
			return sM.committedSandboxes[asi].recycleScore > sM.committedSandboxes[asj].recycleScore
		})

		for i := 0; i < numRecyc && i < len(sortedAppSpaces); i++ {
			appSpace := sortedAppSpaces[i]
			c := sM.committedSandboxes[appSpace]
			c.Status = "recycling"
			c.appSpaceID = ""
			delete(sM.committedSandboxes, appSpace)
			go c.recycle(sM.readyCh)

		}
	}

	go sM.recordSandboxStatusMetric()
}

func (sM *Manager) readyIn() { //deliberately bad name
	for {
		select {
		case readyC := <-sM.readyCh:
			sM.poolMux.Lock()
			sM.readySandboxes.PushBack(readyC)
			sM.poolMux.Unlock()
			sM.dispatchPool()
		case <-sM.readyStop:
			break
		}
	}
}

func (sM *Manager) recordSandboxStatusMetric() {
	var s = &record.SandboxStatuses{
		Starting:   0,
		Ready:      0,
		Committing: 0,
		Committed:  0,
		Recycling:  0}
	for _, c := range sM.sandboxes {
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

// PrintSandboxes outputs containersa and status
func (sM *Manager) PrintSandboxes() {
	var readys []string
	for rc := sM.readySandboxes.Front(); rc != nil; rc = rc.Next() {
		readys = append(readys, rc.Value.(*Sandbox).Name)
	}

	fmt.Println("Ready Sandboxes", readys)

	fmt.Println("Committed sandboxes", sM.committedSandboxes)
	for _, c := range sM.sandboxes {
		tiedUp := "not-tied"
		if c.appSpaceSession.tiedUp {
			tiedUp = "tied-up"
		}
		fmt.Println(c.Name, c.Status, c.appSpaceID, tiedUp, c.recycleScore)
	}
}
