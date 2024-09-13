package events

import (
	"sync"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

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
