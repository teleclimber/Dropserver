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

// ////////////////////////////////////
type AppspaceTSNetModelEvents struct {
	subscribers eventSubs[domain.AppspaceTSNetModelEvent]
}

func (e *AppspaceTSNetModelEvents) Subscribe() <-chan domain.AppspaceTSNetModelEvent {
	return e.subscribers.subscribe()
}

func (e *AppspaceTSNetModelEvents) Unsubscribe(ch <-chan domain.AppspaceTSNetModelEvent) {
	e.subscribers.unsubscribe(ch)
}

func (e *AppspaceTSNetModelEvents) Send(data domain.AppspaceTSNetModelEvent) {
	e.subscribers.send(data)
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

type AppspaceTSNetStatusEvents struct {
	Relations interface {
		GetAppspaceOwnerID(appspaceID domain.AppspaceID) (domain.UserID, bool)
	} `checkinject:"required"`
	subscribers eventSubs[domain.TSNetAppspaceStatus]
	ownerSubs   eventIDSubs[domain.UserID, domain.TSNetAppspaceStatus]
}

func (e *AppspaceTSNetStatusEvents) Subscribe() <-chan domain.TSNetAppspaceStatus {
	return e.subscribers.subscribe()
}

func (e *AppspaceTSNetStatusEvents) SubscribeOwner(ownerID domain.UserID) <-chan domain.TSNetAppspaceStatus {
	return e.ownerSubs.subscribe(ownerID)
}

func (e *AppspaceTSNetStatusEvents) Unsubscribe(ch <-chan domain.TSNetAppspaceStatus) {
	e.subscribers.unsubscribe(ch)
	e.ownerSubs.unsubscribe(ch)
}

func (e *AppspaceTSNetStatusEvents) Send(data domain.TSNetAppspaceStatus) {
	e.subscribers.send(data)
	ownerID, ok := e.Relations.GetAppspaceOwnerID(data.AppspaceID)
	if ok {
		e.ownerSubs.send(ownerID, data)
	}
}

// Appspace TSNet Users/peers changed notification
// It merely sends the appspace ID of the appspace whose peers changed.
type AppspaceTSNetPeersEvents struct {
	Relations interface {
		GetAppspaceOwnerID(appspaceID domain.AppspaceID) (domain.UserID, bool)
	} `checkinject:"required"`
	subscribers eventSubs[domain.AppspaceID] // not clear why we need global subscribers for this?
	ownerSubs   eventIDSubs[domain.UserID, domain.AppspaceID]
}

func (e *AppspaceTSNetPeersEvents) Subscribe() <-chan domain.AppspaceID {
	return e.subscribers.subscribe()
}

func (e *AppspaceTSNetPeersEvents) SubscribeOwner(ownerID domain.UserID) <-chan domain.AppspaceID {
	return e.ownerSubs.subscribe(ownerID)
}

func (e *AppspaceTSNetPeersEvents) Unsubscribe(ch <-chan domain.AppspaceID) {
	e.subscribers.unsubscribe(ch)
	e.ownerSubs.unsubscribe(ch)
}

func (e *AppspaceTSNetPeersEvents) Send(appspaceID domain.AppspaceID) {
	e.subscribers.send(appspaceID)
	ownerID, ok := e.Relations.GetAppspaceOwnerID(appspaceID)
	if ok {
		e.ownerSubs.send(ownerID, appspaceID)
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

// AppGetterEvents
type AppGetterEvents struct {
	ownerSubs eventIDSubs[domain.UserID, domain.AppGetEvent]
}

func (e *AppGetterEvents) SubscribeOwner(ownerID domain.UserID) <-chan domain.AppGetEvent {
	return e.ownerSubs.subscribe(ownerID)
}

func (e *AppGetterEvents) Unsubscribe(ch <-chan domain.AppGetEvent) {
	e.ownerSubs.unsubscribe(ch)
}

func (e *AppGetterEvents) Send(data domain.AppGetEvent) {
	e.ownerSubs.send(data.OwnerID, data)
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
