package trusted

import (
	"bytes"
	"encoding/json"
	"fmt"

	lxd "github.com/lxc/lxd/client"
	lxdApi "github.com/lxc/lxd/shared/api"

	"os"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// this package manages connection with trusted container.
// It has an init that builds the container if needed
// Uses RPC to send functions to trusted.

// Init probably needs to return an instance of a TrustedContainer
// otherwise it's not clear how we track the trusted container

// for now let's just be hacky.

// Trusted manages the ds-trusted container and its communications
type Trusted struct {
	RPCClient domain.TrustedClientI
	Config    *domain.RuntimeConfig
}

// ^^ there has to be config.

const lxdUnixSocket = "/var/snap/lxd/common/lxd/unix.socket" // yikes hardcoded
const trustedDataPath = "/home/developer/ds-data"            // uhm no.

// Init creates the trusted container and launches it
func (t *Trusted) Init(wg *sync.WaitGroup) {
	defer wg.Done()

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

		fmt.Println("Deleting Container", oldContainer.Name) //TODO: Ahh no! don't do taht!! OR find better way to do that!
		// You're going to lose all your data this way.

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
	// now create and start the container

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
	// ^^ only create a container if it's missing or needs to be upgraded?

	// here we should inject runtime config...
	trustedConfig := t.getTrustedConfig()
	jsonConfig, err := json.Marshal(trustedConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	configFileArgs := lxd.ContainerFileArgs{
		Content:   bytes.NewReader(jsonConfig),
		UID:       0,
		GID:       0,
		Mode:      440,
		Type:      "file",
		WriteMode: "overwrite"}
	err = lxdConn.CreateContainerFile("ds-trusted", "/root/ds-trusted-config.json", configFileArgs)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// start the container
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

	IP := t.getIP()

	t.RPCClient.Init(IP)

}

func (t *Trusted) getTrustedConfig() *domain.TrustedConfig {
	return &domain.TrustedConfig{
		Loki: t.Config.Loki}
}

func (t *Trusted) getIP() (containerIP string) {

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println("ds-trusted get ip err connecting to lxc", err)
		os.Exit(1)
	}

	/////// -------- get via leases
	var leases []lxdApi.NetworkLease
	for tries := 0; containerIP == "" && tries < 30; {
		leases, err = lxdConn.GetNetworkLeases("lxdbr0") //TODO: make lxdbr0 a config thing
		if err != nil {
			fmt.Println("error getting lxd network leases", err)
			os.Exit(1)
		}

		for _, l := range leases {
			if l.Hostname == "ds-trusted" {
				containerIP = l.Address
				break
			}
		}

		time.Sleep(time.Second)
		tries++
	}

	if containerIP == "" {
		fmt.Println("no lease found for ds-trusted")
		os.Exit(1)
	}

	fmt.Println("ds-trusted IP:", containerIP)

	return
}

// Stop stops the ds-trusted server.
func (t *Trusted) Stop(wg *sync.WaitGroup) {
	defer wg.Done()

	lxdState := t.getLxdState()

	if lxdState.Status == "Running" {
		// stop it
		fmt.Println("Stopping ds-trusted")

		lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
		if err != nil {
			fmt.Println("ds-trusted", err)
			os.Exit(1)
		}

		reqState := lxdApi.ContainerStatePut{
			Action:  "stop",
			Timeout: -1}

		op, err := lxdConn.UpdateContainerState("ds-trusted", reqState, "")
		if err != nil {
			fmt.Println("ds-trusted", err)
		}

		err = op.Wait()
		if err != nil {
			fmt.Println("ds-trusted", err)
		}
	}
}

func (t *Trusted) getLxdState() *lxdApi.ContainerState {
	fmt.Println("getting ds-trusted LXD state")

	lxdConn, err := lxd.ConnectLXDUnix(lxdUnixSocket, nil)
	if err != nil {
		fmt.Println("ds-trusted", err)
		os.Exit(1)
	}

	state, _, err := lxdConn.GetContainerState("ds-trusted")
	if err != nil {
		fmt.Println("ds-trusted", err)
		os.Exit(1)
	}

	return state
}
