package main

import (
	//"fmt"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type process struct {
	pid     int
	uid     string
	command string
}

const socketPath = "/mnt/cmd-socket"

func main() {
	f, err := os.OpenFile("/var/log/ds-sandbox-d.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("testing log output")

	sleepDuration := 5 * time.Millisecond

	// Baiscally a client that listens andwrites to a UDS
	c, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalln("failed to dial", err)
	}
	defer c.Close()
	log.Println("dial successful")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println("Caught signal, quitting.", sig)
		c.Close() // this should unblock the read loop?
	}()

	_, err = c.Write([]byte("hi"))
	if err != nil {
		log.Fatalln("failed to write to socket", err)
	}

	log.Println("wrote HI")

	stop := make(chan bool)

	go func(conn net.Conn, stopCh chan bool) {
		p := make([]byte, 4)
		for {
			n, err := conn.Read(p)
			if err != nil {
				if err == io.EOF {
					log.Println("got EOF from recycle conn", string(p[:n]))
				} else {
					log.Println("Conn Read error, quitting.", err)
				}

				stopCh <- true
				break
			}
			command := string(p[:n])
			log.Println("got command", command)

			if command == "kill" {
				killNonRoot()
				time.Sleep(sleepDuration)

				killed := nonRootsKilled()

				if !killed {
					// send something to host
					_, err := conn.Write([]byte("fail"))
					if err != nil {
						log.Fatal(err)
					}
				} else {
					// send back "kild" to let host know it can unmount?
					_, err := conn.Write([]byte("kild"))
					if err != nil {
						log.Fatal(err)
					}
				}
			} else if command == "run" {
				startRunner()
			} else {
				log.Fatal("unrecognized command", command)
			}
		}
	}(c, stop)

	<-stop

	log.Println("Exiting")
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

func startRunner() {
	cmd := exec.Command("node", "/root/ds-sandbox-runner.js")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
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
