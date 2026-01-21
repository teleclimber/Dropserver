package events

import (
	"fmt"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestEventIDSubsSendNil(t *testing.T) {
	s := eventIDSubs[domain.AppspaceID, domain.AppURLData]{}
	s.send(domain.AppspaceID(3), domain.AppURLData{URL: "abc"})
}

func TestEventIDSubsSubscribe(t *testing.T) {
	s := eventIDSubs[domain.AppspaceID, domain.AppURLData]{}
	as1 := domain.AppspaceID(1)
	ch1 := s.subscribe(domain.AppspaceID(as1))
	if len(s.subscribers) != 1 {
		t.Error("expected one")
	}
	if len(s.subscribers[as1].subscribers) != 1 {
		t.Error("expected one subscriber")
	}
	s.subscribe(domain.AppspaceID(2))
	if len(s.subscribers) != 2 {
		t.Error("expected one")
	}

	s.unsubscribe(ch1)
	if len(s.subscribers[as1].subscribers) != 0 {
		t.Error("expected zero subscribers")
	}
}

func TestEventIDSubsSend(t *testing.T) {
	s := eventIDSubs[domain.AppspaceID, domain.AppURLData]{}

	as3 := domain.AppspaceID(3)

	doSend := make(chan struct{})
	doneCh := make(chan error)

	go func() {
		ch := s.subscribe(as3)
		doSend <- struct{}{}
		urlData := <-ch
		if urlData.URL != "abc" {
			doneCh <- fmt.Errorf("got wrong url data id")
		}
		s.unsubscribe(ch)
		close(doneCh)
	}()

	<-doSend
	s.send(as3, domain.AppURLData{URL: "abc"})

	err := <-doneCh
	if err != nil {
		t.Error(err)
	}
}

func TestEventIDSubsMultiSendNil(t *testing.T) {
	s := eventIDSubs[domain.UserID, domain.AppURLData]{}
	s.multiSend([]domain.UserID{3, 4, 5}, domain.AppURLData{URL: "abc"})
}

func TestEventIDSubsMultiSend(t *testing.T) {
	s := eventIDSubs[domain.UserID, domain.AppURLData]{}

	u1 := domain.UserID(1)
	u2 := domain.UserID(2)
	u3 := domain.UserID(3) // no subscriber for this one

	doSend := make(chan struct{})
	doneCh := make(chan error)

	// Subscriber for u1
	go func() {
		ch := s.subscribe(u1)
		doSend <- struct{}{}
		urlData := <-ch
		if urlData.URL != "xyz" {
			doneCh <- fmt.Errorf("u1: got wrong url data: %v", urlData.URL)
			return
		}
		s.unsubscribe(ch)
		doneCh <- nil
	}()

	// Subscriber for u2
	go func() {
		ch := s.subscribe(u2)
		doSend <- struct{}{}
		urlData := <-ch
		if urlData.URL != "xyz" {
			doneCh <- fmt.Errorf("u2: got wrong url data: %v", urlData.URL)
			return
		}
		s.unsubscribe(ch)
		doneCh <- nil
	}()

	// Wait for both subscribers to be ready
	<-doSend
	<-doSend

	// Send to u1, u2, and u3 (u3 has no subscriber)
	s.multiSend([]domain.UserID{u1, u2, u3}, domain.AppURLData{URL: "xyz"})

	// Check both subscribers received the data
	for i := 0; i < 2; i++ {
		err := <-doneCh
		if err != nil {
			t.Error(err)
		}
	}
}

func TestEventSubsSendNil(t *testing.T) {
	s := eventSubs[domain.AppspaceID]{}
	s.send(domain.AppspaceID(3))
}

func TestEventSubsSubscribe(t *testing.T) {
	s := eventSubs[domain.AppspaceID]{}
	ch := s.subscribe()
	if len(s.subscribers) != 1 {
		t.Error("expected one subscriber")
	}
	s.unsubscribe(ch)
	if len(s.subscribers) != 0 {
		t.Error("expected zero subscribers")
	}
	// multiple calls should be a problem:
	s.unsubscribe(ch)
	if len(s.subscribers) != 0 {
		t.Error("expected zero subscribers")
	}
}

func TestEventSubsMultiSubscribe(t *testing.T) {
	s := eventSubs[domain.AppspaceID]{}
	ch1 := s.subscribe()
	ch2 := s.subscribe()
	if len(s.subscribers) != 2 {
		t.Error("expected two subscriber")
	}
	s.unsubscribe(ch1)
	if len(s.subscribers) != 1 {
		t.Error("expected zero subscribers")
	}
	s.unsubscribe(ch2)
	if len(s.subscribers) != 0 {
		t.Error("expected zero subscribers")
	}
}

func TestEventSubsSend(t *testing.T) {
	s := eventSubs[domain.AppspaceID]{}

	doSend := make(chan struct{})
	doneCh := make(chan error)

	go func() {
		ch := s.subscribe()
		doSend <- struct{}{}
		asID := <-ch
		if asID != domain.AppspaceID(3) {
			doneCh <- fmt.Errorf("got wrong appspace id %v", asID)
		}
		s.unsubscribe(ch)
		close(doneCh)
	}()

	<-doSend
	s.send(domain.AppspaceID(3))

	err := <-doneCh
	if err != nil {
		t.Error(err)
	}

	if len(s.subscribers) != 0 {
		t.Error("expected zero subscribers")
	}
}
