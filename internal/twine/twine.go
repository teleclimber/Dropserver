package twine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
)

// Going to try to isolate just the protocol code
//..later? or now?

type serviceID int

// reserved service IDs:
const (
	protocolService   serviceID = 1
	refRequestService serviceID = 4 // New mesage with a reference to open message
	replyService      serviceID = 5 // reply to a message
	closeService      serviceID = 6 // reply with standard OK/Err, or acknowledge reply
	// We may still need an error service?
)

type commandID uint8

// reserved command ids:
const (
	protocolHi    commandID = 1
	protocolOK    commandID = 2
	protocolError commandID = 3
)

// Used by protocol:
const (
	protocolPing     commandID = 4
	protocolPong     commandID = 5
	protocolMsgError commandID = 6 // error at protocol/transport level? Is ther anything there?
	protocolGraceful commandID = 7 // Would like to shutdown
)

// messageMeta is what is returned by read and contains all the raw data
type messageMeta struct {
	service  serviceID
	command  commandID
	msgID    uint8
	refMsgID uint8
	payload  *[]byte
}

// Twine holds everything needed for the protocol to function
type Twine struct {
	isServer   bool
	socketPath string
	ln         *net.Listener // consider making that a ReadWriter so we can adapt anything there.
	conn       *net.Conn
	msgReg     *messageRegistry

	ReadyChan   chan struct{}
	MessageChan chan ReceivedMessageI
	ErrorChan   chan error

	closingMux sync.Mutex
	graceful   bool
	connClosed bool

	writerMux sync.Mutex
}

// Big question is how we communicate with outside world
// Maybe a chanel with type that encapsulates whole message?

// NewServer creates a new Twine struct configured to be a server
func NewServer(sockPath string) (*Twine, error) {
	t := makeNew(sockPath)
	t.isServer = true
	t.msgReg.firstMsgID = 128
	t.msgReg.lastMsgID = 255
	t.msgReg.nextID = 128

	// TODO delete it first

	unixListener, err := net.Listen("unix", t.socketPath)
	if err != nil {
		return nil, err
	}

	t.ln = &unixListener

	go t.startServer()

	return t, nil
}

// NewClient creates a new Twine struct configured to be a client
func NewClient(sockPath string) *Twine {
	t := makeNew(sockPath)
	t.isServer = false
	t.msgReg.firstMsgID = 1
	t.msgReg.lastMsgID = 127
	t.msgReg.nextID = 1

	go t.startClient()

	return t
}

func makeNew(sockPath string) *Twine { // NewServer or NewClient
	t := &Twine{
		socketPath:  sockPath,
		ReadyChan:   make(chan struct{}),
		MessageChan: make(chan ReceivedMessageI),
		ErrorChan:   make(chan error),
		msgReg: &messageRegistry{
			messages: make(map[uint8]*msg),
		},
	}
	return t
}

func (t *Twine) startServer() {
	ln := *t.ln

	revConn, err := ln.Accept() // This blocks until aconn shows up
	if err != nil {
		//t.ErrorChan <- errors.New("error accepting rev conn") // causes race condition on close
		go t.close()
		return
	}
	// ^^ here we assume that there is a single client

	t.conn = &revConn

	m, err := t.receiveMessage()
	if err != nil {
		t.ErrorChan <- err
		go t.close()
		return
	}

	if m.service != protocolService || m.command != protocolHi {
		t.ErrorChan <- errors.New("first command not HI")
		payload := []byte("first command not HI")
		t.send(int(m.msgID), 0, m.service, int(protocolError), &payload)
		go t.close() //close(t.ReadyChan)
		return
	}

	// treat as regular message?
	// sent reply with ok? So the other side can also know that everything's good to go?

	err = t.send(int(m.msgID), 0, closeService, int(protocolOK), nil)
	if err != nil {
		t.ErrorChan <- err
		go t.close() //close(t.ReadyChan)
		return
	}

	t.ReadyChan <- struct{}{}

	go t.receive()
}

func (t *Twine) startClient() {
	conn, err := net.Dial("unix", t.socketPath)
	if err != nil {
		t.ErrorChan <- err // failstart
		return
	}

	t.conn = &conn

	go t.receive()

	err = t.sendHi()
	if err != nil {
		t.ErrorChan <- err // failstart
		return
	}

	// then wait for "OK" from server?

	t.ReadyChan <- struct{}{}
}

