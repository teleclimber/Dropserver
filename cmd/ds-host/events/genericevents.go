package events

import (
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type SubscribeIDs interface {
	domain.UserID |
		domain.AppID |
		domain.AppspaceID
}
type DataTypes interface {
	domain.AppURLData |
		domain.AppspaceID |
		domain.AppspaceStatusEvent |
		domain.MigrationJob |
		domain.AppGetEvent
}

type eventIDSubs[T SubscribeIDs, D DataTypes] struct {
	subsMux     sync.Mutex
	subscribers map[T]*eventSubs[D]
}

func (s *eventIDSubs[T, D]) subscribe(subID T) <-chan D {
	s.subsMux.Lock()
	defer s.subsMux.Unlock()
	if s.subscribers == nil {
		s.subscribers = make(map[T]*eventSubs[D])
	}
	var es *eventSubs[D]
	es = s.subscribers[subID]
	if es == nil {
		es = &eventSubs[D]{}
		s.subscribers[subID] = es
	}
	return es.subscribe()
}

func (s *eventIDSubs[T, D]) unsubscribe(ch <-chan D) {
	s.subsMux.Lock()
	defer s.subsMux.Unlock()
	if s.subscribers == nil {
		return
	}
	for _, subs := range s.subscribers {
		subs.unsubscribe(ch)
	}
}

func (s *eventIDSubs[T, D]) send(subID T, data D) {
	s.subsMux.Lock()
	defer s.subsMux.Unlock()
	subs := s.subscribers[subID]
	if subs != nil {
		subs.send(data)
	}
}

type eventSubs[D DataTypes] struct {
	subsMux     sync.Mutex
	subscribers []chan D
}

func (s *eventSubs[D]) subscribe() <-chan D {
	s.subsMux.Lock()
	defer s.subsMux.Unlock()
	ch := make(chan D)
	if s.subscribers == nil {
		s.subscribers = make([]chan D, 0, 10)
	}
	s.subscribers = append(s.subscribers, ch)
	return ch
}

func (s *eventSubs[D]) unsubscribe(ch <-chan D) {
	s.subsMux.Lock()
	defer s.subsMux.Unlock()
	if s.subscribers == nil {
		return
	}
	for i, c := range s.subscribers {
		if c == ch {
			s.subscribers[i] = s.subscribers[len(s.subscribers)-1]
			s.subscribers = s.subscribers[:len(s.subscribers)-1]
			close(c)
			return
		}
	}
}

func (s *eventSubs[D]) send(data D) {
	s.subsMux.Lock()
	defer s.subsMux.Unlock()
	for _, ch := range s.subscribers {
		ch <- data
	}
}
