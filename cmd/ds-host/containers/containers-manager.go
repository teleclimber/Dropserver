package containers

import (
	"fmt"
	"github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/teleclimber/DropServer/cmd/ds-host/mountappspace"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"
)

const lxdUnixSocket = "/var/snap/lxd/common/lxd/unix.socket"

// Manager manages containers
type Manager struct {
	containers []*Container
	nextID     int
}

// Now to really manage containers.
// - first get and import lxd client API
// -

// Init zaps existing sandboxes and creates fresh ones
func (cM *Manager) Init() {
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

	for _, lxdContainer := range lxdContainers {
		// fmt.Println(container.Name, container.Status, container.State.Network)
		// network := container.State.Network
		// for k, v := range network {
		// 	fmt.Println(k, "Hwaddr:", v.Hwaddr, "HostName:", v.HostName, "Addresses:", v.Addresses)
		// }

		if isSandbox(lxdContainer.Name) {
			// shutdown
			// since we are at init, then nothing should be connected to this process.
			// so just turn it off.

			containerID := lxdContainer.Name[11:]

			fmt.Println("Container Status", lxdContainer.Status)

			if lxdContainer.Status == "Running" {
				// stop it
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
		}
	}

	cM.nextID = 1

	// now create a handful of containers
	for i := 0; i < 3; i++ {
		cM.launchNewSandbox()
	}
}

// StopAll takes all known containers and stops them
func (cM *Manager) StopAll() {
	for _, c := range cM.containers {
		// If we get to this point assume the connection from the host http proxy has been stopped
		// so it should be safe to shut things down
		// ..barring anything "waiting for"...
		c.Stop()
	}

}

// launchNewSandbox creates a new container from sandbox image and starts it.
func (cM *Manager) launchNewSandbox() {
	// get a next id, by taking current nextId and checking to be sure there is nothing there in dir.
	// ..AND checking to make sure there is no container by that name ?

	containerID := strconv.Itoa(cM.nextID)
	cM.nextID++

	fmt.Println("Creating new Sandbox", containerID)

	newContainer := Container{
		Name:       containerID, // change that key please
		Status:     "starting",
		appSpaceID: "",
		statusSub:  make(map[string][]chan bool)}

	cM.containers = append(cM.containers, &newContainer)

	newContainer.recycleListener = newRecycleListener(containerID, newContainer.onRecyclerMsg)

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
		os.Exit(1)
	}

	err = op.Wait()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	go newContainer.start()

	newContainer.recycleListener.waitFor("hi")

	newContainer.getIPs()

	newContainer.Address = "http://[" + newContainer.containerIP + "%25ds-sandbox-" + containerID + "]:3030"
	// ^ %25 is % escaped

	newContainer.reverseListener = newReverseListener(newContainer.Name, newContainer.hostIP, newContainer.onReverseMsg)

	fmt.Println("container started, recycling")
	newContainer.recycle()
}

// GetForAppSpace tries to find an available container for an app-space
func (cM *Manager) GetForAppSpace(app string, appSpace string) (retContainer *Container, ok bool) {
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
			if c.Status == "ready" && c.appSpaceID == "" {
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
			if (c.Status == "starting" || c.Status == "recycling") && c.appSpaceID == "" {
				// can we have a c.reserve?
				fmt.Println("reserving container that is starting or recycling", c.Name)
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
		var candidate *Container
		for _, c := range cM.containers {
			if c.Status == "committed" && !c.appSpaceSession.tiedUp {
				if candidate == nil {
					candidate = c
				} else if candidate.appSpaceSession.lastActive.After(c.appSpaceSession.lastActive) {
					candidate = c
				}
			}
		}

		if candidate != nil {
			fmt.Println("forced recycling of container", candidate.Name)
			candidate.appSpaceID = appSpace
			candidate.recycle()
			candidate.commit(app, appSpace)
			retContainer = candidate
			ok = true
		}
	}

	go cM.evaluatePool()

	return
}

func (cM *Manager) evaluatePool() {
	// look at the list of containers and decide whether some should be recycled.
	num := len(cM.containers)
	target := int(math.Round(float64(num) / 3))

	cur := 0
	for _, c := range cM.containers {
		if c.Status == "ready" {
			cur++
			c.recycleScore = 0
		} else {
			// calculate recycle score
			duration := time.Since(c.appSpaceSession.lastActive)
			c.recycleScore = duration.Seconds()
		}
	}

	cM.PrintContainers()

	if cur < target {
		sort.Slice(cM.containers, func(i, j int) bool {
			return cM.containers[i].recycleScore > cM.containers[j].recycleScore
		})

		numRecyc := target - cur
		recycled := 0
		for _, c := range cM.containers {
			if !c.appSpaceSession.tiedUp {
				c.recycle()
				recycled++
				if recycled == numRecyc {
					break
				}
			}
		}
	}
}

// PrintContainers outputs containersa and status
func (cM *Manager) PrintContainers() {
	for _, c := range cM.containers {
		tiedUp := "not-tied"
		if c.appSpaceSession.tiedUp {
			tiedUp = "tied-up"
		}
		fmt.Println(c.Name, c.Status, c.appSpaceID, tiedUp, c.recycleScore)
	}
}
