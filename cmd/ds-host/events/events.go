package events

import (
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

/////////////////////////////////////////
// migration job events

// MigrationJobEvents forwards events related to migration jobs
type MigrationJobEvents struct {
	subscribers eventSubs[domain.MigrationJob]
	ownerSubs   eventIDSubs[domain.UserID, domain.MigrationJob]
}

func (e *MigrationJobEvents) Subscribe() <-chan domain.MigrationJob {
	return e.subscribers.subscribe()
}

func (e *MigrationJobEvents) SubscribeOwner(ownerID domain.UserID) <-chan domain.MigrationJob {
	return e.ownerSubs.subscribe(ownerID)
}

func (e *MigrationJobEvents) Unsubscribe(ch <-chan domain.MigrationJob) {
	e.subscribers.unsubscribe(ch)
	e.ownerSubs.unsubscribe(ch)
}

func (e *MigrationJobEvents) Send(data domain.MigrationJob) {
	e.subscribers.send(data)
	e.ownerSubs.send(data.OwnerID, data)
}

////// Apppsace Files Event

// AppspaceFilesEvents notify subscribers that appsapce files
// have been written to outside of normal appspace use.
// Usually this means they were imported, or a backup restored
type AppspaceFilesEvents struct {
	subscribers eventSubs[domain.AppspaceID]
}

// Send sends an appspace paused or unpaused event
func (e *AppspaceFilesEvents) Send(appspaceID domain.AppspaceID) {
	e.subscribers.send(appspaceID)
}

// Subscribe to an event for when an appspace is paused or unpaused
func (e *AppspaceFilesEvents) Subscribe() <-chan domain.AppspaceID {
	return e.subscribers.subscribe()
}

// Unsubscribe to an event for when an appspace is paused or unpaused
func (e *AppspaceFilesEvents) Unsubscribe(ch <-chan domain.AppspaceID) {
	e.subscribers.unsubscribe(ch)
}

// //////////////////////////////////////

type AppspaceStatusEvents struct {
	subscribers eventSubs[domain.AppspaceStatusEvent]
	ownerSubs   eventIDSubs[domain.UserID, domain.AppspaceStatusEvent]
}

func (e *AppspaceStatusEvents) Subscribe() <-chan domain.AppspaceStatusEvent {
	return e.subscribers.subscribe()
}

func (e *AppspaceStatusEvents) SubscribeOwner(ownerID domain.UserID) <-chan domain.AppspaceStatusEvent {
	return e.ownerSubs.subscribe(ownerID)
}

func (e *AppspaceStatusEvents) Unsubscribe(ch <-chan domain.AppspaceStatusEvent) {
	e.subscribers.unsubscribe(ch)
	e.ownerSubs.unsubscribe(ch)
}

func (e *AppspaceStatusEvents) Send(data domain.AppspaceStatusEvent) {
	e.subscribers.send(data)
	e.ownerSubs.send(data.OwnerID, data)
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

// AppUrlDataEvents sends AppURLData
type AppUrlDataEvents struct {
	ownerSubs eventIDSubs[domain.UserID, domain.AppURLData]
	appSubs   eventIDSubs[domain.AppID, domain.AppURLData]
}

func (e *AppUrlDataEvents) SubscribeOwner(ownerID domain.UserID) <-chan domain.AppURLData {
	return e.ownerSubs.subscribe(ownerID)
}

func (e *AppUrlDataEvents) SubscribeApp(appID domain.AppID) <-chan domain.AppURLData {
	return e.appSubs.subscribe(appID)
}

func (e *AppUrlDataEvents) Unsubscribe(ch <-chan domain.AppURLData) {
	e.appSubs.unsubscribe(ch)
	e.ownerSubs.unsubscribe(ch)
}

func (e *AppUrlDataEvents) Send(ownerID domain.UserID, data domain.AppURLData) {
	e.ownerSubs.send(ownerID, data)
	e.appSubs.send(data.AppID, data)
}
