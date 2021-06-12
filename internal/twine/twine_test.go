package twine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/teleclimber/DropServer/internal/leaktest"
)

func TestEncodeDecode(t *testing.T) {
	payload := make([]byte, 12)
	refMsg := decodedMessage{
		service:     serviceID(7),
		command:     commandID(11),
		msgID:       15,
		payloadSize: len(payload)}

	enc, err := encodeMessage(int(refMsg.msgID), int(refMsg.refMsgID), refMsg.service, refMsg.command, payload)
	if err != nil {
		t.Error(err)
	}

	msg, remainder, err := decodeMessage(enc)
	if err != nil {
		t.Error(err)
	}
	if len(remainder) != 0 {
		t.Error("expeted 0 remainder")
	}

	err = verifyDecodedEqual(refMsg, msg)
	if err != nil {
		t.Error(err)
	}
}

// need more encode/deocde tests

func TestDecodeMessage(t *testing.T) {
	refMsg := decodedMessage{
		service:     serviceID(7),
		command:     commandID(11),
		msgID:       15,
		payloadSize: 60000}

	metaBytes := make([]byte, 5, 10)

	metaBytes[0] = uint8(refMsg.service)
	metaBytes[1] = uint8(refMsg.command)
	metaBytes[2] = uint8(refMsg.msgID)

	binary.BigEndian.PutUint16(metaBytes[3:5], uint16(refMsg.payloadSize))

	msg, remainder, err := decodeMessage(metaBytes)
	if err != nil {
		t.Error(err)
	}
	if len(remainder) != 0 {
		t.Error("expected empty remainder")
	}
	err = verifyDecodedEqual(refMsg, msg)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeMessageRefRequest(t *testing.T) {
	refMsg := decodedMessage{
		service:     refRequestService,
		command:     commandID(11),
		msgID:       15,
		refMsgID:    77,
		payloadSize: 60000}

	metaBytes := make([]byte, 6, 10)

	metaBytes[0] = uint8(refMsg.service)
	metaBytes[1] = uint8(refMsg.command)
	metaBytes[2] = uint8(refMsg.msgID)
	metaBytes[3] = uint8(refMsg.refMsgID)

	binary.BigEndian.PutUint16(metaBytes[4:6], uint16(refMsg.payloadSize))

	msg, remainder, err := decodeMessage(metaBytes)
	if err != nil {
		t.Error(err)
	}
	if len(remainder) != 0 {
		t.Error("expected empty remainder")
	}
	err = verifyDecodedEqual(refMsg, msg)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeMessageRemainder(t *testing.T) {
	refMsg := decodedMessage{
		service:     serviceID(7),
		command:     commandID(11),
		msgID:       15,
		payloadSize: 60000}

	metaBytes := make([]byte, 10)

	metaBytes[0] = uint8(refMsg.service)
	metaBytes[1] = uint8(refMsg.command)
	metaBytes[2] = uint8(refMsg.msgID)

	binary.BigEndian.PutUint16(metaBytes[3:5], uint16(refMsg.payloadSize))

	msg, remainder, err := decodeMessage(metaBytes)
	if err != nil {
		t.Error(err)
	}
	if len(remainder) != 5 {
		t.Error("expected remainder length of 5")
	}
	err = verifyDecodedEqual(refMsg, msg)
	if err != nil {
		t.Error(err)
	}
}

func verifyDecodedEqual(a, b decodedMessage) error {
	if a.service != b.service {
		return fmt.Errorf("Service unequal: %v, %v", a.service, b.service)
	}
	if a.command != b.command {
		return fmt.Errorf("command unequal: %v, %v", a.command, b.command)
	}
	if a.msgID != b.msgID {
		return fmt.Errorf("msgID unequal: %v, %v", a.msgID, b.msgID)
	}
	if a.refMsgID != b.refMsgID {
		return fmt.Errorf("refMsgID unequal: %v, %v", a.refMsgID, b.refMsgID)
	}
	if a.payloadSize != b.payloadSize {
		return fmt.Errorf("payloadSize unequal: %v, %v", a.payloadSize, b.payloadSize)
	}
	return nil
}

func verifyMessageMetaEqual(a, b messageMeta) error {
	if a.service != b.service {
		return fmt.Errorf("Service unequal: %v, %v", a.service, b.service)
	}
	if a.command != b.command {
		return fmt.Errorf("command unequal: %v, %v", a.command, b.command)
	}
	if a.msgID != b.msgID {
		return fmt.Errorf("msgID unequal: %v, %v", a.msgID, b.msgID)
	}
	if a.refMsgID != b.refMsgID {
		return fmt.Errorf("refMsgID unequal: %v, %v", a.refMsgID, b.refMsgID)
	}
	if !bytes.Equal(a.payload, b.payload) {
		return fmt.Errorf("payloads different: %v, %v", len(a.payload), len(b.payload))
	}

	return nil
}

func TestServerClient(t *testing.T) {
	defer leaktest.GoroutineLeakCheck(t)()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	sockPath := path.Join(dir, "sock")

	errs := make(chan error, 10)

	ts, err := NewUnixServer(sockPath)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for err := range ts.ErrorChan {
			errs <- fmt.Errorf("Server: %w", err)
		}
	}()

	tc := NewUnixClient(sockPath)
	go func() {
		for err := range tc.ErrorChan {
			errs <- fmt.Errorf("Client: %w", err)
		}
	}()

	_, ok := <-ts.ReadyChan
	if !ok {
		t.Error("ready chan closed prematurely")
	}

	go func() {
		err := ts.sendPing()
		if err != nil {
			errs <- err
		}
	}()

	// then do things.
	go func() {
		_, ok := <-tc.ReadyChan
		if !ok {
			errs <- errors.New("client ready chan closed prematurely")
		}

		// try a ping
		err := tc.sendPing()
		if err != nil {
			errs <- err
		}

		//then switch things off
		tc.Graceful()
	}()

	// wait for both ts and tc to cleanly exit
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		_, ok := <-ts.MessageChan
		if ok {
			errs <- errors.New("server: expected !ok because chan closed")
		} else {
			wg.Done()
		}
	}()
	go func() {
		_, ok := <-tc.MessageChan
		if ok {
			errs <- errors.New("client: expected !ok because chan closed")
		} else {
			wg.Done()
		}
	}()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	select {
	case err := <-errs:
		if err != nil {
			t.Error(err)
		}
	case <-doneCh:
		// done
	}
}

