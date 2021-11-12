package main

import (
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
		s.InspectSandboxEvents.Send(s.inspect) // OK this is bizarre we er sneding and reciveing on this same object.
		m.SendOK()
	case stopSandbox:
		// force-kill for unruly scripts? Or for when we started with inspect by mistake.
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

	t.WaitClose()

	s.InspectSandboxEvents.Unsubscribe(inspectChan)
}

const clientSetInspectCmd = 13 //? maybe ?

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
