package twine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/teleclimber/DropServer/internal/leaktest"
)

func TestBytesPump(t *testing.T) {
	w := &Websocket{}
	w.init()

	defer close(w.rxBytesChan)
	go w.bytesPump()

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

	w.rxBytesChan <- rxBytes{append(enc, payload...), nil}

	msgData := <-w.rxMessageChan

	err = verifyMessageMetaEqual(msgData.msg, refMsg)
	if err != nil {
		t.Error(err)
	}

	w.Close()
}

func TestWsServerClient(t *testing.T) {
	defer leaktest.GoroutineLeakCheck(t)()

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

	serverErr := make(chan error)
	//closeServer := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsServer, err := newWebsocketServer(w, r)
		if err != nil {
			serverErr <- err
		}

		go func() {
			for err := range wsServer.ErrorChan {
				fmt.Println("wsServer.ErrorChan " + err.Error())
			}
		}()

		msg, err := wsServer.ReadMessage()
		if err != nil {
			serverErr <- fmt.Errorf("wsServer.ReadMessage() error: %v", err)
		}

		err = verifyMessageMetaEqual(*msg, refMsg)
		if err != nil {
			serverErr <- err
		}

		err = wsServer.WriteMessage(enc, payload)
		if err != nil {
			serverErr <- fmt.Errorf("wsServer.ReadMessage() error: %v", err)
		}

		//<-closeServer
		//wsServer.Close()

		close(serverErr)
	}))
	defer ts.Close()

	serverURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Error(err)
	}
	wsClient, err := newWebsocketClient(serverURL.Host, "/")
	if err != nil {
		t.Error(err)
	}
	go func() {
		for err := range wsClient.ErrorChan {
			fmt.Println(err.Error())
		}
	}()

	err = wsClient.WriteMessage(enc, payload)
	if err != nil {
		t.Error(err)
	}

	msg, err := wsClient.ReadMessage()
	if err != nil {
		t.Error(err)
	}

	err = verifyMessageMetaEqual(*msg, refMsg)
	if err != nil {
		t.Error(err)
	}

	//close(closeServer)
	wsClient.Close()

	err = <-serverErr
	if err != nil {
		t.Error(err)
	}

	fmt.Println("abc")
}
