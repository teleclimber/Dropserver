package hostcomms

import (
	"encoding/json"

	"log"

	"nanomsg.org/go/mangos/v2"
	"nanomsg.org/go/mangos/v2/protocol/pair"

	// register transports...
	_ "nanomsg.org/go/mangos/v2/transport/ipc"
)

const socketPath = "/mnt/cmd-socket"

// HostComms keeps a socket so send and close alls can be made
type HostComms struct {
	sock     mangos.Socket
	Ended    chan bool
	callback func(msg string)
}

type msgStruct struct {
	Status   string
	Severity int
	Message  string
}

// LogLevel expresses the severity of a log entry as an int
type LogLevel int

// DEBUG is for debug
const (
	DEBUG LogLevel = iota
	INFO  LogLevel = iota
	WARN  LogLevel = iota
	ERROR LogLevel = iota
)

// ^^ logging stuff is temporary

// GetComms initializes a host comm and retuns a HostComms
func GetComms(callback func(msg string)) *HostComms {
	sock, err := pair.NewSocket()
	if err != nil {
		log.Fatal(err)
	}

	err = sock.Dial("ipc://" + socketPath)
	if err != nil {
		log.Fatal(err)
	}

	hc := HostComms{
		sock:     sock,
		Ended:    make(chan bool),
		callback: callback}

	go hc.recvLoop()

	return &hc
}

// Close closes the host comm
func (hc *HostComms) Close() {
	hc.sock.Close() // this should unblock the read loop?
}

// SendStatus packages status into a message and sends it to host
func (hc *HostComms) SendStatus(status string) {
	hc.SendStruct(msgStruct{Status: status})
}

// SendLog sends a log entry so that host can log it
func (hc *HostComms) SendLog(severity LogLevel, msg string) {
	hc.SendStruct(msgStruct{
		Status:   "log",
		Severity: int(severity),
		Message:  msg})
}

// SendStruct sends the struct verbatim to host
func (hc *HostComms) SendStruct(data msgStruct) {
	j, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("sending", data, string(j))

	err = hc.sock.Send(j)
	if err != nil {
		log.Fatal(err)
	}
}

/// cmd channel handling
func (hc *HostComms) recvLoop() {
	defer func() {
		r := recover()
		if r != nil {
			log.Println("Panic recover:", r)
			hc.SendStatus("fail")
		}
	}()

	for {
		msg, err := hc.sock.Recv()
		command := string(msg)
		if err != nil {
			log.Println("Error in receiving in recvLoop", err)
			hc.Ended <- true
			break
		} else {
			log.Println("Received in loop:", command)

			hc.callback(command)
		}
	}
}
