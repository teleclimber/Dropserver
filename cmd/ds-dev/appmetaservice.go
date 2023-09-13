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
		GetVersionChangelog(locationKey string, version domain.Version) (string, bool, error)
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
			go s.sendChangelog(t)
		}
	}()
	s.sendManifest(t)
	s.sendChangelog(t)

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
const appChangelogCmd = 14

type AppMetaResp struct {
	AppName       string                 `json:"name"`
	AppVersion    domain.Version         `json:"version"`
	SchemaVersion int                    `json:"schema"`
	APIVersion    domain.APIVersion      `json:"api_version"`
	Migrations    []domain.MigrationStep `json:"migration_steps"`
	Schemas       []int                  `json:"schemas"`
}

func (s *AppMetaService) sendManifest(twine *twine.Twine) {
	bytes, err := json.Marshal(s.DevAppModel.Manifest)
	if err != nil {
		fmt.Println("sendManifest json Marshal Error: " + err.Error())
	}
	_, err = twine.SendBlock(appDataService, appDataCmd, bytes)
	if err != nil {
		fmt.Println("sendManifest SendBlock Error: " + err.Error())
	}
}

func (s *AppMetaService) sendChangelog(twine *twine.Twine) {
	cl, ok, err := s.AppFilesModel.GetVersionChangelog("", s.DevAppModel.Manifest.Version)
	if !ok {
		cl = "Error reading changelog"
	} else if err != nil {
		fmt.Println("sendChangelog GetVersionChangelog Error: " + err.Error())
	}
	_, err = twine.SendBlock(appDataService, appChangelogCmd, []byte(cl))
	if err != nil {
		fmt.Println("sendChangelog SendBlock Error: " + err.Error())
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
