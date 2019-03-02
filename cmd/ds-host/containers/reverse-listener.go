package containers

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

type reverseListener struct { //do we really need two distinct types here?
	ln     *net.Listener
	conn   *net.Conn
	msgSub map[string]chan bool
	msgCb  func(msg string)
}

func newReverseListener(containerName string, hostIP net.IP, msgCb func(msg string)) *reverseListener {
	hostPort := "[" + hostIP.String() + "%ds-sandbox-" + containerName + "]:45454"

	// now we need to try and listen in a loop because
	// the kernel might still be doing "duplicate address detection"
	var listener net.Listener
	var err error
	for i := 0; i < 10; i++ {
		listener, err = net.Listen("tcp6", hostPort)
		if err == nil {
			break
		}
		fmt.Println("No luck listening, pausing then trying again", containerName)
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		fmt.Println("rev listen err", err)
		os.Exit(1)
	}

	rl := reverseListener{ln: &listener, msgSub: make(map[string]chan bool), msgCb: msgCb}

	return &rl
}
func (rl *reverseListener) waitForConn() {
	ln := *rl.ln
	revConn, err := ln.Accept() // I think this blocks until aconn shows up?
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
