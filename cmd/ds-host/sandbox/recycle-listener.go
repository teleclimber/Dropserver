package sandbox

import (
	"encoding/json"
	"fmt"
	"os"

	"nanomsg.org/go/mangos/v2"
	"nanomsg.org/go/mangos/v2/protocol/pair"

	// register transports...
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	_ "nanomsg.org/go/mangos/v2/transport/ipc"
)

type recycleListener struct {
	sock      *mangos.Socket
	msgCb     func(msg string)
	msgSub    map[string]chan bool
	logClient *record.DsLogClient
}

type msgStruct struct {
	Status   string
	Severity int
	Message  string
}

func newRecycleListener(containerName string, logClient *record.DsLogClient, msgCb func(msg string)) *recycleListener {
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

	rl := recycleListener{
		sock:      &sock,
		msgCb:     msgCb,
		msgSub:    make(map[string]chan bool),
		logClient: logClient}

	go rl.recvLoop()

	return &rl
}

func (rl *recycleListener) recvLoop() {
	sock := *rl.sock
	for {
		rcv, err := sock.Recv()
		if err != nil {
			fmt.Println("Error in receiving in recvLoop", rcv)
			// end <- true
			break
		} else {
			//str := string(msg) // make that a json message, which will need to be parsed.

			var msg msgStruct

			err = json.Unmarshal(rcv, &msg)
			if err != nil {
				rl.logClient.Log(record.ERROR, nil, "recycleListener: Error unmarshalling json message")
				// probably need to shut things down. This is badnews.
			}

			fmt.Println("received recyc msg data", msg)

			if subChan, ok := rl.msgSub[msg.Status]; ok {
				subChan <- true
			}

			// not sure if this is useful after all
			rl.msgCb(msg.Status)

			if msg.Status == "log" {
				//...
				var sev record.LogLevel
				switch msg.Severity {
				case 0:
					sev = record.DEBUG
				case 1:
					sev = record.INFO
				case 2:
					sev = record.WARN
				case 3:
					sev = record.ERROR
				}
				rl.logClient.Log(sev, nil, "From sandbox: "+msg.Message)
			}
		}
	}
}

func (rl *recycleListener) send(msg string) { // return err?
	sock := *rl.sock
	err := sock.Send([]byte(msg))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func (rl *recycleListener) waitFor(msg string) {
	done := make(chan bool)
	rl.msgSub[msg] = done
	<-done
	delete(rl.msgSub, msg)
}
func (rl *recycleListener) close() {
	//(*rl.conn).Close()
}
