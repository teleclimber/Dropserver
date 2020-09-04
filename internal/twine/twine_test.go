package twine

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/teleclimber/DropServer/internal/leaktest"
)

// test new message
// that it loops around correctly, etc...

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
				str := string(*(message.Payload()))
				if str != "hello world" {
					errs <- fmt.Errorf("expected hello world, got %v", str)
				}
				message.SendOK()

			case 22:
				str := string(*(message.Payload()))
				if str != "hello" {
					errs <- fmt.Errorf("expected hello, got %v", str)
				}
				payload := []byte("world")
				err = message.Reply(88, &payload)
				if err != nil {
					errs <- err
				}

			case 33:
				payload := []byte("Elo")
				_, err = message.RefSendBlock(33, &payload)
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
		reply, err := tc.SendBlock(11, 99, &payload)
		if err != nil {
			errs <- err
		}
		if !reply.OK() {
			errs <- fmt.Errorf("expected OK for 11/99")
		}

		// test Send with Reply
		payload = []byte("hello")
		sent, err := tc.Send(22, 99, &payload)
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
		if string(*repBytes) != "world" {
			errs <- fmt.Errorf("reply was %v", string(*repBytes))
		}
		err = reply.SendOK()
		if err != nil {
			errs <- err
		}

		// test ref requests
		payload = []byte("hello")
		sent, err = tc.Send(33, 99, &payload)
		if err != nil {
			errs <- err
		}
		go func() {
			for refMessage := range sent.GetRefRequestsChan() {
				refBytes := refMessage.Payload()
				if string(*refBytes) != "Elo" {
					errs <- fmt.Errorf("reply was %v", string(*repBytes))
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
