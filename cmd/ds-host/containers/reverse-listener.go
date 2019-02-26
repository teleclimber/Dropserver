package containers

import (
	"fmt"
	"io"
	"net"
	"os"
)

type reverseListener struct { //do we really need two distinct types here?
	ln     *net.Listener
	conn   *net.Conn
	msgSub map[string]chan bool
}

func newReverseListener(containerName string, hostIP net.IP, msgCb func(msg string)) *reverseListener {
	hostPort := "[" + hostIP.String() + "%ds-sandbox-" + containerName + "]:45454"
	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rl := reverseListener{ln: &listener, msgSub: make(map[string]chan bool)}

	go func(ln net.Listener) {
		revConn, err := ln.Accept() // I think this blocks until aconn shows up?
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("reverse connection accepted")

		rl.conn = &revConn

		go func(rc net.Conn) {
			p := make([]byte, 4)
			for {
				n, err := rc.Read(p)
				if err != nil {
					if err == io.EOF {
						fmt.Println("got EOF from reverse conn, closing this side", string(p[:n]))
						err := rc.Close()
						if err != nil {
							fmt.Println("error clsing rev conn after EOF")
						}
						break
					}
					fmt.Println(err)
					os.Exit(1)
				}
				command := string(p[:n])
				//fmt.Println("reverse listener got message", command)
				if subChan, ok := rl.msgSub[command]; ok {
					subChan <- true
				}
				msgCb(command)
			}
		}(revConn)

	}(listener)

	return &rl
}
func (rl *reverseListener) send(msg string) { // return err?
	fmt.Println("Sending to reverse message", msg)
	//c := *rl.conn
	_, err := (*rl.conn).Write([]byte(msg))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func (rl *reverseListener) waitFor(msg string) {
	fmt.Println("rev waiting for", msg)
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
