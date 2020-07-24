package events

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

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
