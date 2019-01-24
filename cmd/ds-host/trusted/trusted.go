package trusted

import (
	"fmt"
	"github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"
	//"github.com/teleclimber/DropServer/internal/trustedinterface"
	"os"
)

// this package manages connection with trusted container.
// It has an init that builds the container if needed
// Uses RPC to send functions to trusted.

// Init probably needs to return an instance of a TrustedContainer
// otherwise it's not clear how we track the trusted container

// for now let's just be hacky.

const lxdUnixSocket = "/var/snap/lxd/common/lxd/unix.socket"
const trustedDataPath = "/home/developer/ds-data"

// Init creates the trusted container and launches it
func Init() {
	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// get container with the right name?
	oldContainer, etag, err := lxdConn.GetContainer("ds-trusted")
	if err != nil {
		fmt.Println("error getting container ds-trusted", etag, err)
	} else {
		// I think this means we got a container. So shut it down and delete it.
		if oldContainer.Status == "Running" {
			// stop it
			fmt.Println("Stopping Trusted Container", oldContainer.Name)

			reqState := lxdApi.ContainerStatePut{
				Action:  "stop",
				Timeout: -1}

			op, err := lxdConn.UpdateContainerState(oldContainer.Name, reqState, "")
			if err != nil {
				fmt.Println(err)
			}

			err = op.Wait()
			if err != nil {
				fmt.Println(err)
			}
		}

		fmt.Println("Deleting Container", oldContainer.Name)

		op, err := lxdConn.DeleteContainer(oldContainer.Name)
		if err != nil {
			fmt.Println(err)
		}

		err = op.Wait()
		if err != nil {
			fmt.Println(err)
		}

	}

	//////////
	// now start the container

	fmt.Println("Creating new Trusted Container")

	// create container from image
	dev := map[string]map[string]string{
		"trusted-data": {
			"type":   "disk",
			"path":   "/mnt/data/",
			"source": trustedDataPath + "/"}}

	req := lxdApi.ContainersPost{
		Name: "ds-trusted",
		Source: lxdApi.ContainerSource{
			Type:  "image",
			Alias: "ds-trusted",
		},
		ContainerPut: lxdApi.ContainerPut{
			Profiles: []string{"ds-trusted-profile"},
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

	reqState := lxdApi.ContainerStatePut{
		Action:  "start",
		Timeout: -1,
	}

	op, err = lxdConn.UpdateContainerState("ds-trusted", reqState, "")
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

}
