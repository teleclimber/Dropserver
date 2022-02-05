package main

import (
	"encoding/json"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/twine-go/twine"
)

type SandboxControlService struct {
	DevSandboxManager interface {
		StopAppspace(domain.AppspaceID)
		SetInspect(bool)
	} `checkinject:"required"`
	DevSandboxMaker interface {
		SetInspect(bool)
	} `checkinject:"required"`
	InspectSandboxEvents interface {
		Send(bool)
		Subscribe(ch chan<- bool)
		Unsubscribe(ch chan<- bool)
	} `checkinject:"required"`
	SandboxStatusEvents interface {
		Subscribe() (SandboxStatus, <-chan SandboxStatus)
		Unsubscribe(<-chan SandboxStatus)
	} `checkinject:"required"`

	inspect bool
}

func (s *SandboxControlService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case setInspect:
		p := m.Payload()
		s.inspect = p[0] != 0x00
		s.DevSandboxMaker.SetInspect(s.inspect)
		s.DevSandboxManager.SetInspect(s.inspect)
		s.DevSandboxManager.StopAppspace(appspaceID)

		// This may seem bizarre: we are sending and reciveing "InspectSandboxEvents" on this same struct.
		// Here is why: there can be different instances of this struct, one per open dropserver-dev tab.
		s.InspectSandboxEvents.Send(s.inspect)
		m.SendOK()
	case stopSandbox:
		// force-kill for unruly scripts? Or for when we started with inspect by mistake.
		// TODO This could be an app sandbox or a migration sandbox too...
		s.DevSandboxManager.StopAppspace(appspaceID)
		m.SendOK()
	}
}

func (s *SandboxControlService) Start(t *twine.Twine) {
	inspectChan := make(chan bool)
	s.InspectSandboxEvents.Subscribe(inspectChan)
	go func() {
		for inspect := range inspectChan {
			go s.sendInspect(t, inspect)
		}
	}()
	go s.sendInspect(t, s.inspect)

	stat, statusChan := s.SandboxStatusEvents.Subscribe()
	go func() {
		for stat := range statusChan {
			go s.sendStatus(t, stat)
		}
	}()
	if stat.Type != "" {
		go s.sendStatus(t, stat)
	}

	t.WaitClose()

	s.SandboxStatusEvents.Unsubscribe(statusChan)
	s.InspectSandboxEvents.Unsubscribe(inspectChan)
}

const clientSetInspectCmd = 13 //? maybe ?
const sandboxStatus = 14

func (s *SandboxControlService) sendInspect(twine *twine.Twine, inspect bool) {
	payload := []byte{0}
	if inspect {
		payload = []byte{1}
	}
	_, err := twine.SendBlock(sandboxControlService, clientSetInspectCmd, payload)
	if err != nil {
		fmt.Println("sendInspect SendBlock Error: " + err.Error())
	}
}

func (s *SandboxControlService) sendStatus(twine *twine.Twine, stat SandboxStatus) {
	payload, err := json.Marshal(stat)
	if err != nil {
		fmt.Println("SandboxControlService json Marshal Error: " + err.Error())
	}
	_, err = twine.SendBlock(sandboxControlService, sandboxStatus, payload)
	if err != nil {
		fmt.Println("sendStatus SendBlock Error: " + err.Error())
	}
}
