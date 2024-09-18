package twineservices

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
)

// MigrationJobService offers subscription to app or appspace logs
type AppspaceLogService struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppModel interface {
		GetFromID(appID domain.AppID) (*domain.App, error)
		GetVersion(appID domain.AppID, version domain.Version) (domain.AppVersion, error)
	} `checkinject:"required"`
	AppspaceLogger interface {
		Get(appspaceID domain.AppspaceID) domain.LoggerI
	} `checkinject:"required"`
	AppLogger interface {
		Get(string) domain.LoggerI
	} `checkinject:"required"`
}

// Start creates listeners and then shuts everything down when twine exits
func (s *AppspaceLogService) Start(authUser domain.UserID, t *twine.Twine) domain.TwineServiceI {
	asl := &appspaceLogService{
		AppspaceLogService: s,
		twine:              t,
		authUser:           authUser}
	return asl
}

type appspaceLogService struct {
	*AppspaceLogService

	twine    *twine.Twine
	authUser domain.UserID
}

// incoming commands:
const subscribeAppspaceLogCmd = 11
const subscribeAppLogCmd = 12

// outgoing commands, ref to 11:
const statusSubCmd = 11
const chunkSubCmd = 12
const entrysubCmd = 13

// incoming sub commands to 11
const unsubscribeLogCmd = 13

// HandleMessage handles incoming twine message
func (s *appspaceLogService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case subscribeAppspaceLogCmd:
		s.handleSubscribeAppspace(m)
	case subscribeAppLogCmd:
		s.handleSubscribeApp(m)
	default:
		m.SendError("command not recognized")
	}
}

// After status and entries. Proposed:
// Top incoming for service:
// - 11> subscribe log
//   - <11 status
//   - <12 chunk
//   - <13 entry
//   - 12> [get chunk from/to]
//     [reply under this message to keep separate from initial chunk]
//   - 13> unsubscribe

// To improve consistency move entries below chunk.
// Guarantees that an entry received is relative to the initial chunk sent.
// - 11> subscribe log
//   - <11 status
//   - <12 chunk
//      - <13 entry
//   - 12> [get chunk from/to]
//     [reply under this message to keep separate from initial chunk]
//   - 13> unsubscribe

func (s *appspaceLogService) handleSubscribeAppspace(m twine.ReceivedMessageI) {
	appspace, err := s.getMessageAppspace(m)
	if err != nil {
		return
	}

	logger := s.AppspaceLogger.Get(appspace.AppspaceID)

	ls := logService{
		twine:  s.twine,
		m:      m,
		logger: logger}
	ls.start()
}

func (s *appspaceLogService) handleSubscribeApp(m twine.ReceivedMessageI) {
	appVersion, err := s.getMessageApp(m)
	if err != nil {
		return
	}

	logger := s.AppLogger.Get(appVersion.LocationKey)

	ls := logService{
		twine:  s.twine,
		m:      m,
		logger: logger}
	ls.start()
}

// IncomingSubscribeAppspace is json encoded payload to subscribe to appspace status
type IncomingSubscribeAppspace struct {
	AppspaceID domain.AppspaceID `json:"appspace_id"`
}

