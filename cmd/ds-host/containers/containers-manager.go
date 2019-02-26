package containers

import (
	"fmt"
	"github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	"github.com/teleclimber/DropServer/cmd/ds-host/mountappspace"
	"os"
	"regexp"
	"strconv"
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

	containers, err := lxdConn.GetContainersFull()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	isSandbox := regexp.MustCompile(`ds-sandbox-[0-9]+$`).MatchString

	for _, container := range containers {
		// fmt.Println(container.Name, container.Status, container.State.Network)
		// network := container.State.Network
		// for k, v := range network {
		// 	fmt.Println(k, "Hwaddr:", v.Hwaddr, "HostName:", v.HostName, "Addresses:", v.Addresses)
		// }

		if isSandbox(container.Name) {
			// shutdown
			// since we are at init, then nothing should be connected to this process.
			// so just turn it off.

			containerID := container.Name[11:]

			fmt.Println("Container Status", container.Status)

			if container.Status == "Running" {
				// stop it
				fmt.Println("Stopping Container", container.Name)

				reqState := lxdApi.ContainerStatePut{
					Action:  "stop",
					Timeout: -1}

				op, err := lxdConn.UpdateContainerState(container.Name, reqState, "")
				if err != nil {
					fmt.Println(err)
				}

				err = op.Wait()
				if err != nil {
					fmt.Println(err)
				}
			}

			//then delete it.
			fmt.Println("Deleting Container", container.Name)

			op, err := lxdConn.DeleteContainer(container.Name)
			if err != nil {
				fmt.Println(err)
			}

			err = op.Wait()
			if err != nil {
				fmt.Println(err)
			}

			// then delete the directory structure
			// should we ensure that there is nothing mounted there first?
			mountappspace.UnMount(containerID)

			// should really check to make sure dirs are empty?
			// or you can delete dirs / files individually, it will simply error if something is left in dir.
			// err = os.Remove(sandboxDataPath + "/" + containerID + "/app/")
			// err = os.Remove(sandboxDataPath + "/" + containerID + "/app_space/")
			// err = os.Remove(sandboxDataPath + "/" + containerID + "/reverse.sock")
			// err = os.Remove(sandboxDataPath + "/" + containerID + "/recycle.sock")
			// err = os.Remove(sandboxDataPath + "/" + containerID + "/")
			// shouldn't this be a separate stop?
			// ..or should we try to go through every directory in sandboxDataPath and delete?
			// -> in case there are containers deleted but remaining dirs for some reason.

		}
	}

	// now create a handful of containers
	for i := 0; i < 1; i++ {
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
		Address:    "",
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

	newContainer.getHostIP()

	newContainer.reverseListener = newReverseListener(newContainer.Name, newContainer.hostIP, newContainer.onReverseMsg)

	fmt.Println("container started, recycling")
	newContainer.recycle()
}

// StartContainer actually just reccycles an existing container for now
func (cM *Manager) StartContainer() {
	newContainer := Container{
		Name:       "c7",
		Status:     "starting",
		Address:    "http://10.140.177.203:3030",
		appSpaceID: "",
		statusSub:  make(map[string][]chan bool)}

	cM.containers = append(cM.containers, &newContainer)
	// ^^ you want it in there early so that you don't start another one?

	newContainer.recycleListener = newRecycleListener("c7", newContainer.onRecyclerMsg)

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
			if c.Status == "starting" || c.Status == "recycling" {
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
		var candidate *Container
		for _, c := range cM.containers {
			if c.Status == "committed" && !c.isTiedUp() {
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
