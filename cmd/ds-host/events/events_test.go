package events

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// Uh oh, does the fact we have a single exported variable mean that
// We have to do things differently in test?

func TestSubscribeAsPaused(t *testing.T) {
	c := make(chan domain.AppspacePausedEvent)
	e := &AppspacePausedEvents{}
	e.Subscribe(c)

	if len(e.subscribers) != 1 {
		t.Error("expected subscribers length of 1")
	}

	e.Unsubscribe(c)

	if len(e.subscribers) != 0 {
		t.Error("expected subscribers length of 0")
	}
}

func TestSendAsPaused(t *testing.T) {
	c := make(chan domain.AppspacePausedEvent)
	e := &AppspacePausedEvents{}
	e.Subscribe(c)

	go func() {
		e.Send(domain.AppspaceID(7), true)
	}()

	eventPayload := <-c
	if !eventPayload.Paused {
		t.Error("payload not sent properly")
	}

	e.Unsubscribe(c)
}
