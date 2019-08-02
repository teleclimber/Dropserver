package sandbox

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// This is really appspaceAPIServer or something like that
// it should be in its own module? (maybe not?)
// There is one server for all sandboxes. -> nope
// -> Somehow have to identify sandboxes even though you can't trust them.
// -> only reliable solution is to use unix sockets, one per sandbox?
// -> meaning one per live sandbox.
// Use this for http over unix domain socket:
// https://gist.github.com/teknoraver/5ffacb8757330715bcbcc90e6d46ac74

// Right now we have a really clumsy messageing system that essentially has no future
// I just need to send a port over and I'm not sure I can do it easily.
// Ultimately, if this is the system that accepts appsapceAPI requests,
// ..it will need to be far more capable, and probably HTTP based?
// can we assume it will be its own package?

type reverseListener struct { //do we really need two distinct types here?
	ln         *net.Listener
	conn       *net.Conn
	socketPath string
	msgSub     map[string]chan bool
	msgCb      func(msg string)
}

//func initializeSockets() ...?

func newReverseListener(config *domain.RuntimeConfig, ID int, msgCb func(msg string)) *reverseListener {
	rl := reverseListener{
		socketPath: path.Join(config.Sandbox.SocketsDir, fmt.Sprintf("%d.sock", ID)),
		msgSub:     make(map[string]chan bool),
		msgCb:      msgCb}

	// I thgink we shold also create the directory just in case it's not there?
	// Or we need a general initialization function that sets the directory up and removes everything
	// ..so that we don't delay things here.

	if err := os.RemoveAll(rl.socketPath); err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("unix", rl.socketPath)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	rl.ln = &ln

	go rl.start()

	return &rl
}
func (rl *reverseListener) start() {
	ln := *rl.ln

	revConn, err := ln.Accept() // This blocks until aconn shows up
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rl.conn = &revConn

	go func(rc net.Conn) {
		p := make([]byte, 4)
		for {
			n, err := rc.Read(p)
			if err != nil {
				if err == io.EOF {
					fmt.Println("got EOF from reverse conn, closing", string(p[:n]))
					err := rc.Close()
					if err != nil {
						fmt.Println("error clsing rev conn after EOF")
					}
					break
				} else {
					fmt.Println(err)
					os.Exit(1)
				}
			} else {
				command := string(p[:n])
				//fmt.Println("reverse listener got message", command)
				if subChan, ok := rl.msgSub[command]; ok {
					subChan <- true
				}
				rl.msgCb(command)
			}
		}
	}(revConn)
}
func (rl *reverseListener) send(msg string) { // return err?
	_, err := (*rl.conn).Write([]byte(msg))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func (rl *reverseListener) waitFor(msg string) {
	done := make(chan bool)
	rl.msgSub[msg] = done
	<-done
	delete(rl.msgSub, msg)
}
func (rl reverseListener) close() {
	//conn.end() or some such
	// err := os.Remove(rl.sockPath)
	// if err != nil {
	// 	fmt.Println(err)
	// 	//os.Exit(1)	// don't exit. if file didn't exist it errs.
	// }
}
