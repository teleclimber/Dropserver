package twine

import "errors"

// MessageI is the baseinterface for all Twine message interfaces
type MessageI interface {
	MsgID() int
	RefMsgID() int
	CommandID() int
	ServiceID() int
	Payload() *[]byte
}

// MessageGetReplyI adds the ability to wait for the reply to a sent message
type MessageGetReplyI interface {
	WaitReply() (ReceivedReplyI, error)
}

// MessageReplyOKErrI adds the ability to reply with OK and (someday) Error
// This can be a reply to an incoming messgae or to a reply.
type MessageReplyOKErrI interface {
	SendOK() error
	// ReplyError
}

// MessageReplierI adds teh ability to reply to an incoming message
type MessageReplierI interface {
	Reply(int, *[]byte) error
}

// MessageReceivedOKI adds getters for OK and Error replies
type MessageReceivedOKI interface {
	OK() bool
	Error() error
}

// MessageRefererI adds the abilty to send and receive new Requests
// that reference the message
type MessageRefererI interface {
	RefSend(int, *[]byte) (SentMessageI, error)
	RefSendBlock(int, *[]byte) (ReceivedReplyI, error)
	GetRefRequestsChan() chan ReceivedMessageI
}

// SentMessageI is a mesage that was just sent and has the ability to
// makeadditional requests refering this message, and wait for replies
type SentMessageI interface {
	MessageI
	MessageGetReplyI
	MessageRefererI
}

// ReceivedMessageI is a message that was initiated on the other side and received here
// You can reply or create new requests referring the message
type ReceivedMessageI interface {
	MessageI
	MessageReplierI
	MessageReplyOKErrI
	MessageRefererI
}

// ReceivedReplyI is a message received in response to a message sent
// You can acknowledge with an OK or error.
type ReceivedReplyI interface {
	MessageI
	MessageReceivedOKI
	MessageReplyOKErrI
	// helpers to get errors and oks?
}

// Message encapsulates a generic message interface
type Message struct {
	msgID    int
	refMsgID int
	service  int
	command  int
	payload  *[]byte
	msg      *msg
	t        *Twine
}

// MsgID returns the message id
func (m *Message) MsgID() int {
	return m.msgID
}

// RefMsgID returns the reference message id
func (m *Message) RefMsgID() int {
	return m.refMsgID
}

// CommandID returns the command id
func (m *Message) CommandID() int {
	return m.command
}

// ServiceID returns the service id
func (m *Message) ServiceID() int {
	return m.service
}

// Payload returns a pointer to the payload
func (m *Message) Payload() *[]byte {
	return m.payload
}

// WaitReply returns the reply or error when it eventually arrives
func (m *Message) WaitReply() (ReceivedReplyI, error) {
	r, ok := <-m.msg.replyChan
	if !ok {
		return nil, errors.New("No reply received, channel closed")
	}
	return r, nil
}

// SendOK sends an OK and closes the message
func (m *Message) SendOK() error {
	err := m.t.ReplyOKClose(m.msgID)
	if err != nil {
		return err
	}
	return nil
}

// SendError sends the error code along with description
func (m *Message) SendError(errStr string) error {
	// TODO: implement
	return nil
}

// Reply to message
// This blocks until the other side returns OK or error
func (m *Message) Reply(cmd int, payload *[]byte) error {
	err := m.t.Reply(m.msgID, cmd, payload)
	if err != nil {
		return err
	}
	return nil
}

// OK checks if the reply was a simple OK
func (m *Message) OK() bool {
	return m.command == int(protocolOK)
}

// Error returns an error if the reply was an error
func (m *Message) Error() error {
	if m.command == int(protocolError) {
		if m.payload != nil {
			return errors.New(string(*m.payload))
		}
		return errors.New("No error description given")
	}
	return nil
}

// RefSend creates a new message with a reference to the current message
func (m *Message) RefSend(cmd int, payload *[]byte) (SentMessageI, error) {
	sent, err := m.t.RefRequest(m.msgID, cmd, payload)
	if err != nil {
		return nil, err
	}
	return sent, nil
}

// RefSendBlock sends a new mssage referencing anexisting one,
// and returns with the response or an error
func (m *Message) RefSendBlock(cmd int, payload *[]byte) (ReceivedReplyI, error) {
	sent, err := m.t.RefRequest(m.msgID, cmd, payload)
	if err != nil {
		return nil, err
	}

	reply, err := sent.WaitReply()
	if err != nil {
		return nil, err
	}

	return reply, nil
}

// GetRefRequestsChan creates a channel if necessary and returns it
func (m *Message) GetRefRequestsChan() chan ReceivedMessageI {
	m.msg.refRequestMux.Lock()
	defer m.msg.refRequestMux.Unlock()
	if !m.msg.refChanOpen {
		m.msg.refRequestsChan = make(chan ReceivedMessageI)
		m.msg.refChanOpen = true
	}
	return m.msg.refRequestsChan
}
