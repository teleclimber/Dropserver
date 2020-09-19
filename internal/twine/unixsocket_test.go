package twine

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestReadLoop(t *testing.T) {
	writeErr := make(chan error)

	uConn, testConn := net.Pipe()
	u := &Unix{
		messageChan: make(chan messageMeta),
		conn:        &uConn,
	}

	payload := []byte("hello world")
	refMsg := messageMeta{
		service: serviceID(7),
		command: commandID(11),
		msgID:   15,
		payload: payload}

	enc, err := encodeMessage(int(refMsg.msgID), int(refMsg.refMsgID), refMsg.service, refMsg.command, payload)
	if err != nil {
		t.Error(err)
	}

	go func() {
		_, err := testConn.Write(append(enc, payload...))
		if err != nil {
			writeErr <- err
		}

		testConn.Close()

		close(writeErr)
	}()

	go u.readLoop()

	m := <-u.messageChan
	err = verifyMessageMetaEqual(m, refMsg)
	if err != nil {
		t.Error(err)
	}

	we := <-writeErr
	if we != nil {
		t.Error(we)
	}
}

func TestReadLoopPartialMessage(t *testing.T) {
	writeErr := make(chan error)

	uConn, testConn := net.Pipe()
	u := &Unix{
		messageChan: make(chan messageMeta),
		conn:        &uConn,
	}

	payload := []byte("hello world")
	refMsg := messageMeta{
		service: serviceID(7),
		command: commandID(11),
		msgID:   15,
		payload: payload}

	enc, err := encodeMessage(int(refMsg.msgID), int(refMsg.refMsgID), refMsg.service, refMsg.command, payload)
	if err != nil {
		t.Error(err)
	}

	go func() {
		_, err := testConn.Write(enc[:3])
		if err != nil {
			writeErr <- err
		}

		time.Sleep(time.Millisecond * 100)

		_, err = testConn.Write(append(enc[3:], payload...))
		if err != nil {
			writeErr <- err
		}

		testConn.Close()

		close(writeErr)
	}()

	go u.readLoop()

	m := <-u.messageChan
	err = verifyMessageMetaEqual(m, refMsg)
	if err != nil {
		t.Error(err)
	}

	we := <-writeErr
	if we != nil {
		t.Error(we)
	}
}

func TestReadLoopPartialPayload(t *testing.T) {
	writeErr := make(chan error)

	uConn, testConn := net.Pipe()
	u := &Unix{
		messageChan: make(chan messageMeta),
		conn:        &uConn,
	}

	payload := []byte("hello world")
	refMsg := messageMeta{
		service: serviceID(7),
		command: commandID(11),
		msgID:   15,
		payload: payload}

	enc, err := encodeMessage(int(refMsg.msgID), int(refMsg.refMsgID), refMsg.service, refMsg.command, payload)
	if err != nil {
		t.Error(err)
	}

	go func() {
		_, err := testConn.Write(append(enc, payload[:4]...))
		if err != nil {
			writeErr <- err
		}

		time.Sleep(time.Millisecond * 100)

		_, err = testConn.Write(payload[4:])
		if err != nil {
			writeErr <- err
		}

		testConn.Close()

		close(writeErr)
	}()

	go u.readLoop()

	m := <-u.messageChan
	err = verifyMessageMetaEqual(m, refMsg)
	if err != nil {
		t.Error(err)
	}

	we := <-writeErr
	if we != nil {
		t.Error(we)
	}
}

func TestDoubleMessages(t *testing.T) {
	writeErr := make(chan error)

	uConn, testConn := net.Pipe()
	u := &Unix{
		messageChan: make(chan messageMeta),
		conn:        &uConn,
	}

	refMsg1 := messageMeta{
		service: serviceID(7),
		command: commandID(11),
		msgID:   15,
		payload: nil}

	payload2 := []byte("goodbye")
	refMsg2 := messageMeta{
		service: serviceID(7),
		command: commandID(11),
		msgID:   15,
		payload: payload2}

	enc1, err := encodeMessage(int(refMsg1.msgID), int(refMsg1.refMsgID), refMsg1.service, refMsg1.command, nil)
	if err != nil {
		t.Error(err)
	}
	enc2, err := encodeMessage(int(refMsg2.msgID), int(refMsg2.refMsgID), refMsg2.service, refMsg2.command, payload2)
	if err != nil {
		t.Error(err)
	}

	go func() {
		send := append(enc1, enc2...)
		send = append(send, payload2...)
		_, err := testConn.Write(send)
		if err != nil {
			writeErr <- err
		}

		testConn.Close()

		close(writeErr)
	}()

	go u.readLoop()

	m := <-u.messageChan
	err = verifyMessageMetaEqual(m, refMsg1)
	if err != nil {
		t.Error(err)
	}

	m2 := <-u.messageChan
	err = verifyMessageMetaEqual(m2, refMsg2)
	if err != nil {
		t.Error(err)
	}

	we := <-writeErr
	if we != nil {
		t.Error(we)
	}
}

// This is literally not testing anything about unxsocket, but is testing the test apparatus
func TestMeta(t *testing.T) {
	rConn, wConn := net.Pipe()

	done := make(chan struct{})

	var readN int

	go func() {
		b := make([]byte, 10)
		var err error
		readN, err = rConn.Read(b)
		if err != nil {
			fmt.Println(err.Error())
		}

		done <- struct{}{}
	}()

	time.Sleep(time.Second)

	fmt.Println("writing")
	n, err := wConn.Write([]byte("Elo"))
	if err != nil {
		fmt.Println("write error", err.Error())
	}
	fmt.Println("write", n)

	<-done

	if readN != 3 {
		t.Error("readN not 3")
	}
}
