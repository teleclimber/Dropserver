package main

import (
	"encoding/json"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/twine-go/twine"
)

// MigrationJobService offers subscription to appspace status by appspace id
type MigrationJobService struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	MigrationJobModel interface {
		GetRunning() ([]domain.MigrationJob, error)
	} `checkinject:"required"`
	MigrationJobEvents interface {
		Subscribe() <-chan domain.MigrationJob
		Unsubscribe(<-chan domain.MigrationJob)
	} `checkinject:"required"`
}

// Start creates listeners and then shuts everything down when twine exits
func (s *MigrationJobService) Start(authUser domain.UserID, t *twine.Twine) domain.TwineServiceI {
	mjs := migrationJobServiceInternal{
		MigrationJobService: s,
		authUser:            authUser,
		twine:               t,
	}
	return &mjs
}

type migrationJobServiceInternal struct {
	*MigrationJobService

	authUser domain.UserID
	twine    *twine.Twine
}

const subscribeMigration = 11
const subscribeAppspaceMigration = 12
const unsubscribeMigration = 13

// HandleMessage handles incoming twine message
func (s *migrationJobServiceInternal) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case subscribeAppspaceMigration:
		go s.handleSubscribeAppspace(m)
	default:
		m.SendError("command not recognized")
	}
}

// IncomingSubscribeAppspace is json encoded payload to subscribe to appspace status
type IncomingSubscribeAppspace struct {
	AppspaceID domain.AppspaceID `json:"appspace_id"`
}

func (s *migrationJobServiceInternal) handleSubscribeAppspace(m twine.ReceivedMessageI) {
	var incoming IncomingSubscribeAppspace
	err := json.Unmarshal(m.Payload(), &incoming)
	if err != nil {
		m.SendError(err.Error())
		return
	}

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
	migrationJobChan := s.MigrationJobEvents.Subscribe()
	go func() {
		for statusEvent := range migrationJobChan {
			go s.sendMigrationJob(m, statusEvent)
		}
	}()

	// then get current data and send the data down as initial/current status
	jobs, err := s.MigrationJobModel.GetRunning() // this should really come from job model
	if err != nil {
		m.SendError(err.Error())
		return
	}
	for _, j := range jobs {
		if j.AppspaceID == incoming.AppspaceID {
			go s.sendMigrationJob(m, j)
			// TODO if no job found, that sould be sent in some manner as well
		}
	}

	//then listen for shutdown request.
	go func() {
		rxChan := m.GetRefRequestsChan()
		for rxM := range rxChan {
			switch rxM.CommandID() {
			case unsubscribeMigration:
				s.MigrationJobEvents.Unsubscribe(migrationJobChan)
				rxM.SendOK()
				m.SendOK()
			default:
				m.SendError("command not recognized")
			}
		}
	}()

	s.twine.WaitClose()

	s.MigrationJobEvents.Unsubscribe(migrationJobChan)
}

// see appspacestatustwine which uses a same pattern wrt Twine
func (s *migrationJobServiceInternal) sendMigrationJob(m twine.ReceivedMessageI, migrationJob domain.MigrationJob) {
	bytes, err := json.Marshal(migrationJob)
	if err != nil {
		s.getLogger("sendMigrationJob json Marshal Error").Error(err)
		m.SendError("Failed to marhsal JSON")
		return
	}

	_, err = m.RefSendBlock(11, bytes)
	if err != nil {
		s.getLogger("sendMigrationJob SendBlock Error").Error(err)
		m.SendError("internal error")
	}
}

func (s *MigrationJobService) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("MigrationJobService")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
