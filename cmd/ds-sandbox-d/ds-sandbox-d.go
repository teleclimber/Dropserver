package main

import (
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"nanomsg.org/go/mangos/v2"
	"nanomsg.org/go/mangos/v2/protocol/pair"
	// register transports...
	_ "nanomsg.org/go/mangos/v2/transport/ipc"
)

type process struct {
	pid     int
	uid     string
	command string
}

const socketPath = "/mnt/cmd-socket"

var sleepDuration = 5 * time.Millisecond

var curState string // starting => ready > start-run -> running -> killing -> ready

func main() {
	curState = "starting"

	f, err := os.OpenFile("/var/log/ds-sandbox-d.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("testing log output")

	sock, err := pair.NewSocket()
	if err != nil {
		log.Fatal(err)
	}

	err = sock.Dial("ipc://" + socketPath)
	if err != nil {
		log.Fatal(err)
	}

	ended := make(chan bool)
	go recvLoop(sock, ended)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println("Caught signal, quitting.", sig)
		killNonRoot()
		sock.Close() // this should unblock the read loop?
	}()

	curState = "ready"

	send(sock, "hi")

	<-ended

	log.Println("Exiting")
}

/// cmd channel handling
func recvLoop(sock mangos.Socket, end chan bool) {
	defer func() {
		r := recover()
		if r != nil {
			log.Println("Panic recover:", r)
			send(sock, "fail")
		}
	}()

	for {
		msg, err := sock.Recv()
		command := string(msg)
		if err != nil {
			log.Println("Error in receiving in recvLoop", err)
			end <- true
			break
		} else {
			log.Println("Received in loop:", command)

			if command == "kill" {
				doKill(sock)
			} else if command[0:3] == "run" {
				if curState != "ready" {
					log.Println("Received kill while in state", curState)
					send(sock, "fail")
					curState = "failed"
				} else {
					log.Println("Gto run with ip", command[4:])
					startRunner(command[4:])
				}
			} else {
				log.Fatal("unrecognized command", command)
			}
		}
	}
}

func send(sock mangos.Socket, msg string) {
	log.Println("Sending", msg)
	err := sock.Send([]byte(msg))
	if err != nil {
		log.Fatal(err)
	}
}

///////////////// process kill
func doKill(sock mangos.Socket) {
	curState = "killing"

	var killed bool

	killNonRoot()

	for i := 1; i < 11; i++ {

		time.Sleep(5 * time.Millisecond)

		killed = nonRootsKilled()

		if killed {
			break
		}
	}

	if !killed {
		send(sock, "fail")
		curState = "failed-to-kill"
	} else {
		send(sock, "kild")
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
				//log.Println("found pid in ps, leaving loop")
				break
			}
		}

		if waits > 10 && leave == false {
			curState = "start-run-failed"
			log.Println(curState, "Never found pid in ps", outputLines)
			// his should really trigger an error condition in sandbox
			break
		}
	}
}
