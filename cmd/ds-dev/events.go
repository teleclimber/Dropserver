package main

import (
	"sync"
)

// PureEvent pushes data-free events to subscriber channels
type PureEvent struct {
	subLock     sync.Mutex
	subscribers []chan<- struct{}
}

// Send triggers a send to all subscribers
func (e *PureEvent) Send() {
	e.subLock.Lock()
	defer e.subLock.Unlock()
	for _, ch := range e.subscribers {
		ch <- struct{}{}
	}
}

// Subscribe to an event
func (e *PureEvent) Subscribe() chan struct{} {
	ch := make(chan struct{})
	e.subLock.Lock()
	defer e.subLock.Unlock()
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
	return ch
}

// Unsubscribe to an event
func (e *PureEvent) Unsubscribe(ch chan struct{}) {
	e.subLock.Lock()
	defer e.subLock.Unlock()
	e.removeSubscriber(ch)
}

func (e *PureEvent) removeSubscriber(ch chan struct{}) {
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
			close(ch)
		}
	}
}

// DevAppVersionEvents notifies that a change was made to an app version
type DevAppVersionEvents struct {
	subLock     sync.Mutex
	subscribers []chan<- string
}

// Send state of app"
func (e *DevAppVersionEvents) Send(state string) {
	e.subLock.Lock()
	defer e.subLock.Unlock()
	for _, ch := range e.subscribers {
		ch <- state
	}
}

// Subscribe to an event for when an app is changed
func (e *DevAppVersionEvents) Subscribe(ch chan<- string) {
	e.subLock.Lock()
	defer e.subLock.Unlock()
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
}

// Unsubscribe to an event for when an app is changed
func (e *DevAppVersionEvents) Unsubscribe(ch chan<- string) {
	e.subLock.Lock()
	defer e.subLock.Unlock()
	e.removeSubscriber(ch)
}

func (e *DevAppVersionEvents) removeSubscriber(ch chan<- string) {
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
			close(ch)
		}
	}
}

// InspectSandboxEvents notifes of changes to the sandbox inspect state
type InspectSandboxEvents struct {
	subLock     sync.Mutex
	subscribers []chan<- bool
}

// Send sends an app is changed
func (e *InspectSandboxEvents) Send(inspect bool) {
	e.subLock.Lock()
	defer e.subLock.Unlock()
	for _, ch := range e.subscribers {
		ch <- inspect
	}
}

// Subscribe to an event for when an app is changed
func (e *InspectSandboxEvents) Subscribe(ch chan<- bool) {
	e.subLock.Lock()
	defer e.subLock.Unlock()
	e.removeSubscriber(ch)
	e.subscribers = append(e.subscribers, ch)
}

// Unsubscribe to an event for when an app is changed
func (e *InspectSandboxEvents) Unsubscribe(ch chan<- bool) {
	e.subLock.Lock()
	defer e.subLock.Unlock()
	e.removeSubscriber(ch)
}

func (e *InspectSandboxEvents) removeSubscriber(ch chan<- bool) {
	for i, c := range e.subscribers {
		if c == ch {
			e.subscribers[i] = e.subscribers[len(e.subscribers)-1]
			e.subscribers = e.subscribers[:len(e.subscribers)-1]
			close(ch)
		}
	}
}
