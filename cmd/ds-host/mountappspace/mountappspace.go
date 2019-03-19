package mountappspace

import (
	"fmt"
	"os"
	"os/exec"
)

// Mount app space in sandbox
func Mount(app, appSpace, sandboxName string) { // later pass app space data so we can get user and app version
	dsAsMounter([]string{app, appSpace, sandboxName})
}

// UnMount app-space from sandbox
func UnMount(sandboxName string) {
	dsAsMounter([]string{sandboxName})
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
