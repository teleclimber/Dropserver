package events

import (
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// AppspacePausedEvents handles appspace pause and unpause events
type AppspacePausedEvents struct {
	subscribers []chan<- domain.AppspacePausedEvent
}

// Send sends an appspace paused or unpaused event
func (e *AppspacePausedEvents) Send(appspaceID domain.AppspaceID, paused bool) {
	p := domain.AppspacePausedEvent{AppspaceID: appspaceID, Paused: paused}
	for _, ch := range e.subscribers {
		ch <- p
	}
}

// Subscribe to an event for when an appspace is paused or unpaused
func (e *AppspacePausedEvents) Subscribe(ch chan<- domain.AppspacePausedEvent) {
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
}

// Unsubscribe to an event for when an appspace is paused or unpaused
func (e *AppspacePausedEvents) Unsubscribe(ch chan<- domain.AppspacePausedEvent) {
	e.removeSubscriber(ch)
}

func (e *AppspacePausedEvents) removeSubscriber(ch chan<- domain.AppspacePausedEvent) {
	// get a feeling you'll need a mutex to cover subscribers?
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}

/////////////////////////////////////////
// migration job events

// TODO: might need to make this per-appspace so frontend can track migrations as they happen?

//MigrationJobEvents forwards events related to migration jobs
type MigrationJobEvents struct {
	subscribers         []chan<- domain.MigrationJob
	appspaceSubscribers map[domain.AppspaceID][]chan<- domain.MigrationJob
}

// Send sends an appspace status event
func (e *MigrationJobEvents) Send(event domain.MigrationJob) {
	for _, ch := range e.subscribers {
		ch <- event
	}
	if e.appspaceSubscribers == nil {
		return
	}
	appspaceSubs, ok := e.appspaceSubscribers[event.AppspaceID]
	if ok {
		for _, ch := range appspaceSubs {
			ch <- event
		}
	}
}

// Subscribe to an event to know when the status of a migration has changed
func (e *MigrationJobEvents) Subscribe(ch chan<- domain.MigrationJob) {
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
}

// SubscribeAppspace to an event to know when the status of a migration for an appspace has changed
func (e *MigrationJobEvents) SubscribeAppspace(appspaceID domain.AppspaceID, ch chan<- domain.MigrationJob) {
	e.removeSubscriber(ch)
	if e.appspaceSubscribers == nil {
		e.appspaceSubscribers = make(map[domain.AppspaceID][]chan<- domain.MigrationJob)
	}
	e.appspaceSubscribers[appspaceID] = append(e.appspaceSubscribers[appspaceID], ch)
}

// Unsubscribe to the event
func (e *MigrationJobEvents) Unsubscribe(ch chan<- domain.MigrationJob) {
	e.removeSubscriber(ch)
}

func (e *MigrationJobEvents) removeSubscriber(ch chan<- domain.MigrationJob) {
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
	if e.appspaceSubscribers == nil {
		return
	}
	for appspaceID, subs := range e.appspaceSubscribers {
		for i, c := range subs {
			if c == ch {
				e.appspaceSubscribers[appspaceID][i] = e.appspaceSubscribers[appspaceID][len(subs)-1]
				e.appspaceSubscribers[appspaceID] = e.appspaceSubscribers[appspaceID][:len(subs)-1]
			}
		}
	}
}

////// Apppsace Files Event

// AppspaceFilesEvents notify subscribers that appsapce files
// have been written to outside of normal appspace use.
// Usually this means they were imported, or a backup restored
type AppspaceFilesEvents struct {
	subscribers []chan<- domain.AppspaceID
}

// Send sends an appspace paused or unpaused event
func (e *AppspaceFilesEvents) Send(appspaceID domain.AppspaceID) {
	for _, ch := range e.subscribers {
		ch <- appspaceID
	}
}

// Subscribe to an event for when an appspace is paused or unpaused
func (e *AppspaceFilesEvents) Subscribe(ch chan<- domain.AppspaceID) {
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
}

// Unsubscribe to an event for when an appspace is paused or unpaused
func (e *AppspaceFilesEvents) Unsubscribe(ch chan<- domain.AppspaceID) {
	e.removeSubscriber(ch)
}

func (e *AppspaceFilesEvents) removeSubscriber(ch chan<- domain.AppspaceID) {
	// get a feeling you'll need a mutex to cover subscribers?
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}

////////////////////////////////////////
// Appspace Status events
type appspaceStatusSubscriber struct {
	appspaceID domain.AppspaceID
	ch         chan<- domain.AppspaceStatusEvent
}

// AppspaceStatusEvents handles appspace pause and unpause events
type AppspaceStatusEvents struct {
	subscribers []appspaceStatusSubscriber
}

// Send sends an appspace status event
func (e *AppspaceStatusEvents) Send(appspaceID domain.AppspaceID, event domain.AppspaceStatusEvent) {
	for _, sub := range e.subscribers {
		if sub.appspaceID == appspaceID {
			sub.ch <- event
		}
	}
}

// Subscribe to an event to know when the status of an appspace has changed
func (e *AppspaceStatusEvents) Subscribe(appspaceID domain.AppspaceID, ch chan<- domain.AppspaceStatusEvent) {
	e.removeSubscriber(appspaceID, ch)
	e.subscribers = append(e.subscribers, appspaceStatusSubscriber{appspaceID, ch})
}

// Unsubscribe to the event
func (e *AppspaceStatusEvents) Unsubscribe(appspaceID domain.AppspaceID, ch chan<- domain.AppspaceStatusEvent) {
	e.removeSubscriber(appspaceID, ch)
}

// UnsubscribeChannel removes the channel from all subscriptions
func (e *AppspaceStatusEvents) UnsubscribeChannel(ch chan<- domain.AppspaceStatusEvent) {
	for i := len(e.subscribers) - 1; i >= 0; i-- {
		if e.subscribers[i].ch == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}

func (e *AppspaceStatusEvents) removeSubscriber(appspaceID domain.AppspaceID, ch chan<- domain.AppspaceStatusEvent) {
	// get a feeling you'll need a mutex to cover subscribers?
	for i, sub := range e.subscribers {
		if sub.appspaceID == appspaceID && sub.ch == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}

//////////////////////////////////////////
// Appspace Route Event
// TODO: Shouldn't subscribers be for specific appspaces?

// AppspaceRouteHitEvents handles appspace pause and unpause events
type AppspaceRouteHitEvents struct {
	subscribers []chan<- *domain.AppspaceRouteHitEvent
}

// Send sends an appspace paused or unpaused event
// Event's timestamp is set if needed
func (e *AppspaceRouteHitEvents) Send(routeEvent *domain.AppspaceRouteHitEvent) {
	if routeEvent.Timestamp.IsZero() {
		routeEvent.Timestamp = time.Now()
	}
	for _, ch := range e.subscribers {
		ch <- routeEvent
	}
}

// Subscribe to an event for when an appspace is paused or unpaused
func (e *AppspaceRouteHitEvents) Subscribe(ch chan<- *domain.AppspaceRouteHitEvent) {
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
}

// Unsubscribe to an event for when an appspace is paused or unpaused
func (e *AppspaceRouteHitEvents) Unsubscribe(ch chan<- *domain.AppspaceRouteHitEvent) {
	e.removeSubscriber(ch)
}

func (e *AppspaceRouteHitEvents) removeSubscriber(ch chan<- *domain.AppspaceRouteHitEvent) {
	// get a feeling you'll need a mutex to cover subscribers?
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}

///////////////////////////////
// App Version data change event

// AppVersionEvents forwards events about changes to an app version's metadata
// This is only useful in ds-dev.
type AppVersionEvents struct {
	subscribers []chan<- domain.AppID
}

// Send sends an appspace paused or unpaused event
func (e *AppVersionEvents) Send(appID domain.AppID) {
	for _, ch := range e.subscribers {
		ch <- appID
	}
}

// Subscribe to an event for when an appspace is paused or unpaused
func (e *AppVersionEvents) Subscribe(ch chan<- domain.AppID) {
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
}

// Unsubscribe to an event for when an appspace is paused or unpaused
func (e *AppVersionEvents) Unsubscribe(ch chan<- domain.AppID) {
	e.removeSubscriber(ch)
}

func (e *AppVersionEvents) removeSubscriber(ch chan<- domain.AppID) {
	// get a feeling you'll need a mutex to cover subscribers?
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}
