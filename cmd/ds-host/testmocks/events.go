package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=events_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks AppspaceFilesEvents,AppspaceStatusEvents

// xxx go:generate mockgen -source=$GOFILE
// ^^ the above fails with an internal error: nil Pkg imports which I have no idea how to fix.

// AppspaceFilesEvents interface for mocking
type AppspaceFilesEvents interface {
	Send(appspaceID domain.AppspaceID)
	Subscribe() <-chan domain.AppspaceID
	Unsubscribe(ch <-chan domain.AppspaceID)
}

// AppspaceStatusEvents interface for mocking
type AppspaceStatusEvents interface {
	Subscribe(appspaceID domain.AppspaceID, ch chan<- domain.AppspaceStatusEvent)
	Unsubscribe(appspaceID domain.AppspaceID, ch chan<- domain.AppspaceStatusEvent)
	UnsubscribeChannel(chan<- domain.AppspaceStatusEvent)
	Send(domain.AppspaceID, domain.AppspaceStatusEvent)
}

// MigrationJobEvents interface for mocking
type MigrationJobEvents interface {
	Send(event domain.MigrationJob)
	Subscribe(ch chan<- domain.MigrationJob)
	SubscribeAppspace(appspaceID domain.AppspaceID, ch chan<- domain.MigrationJob)
	Unsubscribe(ch chan<- domain.MigrationJob)
}
