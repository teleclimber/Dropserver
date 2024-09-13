package main

import (
	"encoding/json"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/twine-go/twine"
)

type AppspaceStatusService struct {
	AppspaceStatus interface {
		Get(appspaceID domain.AppspaceID) domain.AppspaceStatusEvent
	} `checkinject:"required"`
	AppspaceStatusEvents interface {
		Subscribe() <-chan domain.AppspaceStatusEvent
		Unsubscribe(<-chan domain.AppspaceStatusEvent)
	} `checkinject:"required"`
}

func (s *AppspaceStatusService) HandleMessage(m twine.ReceivedMessageI) {

}

func (s *AppspaceStatusService) Start(t *twine.Twine) {
	appspaceStatusChan := s.AppspaceStatusEvents.Subscribe()
	go func() {
		for statusEvent := range appspaceStatusChan {
			go s.sendStatusEvent(t, statusEvent)
		}
	}()

	go s.sendStatusEvent(t, s.AppspaceStatus.Get(appspaceID))

	t.WaitClose()

	s.AppspaceStatusEvents.Unsubscribe(appspaceStatusChan)
}

const statusEventCmd = 11

func (s *AppspaceStatusService) sendStatusEvent(twine *twine.Twine, statusEvent domain.AppspaceStatusEvent) {
	bytes, err := json.Marshal(statusEvent)
	if err != nil {
		fmt.Println("sendAppspaceStatusEvent json Marshal Error: " + err.Error())
	}

	_, err = twine.SendBlock(appspaceStatusService, statusEventCmd, bytes)
	if err != nil {
		fmt.Println("sendAppspaceStatusEvent SendBlock Error: " + err.Error())
	}
}
