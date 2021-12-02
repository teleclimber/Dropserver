package twineservices

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
)

type AppGetterService struct {
	AppGetter interface {
		GetUser(key domain.AppGetKey) (domain.UserID, bool)
		SubscribeKey(key domain.AppGetKey) (domain.AppGetEvent, <-chan domain.AppGetEvent)
		Unsubscribe(ch <-chan domain.AppGetEvent)
	} `checkinject:"required"`

	authUser domain.UserID
}

// Start
func (s *AppGetterService) Start(authUser domain.UserID, t *twine.Twine) {
	// does nothing.
	// I think all messages are supposed to auto-close on twine close, so there should be no residual activity
	s.authUser = authUser
}

const subscribeAppGetKey = 11

// HandleMessage handles incoming twine message
func (s *AppGetterService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case subscribeAppGetKey:
		s.handleSubscribeMessage(m)
	default:
		m.SendError("command not recognized")
	}
}

func (s *AppGetterService) handleSubscribeMessage(m twine.ReceivedMessageI) {
	key := domain.AppGetKey(m.Payload())

	userID, ok := s.AppGetter.GetUser(key)
	if !ok {
		m.SendError("key not found")
		return
	}
	if userID != s.authUser {
		m.SendError("unauthorized")
		return
	}

	serv := appGetEventService{
		appGetter: s.AppGetter,
		m:         m,
		key:       key}
	serv.start()
}

func (s *AppGetterService) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("AppGetterService")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

type appGetEventService struct {
	appGetter interface {
		GetUser(key domain.AppGetKey) (domain.UserID, bool)
		SubscribeKey(key domain.AppGetKey) (domain.AppGetEvent, <-chan domain.AppGetEvent)
		Unsubscribe(ch <-chan domain.AppGetEvent)
	}
	m   twine.ReceivedMessageI
	key domain.AppGetKey
	ch  <-chan domain.AppGetEvent

	errMux sync.Mutex
	err    error
}

func (s *appGetEventService) start() {

	lastEvent, ch := s.appGetter.SubscribeKey(s.key)
	if ch == nil && lastEvent.Done {
		err := s.sendEvent(lastEvent)
		if err != nil {
			s.m.SendError(fmt.Sprintf("closed due to error sending: %v", err))
		} else {
			s.m.SendOK()
		}
		return
	} else if ch == nil {
		s.m.SendError("No event found for this key")
		return
	}

	s.ch = ch

	err := s.sendEvent(lastEvent)
	if err != nil {
		s.appGetter.Unsubscribe(ch)
		s.m.SendError(fmt.Sprintf("closed due to error sending: %v", err))
		return
	}

	go func() {
		for e := range ch {
			s.handleError(s.sendEvent(e))
		}
		// loop exits when channel closes.
		// should we also set a flag on s to indicate the subscription is over?
		// Reason is AppGetter may unilaterally close this (for ex if app get key expires and is deleted.)
		// so if unsubscribed from upstream, we need .. to do what? we're closing the ref message anyways.
		s.errMux.Lock()
		defer s.errMux.Unlock()
		if s.err == nil {
			s.m.SendOK()
		}
	}()

	//then listen for shutdown request.
	go func() {
		rxChan := s.m.GetRefRequestsChan()
		for rxM := range rxChan {
			switch rxM.CommandID() {
			case 13:
				s.appGetter.Unsubscribe(ch)
				rxM.SendOK()
			default:
				rxM.SendError("command not recognized")
			}
		}
	}()
}

func (s *appGetEventService) sendEvent(event domain.AppGetEvent) error {
	s.errMux.Lock()
	defer s.errMux.Unlock()
	if s.err != nil {
		return nil
	}
	bytes, err := json.Marshal(event)
	if err != nil {
		s.getLogger("sendEvent json Marshal Error").Error(err)
		//m.SendError("Failed to unmarhsal JSON")
		// Do we really send an error on parent message?
		// Maybe return error and let caller deal with it (and unsubscribe)
		return err
	}

	sent, err := s.m.RefSend(11, bytes) // here we should send but not block.
	if err != nil {
		s.getLogger("sendAppspaceStatusEvent SendBlock Error").Error(err)
		return err
	}

	go sent.WaitReply() // we don't want to block

	return nil
}

func (s *appGetEventService) handleError(err error) {
	if err == nil {
		return
	}

	s.errMux.Lock()
	defer s.errMux.Unlock()
	s.err = err

	if s.ch != nil {
		s.appGetter.Unsubscribe(s.ch) // unsubscribe closes channel
		s.ch = nil
	}

	// close things down ?
	go s.m.SendError(fmt.Sprintf("closed due to error sending: %v", err))
}

func (s *appGetEventService) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("appGetEventService")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