// startClient
// Read THIS:: https://johnrefior.com/gobits/read?post=12
func (t *Twine) receive() {
	for {
		raw, err := t.receiveMessage()
		if err != nil {
			// t.messagesMux.Lock()
			// if !t.graceful {
			// 	t.ErrorChan <- err
			// }
			// t.messagesMux.Unlock()
			break
		}

		if raw.service == protocolService {
			go t.handleProtocolCmd(raw)
			continue
		}

		if raw.service == refRequestService {
			refMsgData, err := t.msgReg.getOpenMessage(raw.refMsgID)
			if err != nil {
				go t.sendMsgClosed(raw.msgID) // the error is with the message sent, not the ref
				t.ErrorChan <- err
				continue
			}
			msgData, err := t.msgReg.registerMessage(raw)
			if err != nil {
				// We probably need to send something back with the error.
				t.ErrorChan <- err
				continue
			}
			// hang on, This is a new message, so we need to treat it as such!!
			message := t.makeMessage(raw, msgData)
			message.service = int(refMsgData.service)
			refMsgData.refRequestMux.Lock()
			if refMsgData.refChanOpen {
				refMsgData.refRequestsChan <- message
			} else {
				go message.SendError("No listener for ref request")
			}
			refMsgData.refRequestMux.Unlock()
		} else if raw.service == replyService {
			msgData, err := t.msgReg.closeMessage(raw.msgID) // since this is a reply, this was an **outgoing** message id
			if err != nil {
				// How to handle? With new simple single-reply, there is little reason for this to happen unless something truly borked.
				t.ErrorChan <- err
				continue
			}
			message := t.makeMessage(raw, nil) // don't pass ref msg since it's a reply
			message.service = int(msgData.service)

			msgData.replyChan <- message

		} else if raw.service == closeService { // handles OK and Error messages
			msgData, err := t.msgReg.getMessageData(raw.msgID)
			if err != nil {
				// This should not happen in the normal course of things.
				t.ErrorChan <- err
				continue
			}
			if t.msgReg.msgIDIsLocal(raw.msgID) { // This is a reply to a sent message. It should be open.
				if msgData.closed {
					t.ErrorChan <- errors.New("Message is closed")
					continue
				}
			} else {
				if !msgData.closed {
					t.ErrorChan <- errors.New("Received reply acknowledgement on open message")
					continue
				}
			}
			err = t.msgReg.unregisterMessage(raw.msgID)
			if err != nil {
				// This should not happen in the normal course of things.
				t.ErrorChan <- err
				continue
			}
			message := t.makeMessage(raw, nil) // we pass a message, but do not connect any ref msg data because the message is at end of life
			message.service = int(msgData.service)
			msgData.replyChan <- message

		} else {
			// Brand new message, check we're not graceful, then register message id
			msgData, err := t.msgReg.registerMessage(raw) // this is **incoming** message, maybe check it's in the right range
			if err != nil {
				// We probably need to send something back with the error.
				t.ErrorChan <- err
				continue
			}

			if raw.service != protocolService {
				message := t.makeMessage(raw, msgData)
				t.closingMux.Lock()
				if t.graceful {
					// message received while we are terminating.
					// Can happen in normal course of things.
					go message.SendError("terminating")
				} else {
					t.MessageChan <- message
				}
				t.closingMux.Unlock()
			}
		}
	}
}

func (t *Twine) receiveMessage() (*messageMeta, error) {

	serviceBytes, err := t.read(1) // let's read more than that?
	if err != nil {
		return nil, err
	}
	service := serviceID(serviceBytes[0])

	cmdBytes, err := t.read(1)
	if err != nil {
		return nil, err
	}
	cmd := commandID(cmdBytes[0])

	msgIDBytes, err := t.read(1)
	if err != nil {
		return nil, err
	}
	msgID := uint8(msgIDBytes[0])

	// if it's a new message referencing an old message, get the referenced message id
	refMsgID := uint8(0)
	if service == refRequestService {
		msgIDBytes, err = t.read(1)
		if err != nil {
			return nil, err
		}
		refMsgID = uint8(msgIDBytes[0])
		// check that it's not zero
	}

	sizeBytes, err := t.read(2) // two bytes for size
	if err != nil {
		return nil, err
	}
	var size int
	sizeSmol := binary.BigEndian.Uint16(sizeBytes)
	if sizeSmol == 0xff {
		sizeBytesBig, err := t.read(4) // four for big ... that's 4Gigabytes!!!!!! :/
		if err != nil {
			return nil, err
		}
		size = int(binary.BigEndian.Uint32(sizeBytesBig))
	} else {
		size = int(sizeSmol)
	}

	payloadBytes, err := t.readToPtr(size)
	if err != nil {
		return nil, err
	}

	// should probably have a closer byte, just as a check.

	return &messageMeta{
		service:  service,
		command:  cmd,
		msgID:    msgID,
		refMsgID: refMsgID,
		payload:  payloadBytes, // Should this not be a reader?
	}, nil
}

