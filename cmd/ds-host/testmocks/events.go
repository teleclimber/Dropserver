package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/events"
)

//go:generate mockgen -destination=events_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks AppspaceFilesEvents,AppspaceTSNetModelEvents,AppUrlDataEvents,AppspaceStatusEvents,InstanceUserAuthsChangeEvents,AppspaceUsersChangeEvents,UserAppspacesEvent,AppspaceUsersEvent

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

type AppspaceTSNetModelEvents interface {
	Send(domain.AppspaceTSNetModelEvent)
	GenericEvents[domain.AppspaceTSNetModelEvent]
}

type AppUrlDataEvents interface {
	Send(domain.UserID, domain.AppURLData)
	GenericEvents[domain.AppURLData]
}

// AppspaceStatusEvents interface for mocking
type AppspaceStatusEvents interface {
	Send(domain.AppspaceStatusEvent)
	GenericEvents[domain.AppspaceStatusEvent]
}

// MigrationJobEvents interface for mocking
type MigrationJobEvents interface {
	Send(event domain.MigrationJob)
	Subscribe(ch chan<- domain.MigrationJob)
	SubscribeAppspace(appspaceID domain.AppspaceID, ch chan<- domain.MigrationJob)
	Unsubscribe(ch chan<- domain.MigrationJob)
}

// InstanceUserAuthsChangeEvents interface for mocking
type InstanceUserAuthsChangeEvents interface {
	Send(domain.UserID)
	Subscribe() <-chan domain.UserID
	Unsubscribe(ch <-chan domain.UserID)
}

// AppspaceUsersChangeEvents interface for mocking
type AppspaceUsersChangeEvents interface {
	Send(domain.AppspaceID)
	Subscribe() <-chan domain.AppspaceID
	Unsubscribe(ch <-chan domain.AppspaceID)
}

// UserAppspacesEvent interface for mocking
type UserAppspacesEvent interface {
	Send(domain.UserID)
	SubscribeUser(domain.UserID) <-chan struct{}
	Unsubscribe(ch <-chan struct{})
}

// AppspaceUsersEvent interface for mocking
type AppspaceUsersEvent interface {
	Send(domain.AppspaceID)
}
