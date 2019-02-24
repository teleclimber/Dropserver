package containers

import (
	"fmt"
	"nanomsg.org/go/mangos/v2"
	"nanomsg.org/go/mangos/v2/protocol/pair"
	"os"
	// register transports...
	_ "nanomsg.org/go/mangos/v2/transport/ipc"
)

type recycleListener struct {
	sock   *mangos.Socket
	msgCb  func(msg string)
	msgSub map[string]chan bool
}

func newRecycleListener(containerName string, msgCb func(msg string)) *recycleListener {
	recyclerSockPath := "/home/developer/ds-socket-proxies/recycler-" + containerName
	err := os.Remove(recyclerSockPath)
	if err != nil {
		fmt.Println(err)
		//os.Exit(1)	// don't exit. if file didn't exist it errs.
	}

	sock, err := pair.NewSocket()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = sock.Listen("ipc://" + recyclerSockPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rl := recycleListener{sock: &sock, msgCb: msgCb, msgSub: make(map[string]chan bool)}

	go rl.recvLoop()

	return &rl
}

func (rl *recycleListener) recvLoop() {
	sock := *rl.sock
	for {
		msg, err := sock.Recv()
		if err != nil {
			fmt.Println("Error in receiving in recvLoop", err)
			// end <- true
			break
		} else {
			command := string(msg)
			fmt.Println("Received in loop:", command)

			if subChan, ok := rl.msgSub[command]; ok {
				subChan <- true
			}

			// not sure if this is useful after all
			rl.msgCb(command)
		}
	}
}

func (rl *recycleListener) send(msg string) { // return err?
	sock := *rl.sock
	fmt.Println("Sending message", msg)
	err := sock.Send([]byte(msg))
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
	//(*rl.conn).Close()
}
