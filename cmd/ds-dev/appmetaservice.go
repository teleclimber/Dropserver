package main

import (
	"encoding/json"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/twine-go/twine"
)

type AppMetaService struct {
	DevAppModel   *DevAppModel `checkinject:"required"`
	AppFilesModel interface {
		ReadEvaluatedManifest(locationKey string) (domain.AppVersionManifest, error)
	} `checkinject:"required"`
	AppGetter interface {
		ValidateMigrationSteps(migrations []domain.MigrationStep) ([]int, error)
	} `checkinject:"required"`
	DevAppProcessEvents interface {
		Subscribe() (AppProcessEvent, <-chan AppProcessEvent)
		Unsubscribe(<-chan AppProcessEvent)
	} `checkinject:"required"`
	AppVersionEvents interface {
		Subscribe(chan<- string)
		Unsubscribe(chan<- string)
	} `checkinject:"required"`
}

func (s *AppMetaService) HandleMessage(m twine.ReceivedMessageI) {

}

func (s *AppMetaService) Start(t *twine.Twine) {
	appVersionEvent := make(chan string)
	s.AppVersionEvents.Subscribe(appVersionEvent)
	go func() {
		for range appVersionEvent {
			go s.sendManifest(t)
		}
	}()
	s.sendManifest(t)

	ev, appProcessCh := s.DevAppProcessEvents.Subscribe()
	go func() {
		for appProcessEvent := range appProcessCh {
			s.sendAppGetEvent(t, appProcessEvent)
		}
	}()
	go s.sendAppGetEvent(t, ev)

	t.WaitClose()

	s.DevAppProcessEvents.Unsubscribe(appProcessCh)
	s.AppVersionEvents.Unsubscribe(appVersionEvent)
}

const appDataCmd = 12
const appProcessEventCmd = 13

type AppMetaResp struct {
	AppName       string                 `json:"name"`
	AppVersion    domain.Version         `json:"version"`
	SchemaVersion int                    `json:"schema"`
	APIVersion    domain.APIVersion      `json:"api_version"`
	Migrations    []domain.MigrationStep `json:"migration_steps"`
	Schemas       []int                  `json:"schemas"`
}

func (s *AppMetaService) sendManifest(twine *twine.Twine) {
	manifest, err := s.AppFilesModel.ReadEvaluatedManifest("")
	if err != nil {
		fmt.Println("sendManifest ReadEvaluatedManifest Error: ", err)
	}
	bytes, err := json.Marshal(manifest) // kind of a bummer we're parsing the json manifest and converting right back to json
	if err != nil {
		fmt.Println("sendManifest json Marshal Error: " + err.Error())
	}
	_, err = twine.SendBlock(appDataService, appDataCmd, bytes)
	if err != nil {
		fmt.Println("sendManifest SendBlock Error: " + err.Error())
	}
}

func (s *AppMetaService) sendAppGetEvent(twine *twine.Twine, ev AppProcessEvent) {
	bytes, err := json.Marshal(ev)
	if err != nil {
		fmt.Println("sendAppGetEvent json Marshal Error: " + err.Error())
	}
	_, err = twine.SendBlock(appDataService, appProcessEventCmd, bytes)
	if err != nil {
		fmt.Println("sendAppGetEvent SendBlock Error: " + err.Error())
	}
}
