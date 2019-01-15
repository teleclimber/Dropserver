package containers

import (
	"fmt"
	"io"
	"net"
	"os"
)

type recycleListener struct {
	ln     *net.Listener
	conn   *net.Conn
	msgSub map[string]chan bool
}

func newRecycleListener(containerName string, msgCb func(msg string)) *recycleListener {
	recyclerSockPath := "/home/developer/container_sockets/" + containerName + "/recycle.sock"
	err := os.Remove(recyclerSockPath)
	if err != nil {
		fmt.Println(err)
		//os.Exit(1)	// don't exit. if file didn't exist it errs.
	}
	listener, err := net.Listen("unix", recyclerSockPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//defer revListener.Close()	// this looks like it will close the server at end of function
	// ^^ so instead do it when we kill the server

	err = os.Chmod(recyclerSockPath, 0777) //temporary until we figure out our users scenario
	if err != nil {
		fmt.Println(err)
	}

	rl := recycleListener{ln: &listener, msgSub: make(map[string]chan bool)}

	recConn, err := listener.Accept() // I think this blocks until aconn shows up?
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//fmt.Println("recycle connection accepted")

	rl.conn = &recConn

	hi := make(chan bool)
	rl.msgSub["hi"] = hi

	go func() {
		p := make([]byte, 4)
		for {
			n, err := recConn.Read(p)
			if err != nil {
				if err == io.EOF {
					fmt.Println("got EOF from recycle conn", string(p[:n]))
					break
				}
				fmt.Println(err)
				os.Exit(1)
			}
			command := string(p[:n])
			//fmt.Println("recycle listener got message", command)
			if subChan, ok := rl.msgSub[command]; ok {
				subChan <- true
			}

			// not sure if this is useful after all
			msgCb(command)
		}
	}()

	<-hi
	delete(rl.msgSub, "hi")

	return &rl
}
func (rl *recycleListener) send(msg string) { // return err?
	fmt.Println("Sending message", msg)
	_, err := (*rl.conn).Write([]byte(msg))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func (rl *recycleListener) waitFor(msg string) {
	fmt.Println("waiting for", msg)
	done := make(chan bool)
	rl.msgSub[msg] = done
	<-done
	//fmt.Println("DONE waiting for", msg)
	delete(rl.msgSub, msg)
}
func (rl *recycleListener) close() {
	//conn.end() or some such
}
