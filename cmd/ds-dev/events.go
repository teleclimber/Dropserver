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
