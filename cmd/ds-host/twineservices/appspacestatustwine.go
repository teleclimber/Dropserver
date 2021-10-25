package twineservices

import (
	"encoding/json"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
)

// AppspaceStatusService offers subscription to appspace status by appspace id
type AppspaceStatusService struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceStatus interface {
		Track(domain.AppspaceID) domain.AppspaceStatusEvent
	} `checkinject:"required"`
	AppspaceStatusEvents interface {
		Subscribe(domain.AppspaceID, chan<- domain.AppspaceStatusEvent)
		Unsubscribe(domain.AppspaceID, chan<- domain.AppspaceStatusEvent)
	} `checkinject:"required"`

	authUser domain.UserID
}

// Start creates listeners and then shuts everything down when twine exits
func (s *AppspaceStatusService) Start(authUser domain.UserID, t *twine.Twine) {
	// does nothing.
	// I think all messages are supposed to auto-close on twine close, so there should be no residual activity
	s.authUser = authUser
}

const subscribeStatus = 11
const unsubscribeStatus = 13

// HandleMessage handles incoming twine message
func (s *AppspaceStatusService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case subscribeStatus:
		s.handleSubscribeMessage(m)
	default:
		m.SendError("command not recognized")
	}
}

// IncomingAppspaceID is json encoded payload to subscribe to appspace status
type IncomingAppspaceID struct {
	AppspaceID domain.AppspaceID `json:"appspace_id"`
}

func (s *AppspaceStatusService) handleSubscribeMessage(m twine.ReceivedMessageI) {
	var incoming IncomingAppspaceID
	err := json.Unmarshal(m.Payload(), &incoming)
	if err != nil {
		m.SendError(err.Error())
		return
	}

	// TODO first need to verify the appsace is owned by the authenticated user
	// do we just need to load it from appspace model?
	appspace, err := s.AppspaceModel.GetFromID(incoming.AppspaceID)
	if err != nil {
		m.SendError(err.Error())
		return
	}
	if appspace.OwnerID != s.authUser {
		m.SendError("forbidden")
		return
	}

	// First subscribe
	appspaceStatusChan := make(chan domain.AppspaceStatusEvent)
	s.AppspaceStatusEvents.Subscribe(incoming.AppspaceID, appspaceStatusChan)
	go func() {
		for statusEvent := range appspaceStatusChan {
			go s.sendAppspaceStatusEvent(m, statusEvent)
		}
	}()

	// then get current data (causing AppspaceStatus to track the appsapce)
	// .. and send the data down as initial/current status
	go s.sendAppspaceStatusEvent(m, s.AppspaceStatus.Track(incoming.AppspaceID))

	//then listen for shutdown request.
	go func() {
		rxChan := m.GetRefRequestsChan()
		for rxM := range rxChan {
			switch rxM.CommandID() {
			case unsubscribeStatus:
				s.AppspaceStatusEvents.Unsubscribe(incoming.AppspaceID, appspaceStatusChan)
				close(appspaceStatusChan)
				rxM.SendOK()
				m.SendOK()
			default:
				m.SendError("command not recognized")
			}
		}
	}()

}

const statusEventCmd = 11

// TODO maybe sendAppspaceStatusEvent should return an error so that we can unsubscribe and stop the process if there is a problem
func (s *AppspaceStatusService) sendAppspaceStatusEvent(m twine.ReceivedMessageI, statusEvent domain.AppspaceStatusEvent) {
	bytes, err := json.Marshal(statusEvent)
	if err != nil {
		s.getLogger("sendAppspaceStatusEvent json Marshal Error").Error(err)
		m.SendError("Failed to unmarhsal JSON")
		// Do we really send an error on parent message?
		// Maybe return error and let caller deal with it (and unsubscribe)
		return
	}

	_, err = m.RefSendBlock(statusEventCmd, bytes)
	if err != nil {
		s.getLogger("sendAppspaceStatusEvent SendBlock Error").Error(err)
		m.SendError("internal error")
	}
}

func (s *AppspaceStatusService) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("AppspaceStatusService")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
