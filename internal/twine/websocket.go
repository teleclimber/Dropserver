package twine

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// try to create a wrapper around gorilla websocket that is common with unixsockets.

// consts borrowed from gorilla/websckets chat example
// https://github.com/gorilla/websocket/blob/master/examples/chat/client.go

type websocketConstants struct {
	// Time allowed to write a message to the peer.
	writeWait time.Duration

	// Time allowed to read the next pong message from the peer.
	pongWait time.Duration

	// Maximum message size allowed from peer.
	maxMessageSize int64
}

// Send pings to peer with this period. Must be less than pongWait.
func (c *websocketConstants) pingPeriod() time.Duration {
	return (c.pongWait * 9) / 10
}

var wsConsts = websocketConstants{
	writeWait:      10 * time.Second,
	pongWait:       10 * time.Second,
	maxMessageSize: 512} // too small? Should be configurable at Twine level

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // come up with more appropriate size
	WriteBufferSize: 1024,
}

// Channel types:
type rxBytes struct {
	data []byte
	err  error
}
type rxMessage struct {
	msg messageMeta
	err error
}

// Websocket is a transport for Twine
type Websocket struct {
	ErrorChan     chan (error)
	conn          *websocket.Conn
	pingTicker    *time.Ticker
	rxBytesChan   chan rxBytes
	rxMessageChan chan rxMessage
	txBytesChan   chan ([]byte)
	stopWrites    chan struct{}

	closedMux sync.Mutex
	closed    bool
}

func newWebsocketServer(res http.ResponseWriter, req *http.Request) (*Websocket, error) {
	w := &Websocket{}
	w.init()

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		return nil, err
	}
	w.conn = conn

	go w.readPump()
	go w.bytesPump()
	go w.writePump()

	return w, nil
}

func newWebsocketClient(host string, path string) (*Websocket, error) {
	w := &Websocket{}
	w.init()

	u := url.URL{Scheme: "ws", Host: host, Path: path}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	w.conn = conn

	go w.readPump()
	go w.bytesPump()
	go w.writePump()

	return w, nil
}

func (w *Websocket) init() {
	w.pingTicker = time.NewTicker(wsConsts.pingPeriod())
	w.rxBytesChan = make(chan rxBytes)
	w.rxMessageChan = make(chan rxMessage)
	w.txBytesChan = make(chan []byte)
	w.ErrorChan = make(chan error)
	w.stopWrites = make(chan struct{})
	w.closed = false
}

func (w *Websocket) readPump() {
	pongWait := wsConsts.pongWait
	w.conn.SetReadLimit(wsConsts.maxMessageSize)
	w.conn.SetPongHandler(func(string) error {
		err := w.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			//w.getLogger("readPump(), SetPongHandler(), SetReadDeadline()").Error(err)
			// TODO how to handle errors?
		}
		return err
	})

	defer close(w.rxBytesChan)
	for {
		msgType, data, err := w.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// this is an error
				w.rxBytesChan <- rxBytes{[]byte{}, err}
			} else {
				// do we need to reply ?
			}
			w.Close()
			break
		}
		if msgType != websocket.BinaryMessage {
			w.rxBytesChan <- rxBytes{[]byte{}, errors.New("Message not binary")}
			break
		}
		w.rxBytesChan <- rxBytes{data, nil}
	}
}

// For now assume one twine message per ws message.
func (w *Websocket) bytesPump() {
	defer close(w.rxMessageChan)
	for msgData := range w.rxBytesChan {
		if msgData.err != nil {
			w.rxMessageChan <- rxMessage{messageMeta{}, msgData.err}
			break
		}

		decoded, remainder, err := decodeMessage(msgData.data)
		if err != nil {
			w.rxMessageChan <- rxMessage{messageMeta{}, msgData.err}
			break
		}

		if decoded.payloadSize != len(remainder) {
			w.rxMessageChan <- rxMessage{messageMeta{}, errors.New("payload and data size mismatch")}
		}

		w.rxMessageChan <- rxMessage{messageMeta{
			service:  decoded.service,
			command:  decoded.command,
			msgID:    decoded.msgID,
			refMsgID: decoded.refMsgID,
			payload:  remainder}, nil}
	}
}

func (w *Websocket) writePump() {
	defer close(w.ErrorChan)
	defer w.Close()
	for {
		select {
		case b := <-w.txBytesChan:
			err := w.conn.SetWriteDeadline(time.Now().Add(wsConsts.writeWait))
			if err != nil {
				w.ErrorChan <- fmt.Errorf("WS SetWriteDeadline error: %v", err)
				return
			}
			err = w.conn.WriteMessage(websocket.BinaryMessage, b)
			if err != nil {
				w.ErrorChan <- fmt.Errorf("WS WriteMessage error: %v", err)
				return
			}
		case <-w.pingTicker.C:
			err := w.conn.SetWriteDeadline(time.Now().Add(wsConsts.writeWait))
			if err != nil {
				w.ErrorChan <- fmt.Errorf("WS SetWriteDeadline for ping error: %v", err)
				return
			}
			err = w.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				w.ErrorChan <- fmt.Errorf("WS WriteMessage for ping error: %v", err)
				return
			}
		case <-w.stopWrites:
			return
		}
	}
}

// ReadMessage reuturs the next message or an error
// This is a problem. Because if you don't call ReadMessage you just hang
// Should ebe replaced with GetMessageChannel
func (w *Websocket) ReadMessage() (*messageMeta, error) {
	msgData, ok := <-w.rxMessageChan
	if ok {
		return &msgData.msg, msgData.err
	}
	return nil, nil
}

// WriteMessage implements twineConn's message sending
func (w *Websocket) WriteMessage(meta []byte, payload []byte) error {
	w.txBytesChan <- append(meta, payload...)
	return nil
}

// Close the websocket connection and shutdown read and write loops
func (w *Websocket) Close() {
	w.closedMux.Lock()
	defer w.closedMux.Unlock()
	if w.closed {
		return
	}
	fmt.Println("closing twine-websocket") // This should definitely not be fmt.Println
	w.pingTicker.Stop()
	if w.conn != nil {
		w.conn.Close()
	}
	close(w.stopWrites)

	w.closed = true
}
