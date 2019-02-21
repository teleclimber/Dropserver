package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"regexp"
	"sync"
)

// todo:
// - Add tests for inputs
// - Error on bad inputs
// - Error on mount / unount problems
// - use external config for base dirs
// - proper exit code so errors can be spotted from caller script
// - refuse to work if permissions on self or config wrong?
// ...

func main() {
	args := os.Args[1:]
	// we will also need args for user_id and app version

	isValid := regexp.MustCompile(`^[A-Za-z0-9\-]+$`).MatchString
	for _, a := range args {
		if !isValid(a) {
			fmt.Fprintf(os.Stderr, "invalid arg %v\n", a)
			os.Exit(1)
		}
	}

	appsPath := "/var/snap/lxd/common/lxd/storage-pools/dir-pool/containers/ds-trusted/rootfs/data/apps/"
	appSpacesPath := "/var/snap/lxd/common/lxd/storage-pools/dir-pool/containers/ds-trusted/rootfs/data/app-spaces/"
	containersPath := "/var/snap/lxd/common/lxd/storage-pools/dir-pool/containers/ds-sandbox-"

	// ^^ really need to ingest a config file!

	var wg sync.WaitGroup

	numArg := len(args)

	if numArg == 1 {
		wg.Add(2)
		go unmount(containersPath+args[0]+"/rootfs/app/", &wg)
		go unmount(containersPath+args[0]+"/rootfs/app-space/", &wg)
	} else if numArg == 3 {
		wg.Add(2)
		go mount(appsPath+args[0], containersPath+args[2]+"/rootfs/app/", &wg)
		go mount(appSpacesPath+args[1], containersPath+args[2]+"/rootfs/app-space/", &wg)
	} else {
		fmt.Println("wrong number of arguments")
	}

	wg.Wait()
}

func mount(src, dest string, wg *sync.WaitGroup) {
	defer wg.Done()
	//fmt.Println("Mounting", src, dest)
	err := unix.Mount(src, dest, "", unix.MS_BIND, "")
	if err != nil {
		fmt.Println(src, dest, err)
	}
}

func unmount(dest string, wg *sync.WaitGroup) {
	defer wg.Done()
	//fmt.Println("Unmounting", dest)
	err := unix.Unmount(dest, 0)
	if err != nil {
		fmt.Println("Unmount", dest, err)
	}
}
