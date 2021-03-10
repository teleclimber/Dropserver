package twine

import (
	"errors"
	"fmt"
	"sync"
)

// msg is the data that is stashed in messages hash
type msg struct {
	service          serviceID
	closed           bool
	replyChanCreated bool
	replyChan        chan ReceivedReplyI
	unregWait        *sync.WaitGroup

	refRequestMux   sync.Mutex
	refChanOpen     bool
	refRequestsChan chan ReceivedMessageI
}

type messageRegistry struct {
	firstMsgID  uint8
	lastMsgID   uint8
	messagesMux sync.Mutex
	messages    map[uint8]*msg
	nextID      uint8
}

// wondering if we could move message stashing and ids to a separate struct
// The only catch is t.graceful, whihc is muxed my messagesMux,
// ..and unregisterMessage calls t.close when graceful and zero messages.
// -> might just be a matter of getting back to the waitGroup idea, which wasn't bad.
func (r *messageRegistry) incrementNextID() {
	r.nextID++
	if r.nextID > r.lastMsgID {
		r.nextID = r.firstMsgID
	}
}
func (r *messageRegistry) checkMsgIDRange(msgID int) (uint8, error) {
	if msgID == 0 || msgID > 0xff {
		return 0, fmt.Errorf("message ID out of bounds: %v", msgID)
	}
	return uint8(msgID), nil
}
func (r *messageRegistry) msgIDIsLocal(msgID8 uint8) bool {
	return msgID8 >= r.firstMsgID && msgID8 <= r.lastMsgID
}
func (r *messageRegistry) checkMsgIDRemote(msgID int) (uint8, error) {
	msgID8, err := r.checkMsgIDRange(msgID)
	if err != nil {
		return 0, err
	}
	if r.msgIDIsLocal(msgID8) {
		return 0, fmt.Errorf("message ID in wrong range: %v", msgID)
	}
	return msgID8, nil
}

// newMessage creates a message ID for an outgoing message
// ..as in a message initiated here, as opposed to a reply
func (r *messageRegistry) newMessage(service serviceID) (int, *msg, error) {
	r.messagesMux.Lock()
	defer r.messagesMux.Unlock()

	_, ok := r.messages[r.nextID]
	for ok { // should only loop around once. Since we lock, ther eis no way this changes.
		r.incrementNextID()
		_, ok = r.messages[r.nextID]
	}

	newID := r.nextID
	newMsg := msg{
		service:   service,
		replyChan: make(chan ReceivedReplyI),
		closed:    false}

	r.messages[newID] = &newMsg

	r.incrementNextID()

	return int(newID), &newMsg, nil
}
func (r *messageRegistry) registerMessage(raw *messageMeta) (*msg, error) { // return the msg so caller can connect replyChan n ,.
	r.messagesMux.Lock()
	defer r.messagesMux.Unlock()

	if r.msgIDIsLocal(raw.msgID) {
		return nil, fmt.Errorf("message ID invalid: %v", raw.msgID)
	}

	msgID8 := uint8(raw.msgID)
	_, ok := r.messages[msgID8]
	if ok {
		return nil, fmt.Errorf("message ID is already open: %v", msgID8)
	}

	newMsg := &msg{
		service:   raw.service,
		replyChan: make(chan ReceivedReplyI),
		closed:    false}
	r.messages[msgID8] = newMsg

	return newMsg, nil
}
func (r *messageRegistry) closeMessage(msgID uint8) (*msg, error) {
	r.messagesMux.Lock()
	defer r.messagesMux.Unlock()

	msgData, ok := r.messages[msgID]
	if !ok {
		return nil, fmt.Errorf("message ID not found: %v", msgID)
	}
	if msgData.closed {
		return nil, errors.New("message was already closed")
	}
	msgData.closed = true

	msgData.refRequestMux.Lock()
	if msgData.refChanOpen {
		close(msgData.refRequestsChan)
		msgData.refChanOpen = false
	}
	msgData.refRequestMux.Unlock()

	return msgData, nil
}
func (r *messageRegistry) unregisterMessage(msgID uint8) error {
	r.messagesMux.Lock()
	defer r.messagesMux.Unlock()

	msgData, ok := r.messages[msgID]
	if !ok {
		return fmt.Errorf("message ID is not registered: %v", msgID)
	}

	msgData.closed = true

	msgData.refRequestMux.Lock()
	if msgData.refChanOpen {
		close(msgData.refRequestsChan)
		msgData.refChanOpen = false
	}
	msgData.refRequestMux.Unlock()

	delete(r.messages, msgID)

	if msgData.unregWait != nil {
		msgData.unregWait.Done()
	}

	return nil
}
func (r *messageRegistry) getOpenMessage(msgID uint8) (*msg, error) {
	r.messagesMux.Lock()
	defer r.messagesMux.Unlock()

	msgData, ok := r.messages[msgID]
	if !ok {
		return nil, fmt.Errorf("message ID not found: %v", msgID)
	}

	if msgData.closed {
		return nil, fmt.Errorf("message ID is closed: %v", msgID)
	}

	return msgData, nil
}
func (r *messageRegistry) getMessageData(msgID uint8) (*msg, error) {
	r.messagesMux.Lock()
	defer r.messagesMux.Unlock()

	msgData, ok := r.messages[msgID]
	if !ok {
		return nil, fmt.Errorf("message ID not found: %v", msgID)
	}

	return msgData, nil
}

func (r *messageRegistry) waitAllUnregistered() {
	var wg sync.WaitGroup
	r.messagesMux.Lock()
	wg.Add(len(r.messages))
	for _, msgData := range r.messages {
		msgData.unregWait = &wg
	}
	r.messagesMux.Unlock()

	wg.Wait()
}
