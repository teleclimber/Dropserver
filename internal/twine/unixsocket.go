package twine

import (
	"errors"
	"net"
)

// Unix is the unix domain socket transport for Twine
type Unix struct {
	socketPath string
	ln         *net.Listener // consider making that a ReadWriter so we can adapt anything there.
	conn       *net.Conn
}

func newUnixServer(sockPath string) (*Unix, error) {
	u := &Unix{
		socketPath: sockPath}

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

	return nil
}

//
func newUnixClient(sockPath string) (*Unix, error) {
	u := &Unix{
		socketPath: sockPath}
	conn, err := net.Dial("unix", u.socketPath)
	if err != nil {
		return nil, err
	}

	u.conn = &conn

	return u, nil
}

// Read returns size bytes read from the connection
func (u *Unix) Read(size int) ([]byte, error) {
	rc := *u.conn
	p := make([]byte, size)
	_, err := rc.Read(p)
	if err != nil {
		return p, err
	}

	return p, nil
}

// ReadToPtr reads from conn but returns a pointer to byte arrray
func (u *Unix) ReadToPtr(size int) (*[]byte, error) {
	rc := *u.conn
	p := make([]byte, size)
	_, err := rc.Read(p)
	if err != nil {
		return &p, err
	}

	return &p, nil
}

// WriteMessage sends a message on the connection
func (u *Unix) WriteMessage(metaBytes []byte, payload *[]byte) error {

	rc := *u.conn

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