func (s *appspaceLogService) getMessageAppspace(m twine.ReceivedMessageI) (domain.Appspace, error) {
	var incoming IncomingSubscribeAppspace
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

type IncomingSubscribeApp struct {
	AppID   domain.AppID   `json:"app_id"`
	Version domain.Version `json:"version"`
}

func (s *appspaceLogService) getMessageApp(m twine.ReceivedMessageI) (domain.AppVersion, error) {
	var incoming IncomingSubscribeApp
	err := json.Unmarshal(m.Payload(), &incoming)
	if err != nil {
		m.SendError(err.Error())
		return domain.AppVersion{}, err
	}

	app, err := s.AppModel.GetFromID(incoming.AppID)
	if err != nil {
		m.SendError(err.Error())
		return domain.AppVersion{}, err
	}
	if app.OwnerID != s.authUser {
		m.SendError("forbidden")
		return domain.AppVersion{}, errors.New("forbidder")
	}

	appVersion, err := s.AppModel.GetVersion(incoming.AppID, incoming.Version)
	if err != nil {
		m.SendError(err.Error())
		return domain.AppVersion{}, err
	}

	return appVersion, nil
}

type logService struct {
	twine  *twine.Twine
	m      twine.ReceivedMessageI
	logger domain.LoggerI

	entriesMux sync.Mutex
	entriesCh  <-chan string
}

func (s *logService) start() {
	logOpen, statusCh := s.logger.SubscribeStatus()
	go func() {
		for status := range statusCh { //goroutine stuck here after tab close
			// HERE if status is open, then resend initial chunk.
			if status {
				s.sendInitialChunk()
			}
			s.sendStatus(status)
		}
	}()
	s.sendStatus(logOpen)

	s.sendInitialChunk()

	go func() {
		rxChan := s.m.GetRefRequestsChan()
		for rxM := range rxChan { //goroutine stuck here after tab close
			switch rxM.CommandID() {
			case unsubscribeLogCmd:
				s.logger.UnsubscribeStatus(statusCh)
				s.handleUnsubscribe(rxM)

			default:
				rxM.SendError("command not recognized")
			}
		}
	}()

	go func() {
		s.twine.WaitClose()
		s.logger.UnsubscribeStatus(statusCh)
	}()
}
func (s *logService) handleUnsubscribe(rxM twine.ReceivedMessageI) {
	s.entriesMux.Lock()
	defer s.entriesMux.Unlock()
	if s.entriesCh != nil {
		s.logger.UnsubscribeEntries(s.entriesCh)
		s.entriesCh = nil
	}

	rxM.SendOK()
	s.m.SendOK()
}

func (s *logService) sendStatus(status bool) {
	p := []byte("\x00")
	if status {
		p = []byte("\xff")
	}
	sent, err := s.m.RefSend(statusSubCmd, p)
	if err != nil {
		s.getLogger("sendStatus RefSendBlock Error").Error(err)
		s.m.SendError("internal error")
		return
	}
	go sent.WaitReply() // we don't want this to block. This is where I'd use twine SendForget
}

func (s *logService) sendInitialChunk() {
	// unsubscribe if there is a entries chan
	s.entriesMux.Lock()
	defer s.entriesMux.Unlock()

	if s.entriesCh != nil {
		s.logger.UnsubscribeEntries(s.entriesCh)
		s.entriesCh = nil
	}

	chunk, entriesCh, err := s.logger.SubscribeEntries(16 * 1024)
	if err != nil {
		// maybe log is closed, but if we send error, this closes the message, so quiet.
		return
	}
	s.entriesCh = entriesCh

	bytes, err := json.Marshal(chunk)
	if err != nil {
		s.getLogger("sendInitialChunk json Marshal Error").Error(err)
		s.m.SendError("Failed to marhsal JSON")
		return
	}
	_, err = s.m.RefSendBlock(chunkSubCmd, bytes)
	if err != nil {
		s.getLogger("sendInitialChunk RefSendBlock Error").Error(err) // Error here: msg ID not found
		s.m.SendError("internal error")
	}

	go func() {
		for entry := range entriesCh { //goroutine stuck here after tab close
			s.sendEntry(entry)
		}
	}()

	go func() {
		s.twine.WaitClose()
		s.entriesMux.Lock()
		defer s.entriesMux.Unlock()
		if s.entriesCh != nil {
			s.logger.UnsubscribeEntries(s.entriesCh)
			s.entriesCh = nil
		}
	}()
}

func (s *logService) sendEntry(entry string) {
	sent, err := s.m.RefSend(entrysubCmd, []byte(entry))
	if err != nil {
		s.getLogger("sendEntry m.RefSend Error").Error(err)
		return
	}
	go func(snt twine.SentMessageI) {
		r, err := snt.WaitReply()
		if err != nil {
			s.getLogger("sendEntry snt.WaitReply Error").Error(err)
		}
		err = r.Error()
		if err != nil {
			s.getLogger("sendEntry r.Error Error").Error(err)
		}
	}(sent)
}

func (s *logService) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("logService")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
