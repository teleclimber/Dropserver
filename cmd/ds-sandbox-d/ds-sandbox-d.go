package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// at what level are we operating here?
// -> assume the container is running, and we know its name.
// - get all PID that are owned by non-root
// - for each PID send sig term
// - wait some, then if any PID still exists owned by non-root send hard kill (or reboot the container)
// -

type process struct {
	pid     int
	uid     string
	command string
}

const socketPath = "/mnt/priv_sockets/recycle.sock"

func main() {
	sleepDuration, err := time.ParseDuration("5ms")
	if err != nil {
		fmt.Println("bad duration", err)
		os.Exit(1)
	}

	// Baiscally a client that listens andwrites to a UDS
	c, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer c.Close()

	fmt.Println("dial successful")

	_, err = c.Write([]byte("hi"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	stop := make(chan bool)

	go func(conn net.Conn, stopCh chan bool) {
		p := make([]byte, 4)
		for {
			n, err := conn.Read(p)
			if err != nil {
				if err == io.EOF {
					fmt.Println("got EOF from recycle conn", string(p[:n]))
					// does that mean host disconnected? Presumably this code disconnects first?
					// Should we then kill everything?
					stopCh <- true
					break
				}
				fmt.Println(err)
				os.Exit(1)
			}
			command := string(p[:n])
			fmt.Println("got command", command)

			if command == "kill" {
				killNonRoot()
				time.Sleep(sleepDuration)

				killed := nonRootsKilled()

				if !killed {
					// send something to host
					_, err := conn.Write([]byte("fail"))
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				} else {
					// send back "kild" to let host know it can unmount?
					_, err := conn.Write([]byte("kild"))
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
				}
			} else if command == "run" {
				startRunner()
			} else {
				fmt.Println("unrecognized command", command)
				os.Exit(1)
			}
		}
	}(c, stop)

	<-stop
}

func killNonRoot() {
	processes := getNonRootProcesses()

	sendSignal(processes)
}
func nonRootsKilled() bool {
	processes := getNonRootProcesses()
	if len(processes) > 0 {
		fmt.Println("remaining processes")
		for _, p := range processes {
			fmt.Println(*p)
		}
		return false
	}
	return true
}

func getNonRootProcesses() (processes []*process) {
	processes = getAllProcesses()
	processes = getNonRoot(processes)
	return
}

func getAllProcesses() (processes []*process) {
	cmd := exec.Command("ps", "-opid,user,comm")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("error in getPIDs", err)
	}

	outputLines := strings.Split(string(output), "\n")

	for _, line := range outputLines {
		pieces := strings.Fields(line)
		if len(pieces) > 0 {
			if pid, err := strconv.Atoi(pieces[0]); err == nil {
				processes = append(processes, &process{pid, pieces[1], pieces[2]})
			}
		}
	}
	return
}

func getNonRoot(processes []*process) (nonRoot []*process) {
	for _, p := range processes {
		if p.uid != "root" {
			nonRoot = append(nonRoot, p)
		}
	}
	return
}

func sendSignal(processes []*process) {
	for _, p := range processes {
		osProc, err := os.FindProcess(p.pid)
		if err == nil {
			err := osProc.Signal(unix.SIGTERM)
			if err != nil {
				fmt.Println("SIGTERM error", err)
			}
		} else {
			fmt.Println("Process not found for pid", p.pid, err)
		}
	}
}

func startRunner() {
	cmd := exec.Command("node", "/home/cdeveloper/runner.js")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		fmt.Println(err)
	}

	go func() {
		fmt.Println("wating for cmd")
		err = cmd.Wait()
		if err != nil {
			fmt.Println("cmd Wait error", err) //this could be handy to catch node crashing out!
		}
		fmt.Println("done wating for cmd")
	}()

	fmt.Printf("Just started node as subprocess %d\n", cmd.Process.Pid)
}
