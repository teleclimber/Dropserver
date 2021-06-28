package main

import (
	"encoding/json"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

type AppMetaService struct {
	AppFilesModel interface {
		ReadMeta(locationKey string) (*domain.AppFilesMetadata, error)
	} `checkinject:"required"`
	AppVersionEvents interface {
		Subscribe(chan<- domain.AppID)
		Unsubscribe(chan<- domain.AppID)
	} `checkinject:"required"`
}

func (s *AppMetaService) HandleMessage(m twine.ReceivedMessageI) {

}

func (s *AppMetaService) Start(t *twine.Twine) {
	appVersionEvent := make(chan domain.AppID)
	s.AppVersionEvents.Subscribe(appVersionEvent)
	go func() {
		for range appVersionEvent {
			go s.sendAppData(t)
		}
	}()
	// push initial app data:
	s.sendAppData(t)

	t.WaitClose()

	s.AppVersionEvents.Unsubscribe(appVersionEvent)
	close(appVersionEvent)
}

const appDataCmd = 12

func (s *AppMetaService) sendAppData(twine *twine.Twine) {
	appFilesMeta, err := s.AppFilesModel.ReadMeta("")
	if err != nil {
		panic(err)
	}
	bytes, err := json.Marshal(appFilesMeta)
	if err != nil {
		fmt.Println("sendAppData json Marshal Error: " + err.Error())
	}

	_, err = twine.SendBlock(baseDataService, appDataCmd, bytes)
	if err != nil {
		fmt.Println("sendAppData SendBlock Error: " + err.Error())
	}
}
