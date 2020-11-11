package main

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// AppVersionEvents notifies that a change was made to an app version
type AppVersionEvents struct {
	subscribers []chan<- domain.AppID
}

// Send sends an app is changed
func (e *AppVersionEvents) Send(appID domain.AppID) {
	for _, ch := range e.subscribers {
		ch <- appID
	}
}

// Subscribe to an event for when an app is changed
func (e *AppVersionEvents) Subscribe(ch chan<- domain.AppID) {
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
}

// Unsubscribe to an event for when an app is changed
func (e *AppVersionEvents) Unsubscribe(ch chan<- domain.AppID) {
	e.removeSubscriber(ch)
}

func (e *AppVersionEvents) removeSubscriber(ch chan<- domain.AppID) {
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}

////////////////////////////////////////
// Appspace Status events
type appspaceRouteSubscriber struct {
	appspaceID domain.AppspaceID
	ch         chan<- domain.AppspaceRouteEvent
}

// AppspaceRouteEvents handles appspace pause and unpause events
type AppspaceRouteEvents struct {
	subscribers []appspaceRouteSubscriber
}

// Send sends an appspace status event
func (e *AppspaceRouteEvents) Send(appspaceID domain.AppspaceID, event domain.AppspaceRouteEvent) {
	for _, sub := range e.subscribers {
		if sub.appspaceID == appspaceID {
			sub.ch <- event
		}
	}
}

// Subscribe to an event to know when the status of an appspace has changed
func (e *AppspaceRouteEvents) Subscribe(appspaceID domain.AppspaceID, ch chan<- domain.AppspaceRouteEvent) {
	e.removeSubscriber(appspaceID, ch)
	e.subscribers = append(e.subscribers, appspaceRouteSubscriber{appspaceID, ch})
}

// Unsubscribe to the event
func (e *AppspaceRouteEvents) Unsubscribe(appspaceID domain.AppspaceID, ch chan<- domain.AppspaceRouteEvent) {
	e.removeSubscriber(appspaceID, ch)
}

// UnsubscribeChannel removes the channel from all subscriptions
func (e *AppspaceRouteEvents) UnsubscribeChannel(ch chan<- domain.AppspaceRouteEvent) {
	for i := len(e.subscribers) - 1; i >= 0; i-- {
		if e.subscribers[i].ch == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}

func (e *AppspaceRouteEvents) removeSubscriber(appspaceID domain.AppspaceID, ch chan<- domain.AppspaceRouteEvent) {
	// get a feeling you'll need a mutex to cover subscribers?
	for i, sub := range e.subscribers {
		if sub.appspaceID == appspaceID && sub.ch == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
		}
	}
}
