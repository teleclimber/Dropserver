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

func main() {
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

	send(sock, "hi")

	<-ended

	log.Println("Exiting")
}

/// cmd channel handling
func recvLoop(sock mangos.Socket, end chan bool) {
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
				log.Println("Gto run with ip", command[4:])
				startRunner(command[4:])
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
	} else {
		send(sock, "kild")
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

	go func() {
		log.Println("wating for cmd")
		err = cmd.Wait()
		if err != nil {
			log.Println("cmd Wait error", err) //this could be handy to catch node crashing out!
		}
		log.Println("done wating for cmd")
	}()

	log.Printf("Just started node as subprocess %d\n", cmd.Process.Pid)
}
