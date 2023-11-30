package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
)

//go:generate mockgen -destination=events_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks AppspaceFilesEvents,AppUrlDataEvents,AppspaceStatusEvents

// xxx go:generate mockgen -source=$GOFILE
// ^^ the above fails with an internal error: nil Pkg imports which I have no idea how to fix.

type GenericEvents[D events.DataTypes] interface {
	Subscribe() <-chan D
	SubscribeOwner(domain.UserID) <-chan D
	SubscribeApp(domain.AppID) <-chan D
	// more subs...
	Unsubscribe(ch <-chan D)
}

type AppspaceFilesEvents interface {
	Send(domain.AppspaceID)
	GenericEvents[domain.AppspaceID]
}

type AppUrlDataEvents interface {
	Send(domain.UserID, domain.AppURLData)
	GenericEvents[domain.AppURLData]
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
