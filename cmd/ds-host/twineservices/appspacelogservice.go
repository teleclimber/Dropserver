package twineservices

import (
	"encoding/json"
	"errors"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
)

// MigrationJobService offers subscription to appspace status by appspace id
type AppspaceLogService struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceLogger interface {
		SubscribeStatus(appspaceID domain.AppspaceID) (bool, <-chan bool)
		UnsubscribeStatus(appspaceID domain.AppspaceID, ch <-chan bool)
		SubscribeEntries(appspaceID domain.AppspaceID, n int64) (domain.LogChunk, <-chan string, error)
		UnsubscribeEntries(appspaceID domain.AppspaceID, ch <-chan string)
	} `checkinject:"required"`

	authUser domain.UserID
}

// Start creates listeners and then shuts everything down when twine exits
func (s *AppspaceLogService) Start(authUser domain.UserID, t *twine.Twine) {
	s.authUser = authUser
}

// incoming commands:
const logStatusSubscribeCmd = 11
const tailSubscribeCmd = 12

//HandleMessage handles incoming twine message
func (s *AppspaceLogService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case logStatusSubscribeCmd:
		s.handleSubscribeStatus(m)
	case tailSubscribeCmd:
		s.handleTailSubscribe(m)
	default:
		m.SendError("command not recognized")
	}
}

// Top incoming for service:
// - 11> subscribe log status
//   - <11 status
//   - 13> unsbuscribe
// - 12> get chunk and subscribe to entries
//   - <11 chunk
//   - <12 entry
//   - 13> unsubscribe

func (s *AppspaceLogService) handleSubscribeStatus(m twine.ReceivedMessageI) {
	appspace, err := s.getMessageAppspace(m)
	if err != nil {
		return
	}

	logOpen, statusCh := s.AppspaceLogger.SubscribeStatus(appspace.AppspaceID)
	go func() {
		for status := range statusCh {
			s.sendStatus(m, status)
		}
	}()
	s.sendStatus(m, logOpen)

	go func() {
		rxChan := m.GetRefRequestsChan()
		for rxM := range rxChan {
			switch rxM.CommandID() {
			case 13:
				s.AppspaceLogger.UnsubscribeStatus(appspace.AppspaceID, statusCh)
				rxM.SendOK()
				m.SendOK()
			default:
				m.SendError("command not recognized")
			}
		}
	}()
}

func (s *AppspaceLogService) sendStatus(m twine.ReceivedMessageI, status bool) {
	p := []byte("\x00")
	if status {
		p = []byte("\xff")
	}
	sent, err := m.RefSend(11, p)
	if err != nil {
		// log it?
		return
	}
	go sent.WaitReply() // we don't want this to block. This is where I'd use twine SendForget
}

func (s *AppspaceLogService) handleTailSubscribe(m twine.ReceivedMessageI) {
	appspace, err := s.getMessageAppspace(m)
	if err != nil {
		return
	}

	chunk, entriesCh, err := s.AppspaceLogger.SubscribeEntries(appspace.AppspaceID, 4*1024)
	if err != nil {
		m.SendError("got error on SubscribeEntries")
		return
	}

	go func() {
		for entry := range entriesCh {
			sent, err := m.RefSend(12, []byte(entry))
			if err != nil {
				s.getLogger("handleTailSubscribe m.RefSend Error").Error(err)
			}
			go func(snt twine.SentMessageI) {
				r, err := snt.WaitReply()
				if err != nil {
					s.getLogger("handleTailSubscribe snt.WaitReply Error").Error(err)
				}
				err = r.Error()
				if err != nil {
					s.getLogger("handleTailSubscribe r.Error Error").Error(err)
				}
			}(sent)
		}
	}()
	go func() {
		rxChan := m.GetRefRequestsChan()
		for rxM := range rxChan {
			switch rxM.CommandID() {
			case 13:
				s.AppspaceLogger.UnsubscribeEntries(appspace.AppspaceID, entriesCh)
				rxM.SendOK()
				m.SendOK()
			default:
				m.SendError("command not recognized")
			}
		}
	}()

	bytes, err := json.Marshal(chunk)
	if err != nil {
		s.getLogger("handleTailSubscribe json Marshal Error").Error(err)
		m.SendError("Failed to marhsal JSON")
		return
	}
	_, err = m.RefSendBlock(11, bytes)
	if err != nil {
		s.getLogger("handleTailSubscribe RefSendBlock Error").Error(err)
		m.SendError("internal error")
	}
}

func (s *AppspaceLogService) getMessageAppspace(m twine.ReceivedMessageI) (domain.Appspace, error) {
	var incoming IncomingSubscribeAppspace // reused from MigrationService
	err := json.Unmarshal(m.Payload(), &incoming)
	if err != nil {
		m.SendError(err.Error())
		return domain.Appspace{}, err
	}

	appspace, err := s.AppspaceModel.GetFromID(incoming.AppspaceID)
	if err != nil {
		m.SendError(err.Error())
		return domain.Appspace{}, err
	}
	if appspace.OwnerID != s.authUser {
		m.SendError("forbidden")
		return domain.Appspace{}, errors.New("forbidder")
	}
	return *appspace, nil
}

func (s *AppspaceLogService) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("AppspaceLogService")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
