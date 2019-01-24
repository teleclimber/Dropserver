package mountappspace

import (
	"fmt"
	"os"
	"os/exec"
)

// Mount app space in container
func Mount(app, appSpace, containerName string) { // later pass app space data so we can get user and app version
	dsAsMounter([]string{app, appSpace, containerName})
}

// UnMount app-space from container
func UnMount(containerName string) {
	dsAsMounter([]string{containerName})
}
func dsAsMounter(args []string) {
	cmd := exec.Command("/home/developer/ds-files/bin/ds-mount-appspace", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("done running ds-as-mounter command", args)
}