func (t *Twine) read(size int) ([]byte, error) {
	rc := *t.conn
	p := make([]byte, size)
	_, err := rc.Read(p)
	if err != nil {
		return p, err
	}

	return p, nil
}

// readToPtr reads from conn but returns a pointer to byte arrray
func (t *Twine) readToPtr(size int) (*[]byte, error) {
	rc := *t.conn
	p := make([]byte, size)
	_, err := rc.Read(p)
	if err != nil {
		return &p, err
	}

	return &p, nil
}

/// sends
// There are a few send possibilities:
// - Send, initiated here with new message id
// - Reply, as a response to a msg, using that msg id and closing it.
// - RefSend, a new message that references an existing one

// Send a new message
// Shouldn't you get a callback, or should you pass a callback?
func (t *Twine) Send(servInt int, cmd int, payload *[]byte) (SentMessageI, error) {
	serviceID := serviceID(servInt)
	newMsgID, newMsg, err := t.msgReg.newMessage(serviceID) // should maybe return an error in case no message ids left
	if err != nil {
		return nil, err
	}

	m := Message{
		command: cmd,
		service: servInt,
		msgID:   newMsgID,
		msg:     newMsg,
		t:       t}

	err = t.send(newMsgID, 0, serviceID, cmd, payload)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// SendBlock sends a message and waits for the response before returning
// An error is returned iw the other side sent an error code.
func (t *Twine) SendBlock(servInt int, cmd int, payload *[]byte) (ReceivedReplyI, error) {
	serviceID := serviceID(servInt)
	newMsgID, newMsg, err := t.msgReg.newMessage(serviceID) // should maybe return an error in case no message ids left
	if err != nil {
		return nil, err
	}

	err = t.send(newMsgID, 0, serviceID, cmd, payload)
	if err != nil {
		return nil, err
	}

	reply, ok := <-newMsg.replyChan
	if !ok {
		return nil, errors.New("reply channel closed before reply")
	}

	err = reply.Error()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

// Reply to a message
func (t *Twine) Reply(msgID int, cmd int, payload *[]byte) error {
	msgID8, err := t.msgReg.checkMsgIDRemote(msgID)
	if err != nil {
		return err
	}

	msgData, err := t.msgReg.closeMessage(msgID8)
	if err != nil {
		return err
	}

	// check session is same and still open?
	// check msgID is still open (it should be if this is the reply, but need to be sure they only reply once)

	err = t.send(msgID, 0, replyService, cmd, payload)
	if err != nil {
		return err // something went wrong in the send
	}

	/*reply*/
	_, ok := <-msgData.replyChan
	if !ok {
		return errors.New("reply channel closed before reply")
	}

	// TODO: check reply for errors
	//reply.

	return nil
}

// ReplyClose sends an OK and closes the message
func (t *Twine) ReplyClose(msgID int, ok bool, errStr string) error {
	// This one could be either a local ID or a remote ID.
	// because we can send OK in response to an incoming message

	msgID8, err := t.msgReg.checkMsgIDRange(msgID)
	if err != nil {
		return err
	}

	msgData, err := t.msgReg.getMessageData(msgID8)
	if err != nil {
		return fmt.Errorf("ReplyOKClose: %w", err)
	}

	// If the message ID is local, the remote sent a reply, whicih would have closed the message
	// If the message ID is remote, then we're sending OK as the original reply, so messgae has to be open
	if t.msgReg.msgIDIsLocal(msgID8) {
		if !msgData.closed {
			return errors.New("expected to send OK on closed message")
		}
	} else {
		if msgData.closed {
			return errors.New("msg ID is closed")
		}
	}

	cmd := protocolOK
	var payload []byte
	if !ok {
		cmd = protocolError
		payload = []byte(errStr)
	}

	err = t.send(msgID, 0, closeService, int(cmd), &payload) // cmd is 0 on ok close?
	if err != nil {
		t.ErrorChan <- err
	}

	err = t.msgReg.unregisterMessage(msgID8)
	if err != nil {
		t.ErrorChan <- err
	}

	return nil
}

// RefRequest sneds a new message with a reference to an open message
func (t *Twine) RefRequest(refID int, cmd int, payload *[]byte) (SentMessageI, error) {
	refMsgID8, err := t.msgReg.checkMsgIDRange(refID)
	if err != nil {
		return nil, err
	}

	refMsgData, err := t.msgReg.getOpenMessage(refMsgID8)
	if err != nil {
		return nil, err
	}

	if refMsgData.closed {
		return nil, errors.New("Message ID is closed")
	}

	newMsgID, newMsg, err := t.msgReg.newMessage(refMsgData.service)
	if err != nil {
		return nil, err
	}

	m := Message{
		command:  cmd,
		service:  int(refRequestService),
		msgID:    newMsgID,
		refMsgID: refID,
		msg:      newMsg,
		t:        t}

	err = t.send(newMsgID, refID, refRequestService, cmd, payload)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// RefRequestBlock sends a new message with a reference to an open message
// and blocks until a reply is received
func (t *Twine) RefRequestBlock(refID int, cmd int, payload *[]byte) (ReceivedReplyI, error) {
	refMsgID8, err := t.msgReg.checkMsgIDRange(refID)
	if err != nil {
		return nil, err
	}

	refMsgData, err := t.msgReg.getOpenMessage(refMsgID8)
	if err != nil {
		return nil, err
	}

	if refMsgData.closed {
		return nil, errors.New("Message ID is closed")
	}

	newMsgID, newMsg, err := t.msgReg.newMessage(refMsgData.service)
	if err != nil {
		return nil, err
	}

	err = t.send(newMsgID, refID, refRequestService, cmd, payload)
	if err != nil {
		return nil, err
	}

	reply, ok := <-newMsg.replyChan
	if !ok {
		return nil, errors.New("reply channel closed before reply")
	}

	return reply, nil
}

// A ref msg attached to an incoming message was closed
// This is normal course of things given either side can close messages.
// The message id is the message that refered to the closed message, not the closed message.
func (t *Twine) sendMsgClosed(msgID uint8) { // do we send any kind of error?
	t.send(int(msgID), 0, protocolService, int(protocolMsgError), nil)
}

// func (t *Twine) sendMsgError(msgID uint8) { // do we send any kind of error?
// 	t.send(msgID, protocolService, uint8(protocolMsgError), nil)
// }

func (t *Twine) sendHi() error {
	_, err := t.SendBlock(int(protocolService), int(protocolHi), nil)
	if err != nil {
		return err
	}

	return nil
}

func (t *Twine) sendPing() error {
	reply, err := t.SendBlock(int(protocolService), int(protocolPing), nil)
	if err != nil {
		return err
	}

	if reply.CommandID() != int(protocolPong) {
		return fmt.Errorf("response to Ping was not Pong: %v", reply.CommandID())
	}

	err = reply.SendOK()
	if err != nil {
		return err
	}

	return nil
}

// send is the low level send function
// But I think we established that if service is 5 or 6, that's ref-msg.
// command gets sent, it's just ath service is 5 or 6, and ctual service will be looked up by client.
func (t *Twine) send(msgID int, refMsgID int, service serviceID, cmd int, payload *[]byte) error {
	// this will probably have to return an error.
	if service < 1 || service > 0xff {
		return fmt.Errorf("service id is out of bounds: %v", service)
	}

	if cmd < 0 || cmd > 0xff {
		return fmt.Errorf("cmd id is out of bounds: %v", cmd)
	}

	if msgID < 0 || msgID > 0xff { // allow zero to send Bye
		return fmt.Errorf("send: message id is out of bounds: %v", msgID)
	}

	// should we use a mutex to lock access to socket
	// .. to ensure we don't have interleaved data when writing simultaneously

	metaBytes := make([]byte, 0, 10)
	metaBytes = append(metaBytes, uint8(service))
	metaBytes = append(metaBytes, uint8(cmd))
	metaBytes = append(metaBytes, uint8(msgID))

	if service == refRequestService {
		if refMsgID < 1 || refMsgID > 0xff {
			return fmt.Errorf("send: ref message id is out of bounds: %v", refMsgID)
		}
		metaBytes = append(metaBytes, uint8(refMsgID))
	}

	// payload size
	size := 0
	if payload != nil {
		size = len(*payload)
	}
	bSmol := make([]byte, 2)
	if size >= 0xff {
		binary.BigEndian.PutUint16(bSmol, 0xff)
		metaBytes = append(metaBytes, bSmol...)

		bBig := make([]byte, 4)
		binary.BigEndian.PutUint32(bBig, uint32(size))
		metaBytes = append(metaBytes, bBig...)
	} else {
		binary.BigEndian.PutUint16(bSmol, uint16(size))
		metaBytes = append(metaBytes, bSmol...)
	}

	// I don't know that this is needed.
	t.writerMux.Lock()
	defer t.writerMux.Unlock()

	rc := *t.conn

	_, err := rc.Write(metaBytes)
	if err != nil {
		// this is a problem.
		// probably need to kill things off at this point.
		return errors.New("error writing to socket")
	}

	if payload != nil {
		_, err = rc.Write(*payload)
		if err != nil {
			// this is a problem.
			// probably need to kill things off at this point.
			return errors.New("error writing to socket")
		}
	}

	return nil
}

// makeMessage returns a *Message populated with the data from messageData.
func (t *Twine) makeMessage(raw *messageMeta, ref *msg) *Message {
	return &Message{
		service:  int(raw.service),
		command:  int(raw.command),
		msgID:    int(raw.msgID),
		refMsgID: int(raw.refMsgID),
		payload:  raw.payload,
		msg:      ref,
		t:        t}
}

func (t *Twine) handleProtocolCmd(raw *messageMeta) {
	switch raw.command {
	// 1 is "hi", handled separately
	case protocolGraceful:
		newMsg, err := t.msgReg.registerMessage(raw)
		message := t.makeMessage(raw, newMsg)
		if err != nil {
			t.ErrorChan <- err
			// bail presumably
		}
		t.receivedGraceful(message)

	case protocolPing:
		newMsg, err := t.msgReg.registerMessage(raw)
		message := t.makeMessage(raw, newMsg)
		if err != nil {
			t.ErrorChan <- err
		}
		err = message.Reply(int(protocolPong), nil)
		if err != nil {
			t.ErrorChan <- err
		}

	default:
		t.ErrorChan <- fmt.Errorf("unrecognized command for protocol service: %v", raw.command)
	}

	return
}

// Graceful stops new incoming requests and
// waits for all messages to terminate before shutting down.
func (t *Twine) Graceful() {
	_, err := t.SendBlock(int(protocolService), int(protocolGraceful), nil)
	if err != nil {
		t.ErrorChan <- err
		//t.close() //no point in trying to salvage this situation?
	}

	err = t.doGraceful()
	if err != nil {
		// assume it's "already closing" error for now.
		return
	}

	// we expect OK or Error, so if it returned without error we can proceed.

}
func (t *Twine) receivedGraceful(received ReceivedMessageI) {
	// maybe check to make sure we're not already closing?
	err := t.doGraceful()
	if err != nil {
		//check if error is already closing. If so ignore it, sned OK anyways.
	}

	err = received.SendOK()
	if err != nil {
		t.ErrorChan <- err
	}
}

func (t *Twine) doGraceful() (err error) { // this should return a channel so it can return quickly but still let caller know it's been done
	t.closingMux.Lock()
	if !t.graceful {
		t.graceful = true
	} else {
		err = errors.New("already graceful")
	}
	t.closingMux.Unlock()

	go func() {
		t.msgReg.waitAllUnregistered()
		t.close()
	}()

	return
}

// Stop kills twine without trying to be nice about it.
func (t *Twine) Stop() {
	t.close()
}

func (t *Twine) close() {
	t.closingMux.Lock()
	defer t.closingMux.Unlock()
	if !t.connClosed {
		t.connClosed = true

		if t.conn != nil {
			rc := *t.conn
			rc.Close() //might return an error
		}

		if t.ln != nil {
			ln := *t.ln
			ln.Close()
		}

		close(t.ReadyChan)
		close(t.MessageChan)
		close(t.ErrorChan)
	}
}
