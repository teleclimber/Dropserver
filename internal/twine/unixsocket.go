package twine

import (
	"errors"
	"net"
)

// Unix is the unix domain socket transport for Twine
type Unix struct {
	messageChan chan (messageMeta)
	errorChan   chan (error)
	socketPath  string
	ln          *net.Listener // consider making that a ReadWriter so we can adapt anything there.
	conn        *net.Conn
}

// Make conn something we can easily put up for tests?
// Basically need Read, Write and Close.

func newUnixServer(sockPath string) (*Unix, error) {
	u := &Unix{
		messageChan: make(chan messageMeta),
		errorChan:   make(chan error),
		socketPath:  sockPath}

	// TODO delete it first

	unixListener, err := net.Listen("unix", u.socketPath)
	if err != nil {
		return nil, err
	}

	u.ln = &unixListener

	return u, nil
}

// StartServer blocks until a peer has connected
func (u *Unix) StartServer() error {
	ln := *u.ln
	conn, err := ln.Accept() // This blocks until aconn shows up
	if err != nil {
		return err
	}
	// ^^ here we assume that there is a single client
	u.conn = &conn

	go u.readLoop()

	return nil
}

//
func newUnixClient(sockPath string) (*Unix, error) {
	u := &Unix{
		messageChan: make(chan messageMeta),
		errorChan:   make(chan error),
		socketPath:  sockPath}
	conn, err := net.Dial("unix", u.socketPath)
	if err != nil {
		return nil, err
	}
	u.conn = &conn

	go u.readLoop()

	return u, nil
}

// ReadMessage returns an incoming message after it has arrived
func (u *Unix) ReadMessage() (*messageMeta, error) {
	select {
	case msg := <-u.messageChan:
		return &msg, nil
	case err := <-u.errorChan:
		return &messageMeta{}, err
	}
}

func (u *Unix) readLoop() {
	rc := *u.conn
	readBuf := make([]byte, 0, 100)
	doRead := true
	for {
		if doRead {
			readSize, err := rc.Read(readBuf[len(readBuf):cap(readBuf)])
			if err != nil {
				u.errorChan <- err
				break
			}
			readBuf = readBuf[:len(readBuf)+readSize]
		}

		doRead = false
		decoded, remainder, err := decodeMessage(readBuf)
		if err == errMessageIncomplete {
			doRead = true
			continue
		} else if err != nil {
			u.errorChan <- err
			break
		}

		copy(readBuf, remainder)
		readBuf = readBuf[:len(remainder)]

		msg := messageMeta{
			service:  decoded.service,
			command:  decoded.command,
			msgID:    decoded.msgID,
			refMsgID: decoded.refMsgID}

		if decoded.payloadSize > 0 {
			msg.payload = make([]byte, decoded.payloadSize)

			remainderPayloadSize := len(readBuf)
			if decoded.payloadSize < remainderPayloadSize {
				remainderPayloadSize = decoded.payloadSize
			}
			copy(msg.payload, readBuf[:remainderPayloadSize])

			readBuf = readBuf[remainderPayloadSize:]

			if decoded.payloadSize > remainderPayloadSize {
				payloadLeftSize := decoded.payloadSize - remainderPayloadSize
				readSize, err := rc.Read(msg.payload[remainderPayloadSize:])
				if err != nil {
					u.errorChan <- err
				}
				if readSize < payloadLeftSize {
					u.errorChan <- err
				}
			}
		}
		u.messageChan <- msg
	}
}

// WriteMessage sends a message on the connection
func (u *Unix) WriteMessage(metaBytes []byte, payload []byte) error {
	rc := *u.conn

	_, err := rc.Write(metaBytes)
	if err != nil {
		// this is a problem.
		// probably need to kill things off at this point.
		return errors.New("error writing to socket")
	}

	if payload != nil {
		_, err = rc.Write(payload)
		if err != nil {
			// this is a problem.
			// probably need to kill things off at this point.
			return errors.New("error writing to socket")
		}
	}
	return nil
}

// Close the conn and listener
func (u *Unix) Close() {
	if u.conn != nil {
		rc := *u.conn
		rc.Close() //might return an error
	}

	if u.ln != nil {
		ln := *u.ln
		ln.Close()
	}
}