// test messages to outside service
func TestServiceMessages(t *testing.T) {
	defer leaktest.GoroutineLeakCheck(t)()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	sockPath := path.Join(dir, "sock")

	errs := make(chan error, 10)

	var wg sync.WaitGroup
	wg.Add(2)

	ts, err := NewUnixServer(sockPath)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for err := range ts.ErrorChan {
			errs <- fmt.Errorf("Server: %w", err)
		}
		wg.Done()
	}()

	tc := NewUnixClient(sockPath)
	go func() {
		for err := range tc.ErrorChan {
			errs <- fmt.Errorf("Client: %w", err)
		}
		wg.Done()
	}()

	_, ok := <-ts.ReadyChan
	if !ok {
		t.Error("ready chan closed prematurely")
	}

	go func() {
		for message := range ts.MessageChan {
			switch message.ServiceID() {
			case 11:
				if message.CommandID() != 99 {
					errs <- errors.New("expected 99 for command")
				}
				str := string(message.Payload())
				if str != "hello world" {
					errs <- fmt.Errorf("expected hello world, got %v", str)
				}
				message.SendOK()

			case 22:
				str := string(message.Payload())
				if str != "hello" {
					errs <- fmt.Errorf("expected hello, got %v", str)
				}
				payload := []byte("world")
				err = message.Reply(88, payload)
				if err != nil {
					errs <- err
				}

			case 33:
				payload := []byte("Elo")
				_, err = message.RefSendBlock(33, payload)
				if err != nil {
					errs <- err
				}
				err = message.SendOK()
				if err != nil {
					errs <- err
				}

			default:
				errs <- fmt.Errorf("unexpected id for service: %v", message.ServiceID())
			}
		}
	}()

	go func() {
		_, ok := <-tc.ReadyChan
		if !ok {
			errs <- errors.New("client ready chan closed prematurely")
		}

		// test simple SendBlock
		payload := []byte("hello world")
		reply, err := tc.SendBlock(11, 99, payload)
		if err != nil {
			errs <- err
		}
		if !reply.OK() {
			errs <- fmt.Errorf("expected OK for 11/99")
		}

		// test Send with Reply
		payload = []byte("hello")
		sent, err := tc.Send(22, 99, payload)
		if err != nil {
			errs <- err
		}
		reply, err = sent.WaitReply()
		if err != nil {
			errs <- err
		}
		if reply.CommandID() != 88 {
			errs <- fmt.Errorf("expected 88 command id , got %v", reply.CommandID())
		}
		repBytes := reply.Payload()
		if string(repBytes) != "world" {
			errs <- fmt.Errorf("reply was %v", string(repBytes))
		}
		err = reply.SendOK()
		if err != nil {
			errs <- err
		}

		// test ref requests
		payload = []byte("hello")
		sent, err = tc.Send(33, 99, payload)
		if err != nil {
			errs <- err
		}
		go func() {
			for refMessage := range sent.GetRefRequestsChan() {
				refBytes := refMessage.Payload()
				if string(refBytes) != "Elo" {
					errs <- fmt.Errorf("reply was %v", string(repBytes))
				}
				refMessage.SendOK()
			}

			//fmt.Println("exiting refMessages req chan")
		}()
		_, err = sent.WaitReply()
		if err != nil {
			errs <- err
		}

		// Then shut it down
		tc.Graceful()
	}()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	select {
	case err := <-errs:
		if err != nil {
			t.Error(err)
		}
	case <-doneCh:
		// done
	}
}
