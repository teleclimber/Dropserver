package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/teleclimber/DropServer/cmd/ds-sandbox-d/hostcomms"
)

type process struct {
	pid     int
	uid     string
	command string
}

var sleepDuration = 5 * time.Millisecond

var curState string // starting => ready > start-run -> running -> killing -> ready

var hostComms *hostcomms.HostComms

func main() {
	curState = "starting"

	f, err := os.OpenFile("/var/log/ds-sandbox-d.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("testing log output")

	hostComms = hostcomms.GetComms(hostCommsCb)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		hostComms.SendLog(hostcomms.INFO, "Caught signal, quitting.")
		log.Println("Caught signal, quitting.", sig)
		killNonRoot()
		hostComms.Close()
	}()

	curState = "ready"

	hostComms.SendStatus("hi")

	<-hostComms.Ended

	log.Println("Exiting")
}

func hostCommsCb(command string) {
	if command == "kill" {
		doKill()
	} else if command[0:3] == "run" {
		if curState != "ready" {
			log.Println("Received kill while in state", curState)
			hostComms.SendLog(hostcomms.ERROR, "Received kill while in state " + curState )
			hostComms.SendStatus("fail")
			curState = "failed"
		} else {
			log.Println("Gto run with ip", command[4:])
			startRunner(command[4:])
		}
	} else {
		log.Fatal("unrecognized command", command)
	}
}

///////////////// process kill
func doKill() {
	curState = "killing"

	var killed bool

	killNonRoot()

	for i := 1; i < 21; i++ {

		time.Sleep(5 * time.Millisecond)

		killed = nonRootsKilled()

		if killed {
			break
		}
	}

	if !killed {
		hostComms.SendLog(hostcomms.ERROR, "Failed to kill all non-root processes")
		hostComms.SendStatus("fail")
		curState = "failed-to-kill"
	} else {
		hostComms.SendStatus("kild")
		curState = "ready"
	}
}
func killNonRoot() {
	processes := getNonRootProcesses()

	sendSignal(processes)
}
func nonRootsKilled() bool {
	processes := getNonRootProcesses()
	if len(processes) > 0 {
		log.Println("remaining processes")
		for _, p := range processes {
			log.Println(*p)
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
		hostComms.SendLog(hostcomms.ERROR, "error in getPIDs "+ err.Error() )
		log.Println("error in getPIDs", err)
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
				log.Println("SIGTERM error", err)
			}
		} else {
			log.Println("Process not found for pid", p.pid, err)
		}
	}
}

func startRunner(hostIP string) {
	curState = "start-run"
	log.Println(curState)

	f, err := os.OpenFile("/var/log/ds-sandbox-runner.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("node", "/root/ds-sandbox-runner.js", hostIP)
	cmd.Stdout = f
	cmd.Stderr = f
	err = cmd.Start()
	if err != nil {
		log.Println(err)
	}

	// var wg sync.WaitGroup
	// wg.Add(1)
	leave := false

	go func() {
		log.Println("wating for cmd")
		err = cmd.Wait()
		if err != nil {
			log.Println("cmd Wait error", err) //this could be handy to catch node crashing out!
		}
		log.Println("done wating for cmd")
		//wg.Done()
		leave = true
	}()

	// maybe we should wait until we see the process show up before returning
	// ..however if the process crashes before we see it, we have to leave it
	waits := 0
	for leave == false {
		time.Sleep(2 * time.Millisecond)
		waits++

		//log.Println("looking for cmd new pid in ps", cmd.Process.Pid)
		pidStr := strconv.Itoa(cmd.Process.Pid)

		psCmd := exec.Command("ps", "-opid,comm")
		output, err := psCmd.CombinedOutput()
		if err != nil {
			log.Println("error in getting PIDs", err)
		}

		outputLines := strings.Split(string(output), "\n")

		for _, line := range outputLines {
			pieces := strings.Fields(line)
			log.Println(line, pieces)
			if len(pieces) > 1 && pieces[0] == pidStr && pieces[1] == "node" {
				leave = true
				curState = "running"
				log.Printf("running: Started node as subprocess %d, found it after %d waits\n", cmd.Process.Pid, waits)
				hostComms.SendLog(hostcomms.INFO, "running: Started node as subprocess")
				break
			}
		}

		if waits > 10 && leave == false {
			curState = "start-run-failed"
			log.Println(curState, "Never found pid in ps", outputLines)
			hostComms.SendLog(hostcomms.ERROR, "Never found pid in ps" )
			// his should really trigger an error condition in sandbox
			break
		}
	}
}
