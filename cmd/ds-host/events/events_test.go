package events

import (
	"errors"
	"sync"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// Uh oh, does the fact we have a single exported variable mean that
// We have to do things differently in test?

// test appspace status events with per-appspace subscription
func TestSubscribeAsStatus(t *testing.T) {
	appspaceID := domain.AppspaceID(7)
	c := make(chan domain.AppspaceStatusEvent)
	e := &AppspaceStatusEvents{}
	e.Subscribe(appspaceID, c)

	if len(e.subscribers) != 1 {
		t.Error("expected subscribers length of 1")
	}

	doneCh := make(chan error)

	go func() {
		event := <-c
		if event.AppspaceSchema != 15 {
			doneCh <- errors.New("not 15")
		} else {
			close(doneCh)
		}
	}()

	e.Send(domain.AppspaceID(13), domain.AppspaceStatusEvent{}) // no effect
	e.Send(appspaceID, domain.AppspaceStatusEvent{AppspaceSchema: 15})

	err := <-doneCh
	if err != nil {
		t.Error(err)
	}

	e.Unsubscribe(appspaceID, c)

	if len(e.subscribers) != 0 {
		t.Error("expected subscribers length of 0")
	}
}

func TestMultiSubscribeAsStatus(t *testing.T) {
	appspaceID1 := domain.AppspaceID(7)
	appspaceID2 := domain.AppspaceID(11)
	c := make(chan domain.AppspaceStatusEvent)
	e := &AppspaceStatusEvents{}
	e.Subscribe(appspaceID1, c)
	e.Subscribe(appspaceID2, c)

	if len(e.subscribers) != 2 {
		t.Error("expected subscribers length of 2")
	}

	e.UnsubscribeChannel(c)

	if len(e.subscribers) != 0 {
		t.Error("expected subscribers length of 0")
	}
}

func TestMigrationJobAppspace(t *testing.T) {
	appspaceID1 := domain.AppspaceID(7)
	appspaceID2 := domain.AppspaceID(11)

	e := &MigrationJobEvents{}

	c := e.SubscribeAppspace(appspaceID1)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		d := <-c
		if d.JobID != 77 {
			t.Error("got the wrong data")
		}
		wg.Done()
	}()

	e.Send(domain.MigrationJob{AppspaceID: appspaceID2})
	e.Send(domain.MigrationJob{AppspaceID: appspaceID1, JobID: 77})

	wg.Wait()

	e.Unsubscribe(c)

	if len(e.appspaceSubscribers[appspaceID1]) != 0 {
		t.Error("unsubscribe did not work")
	}
}
