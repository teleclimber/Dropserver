package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

const routeHitEventCmd = 11

type RequestJSON struct {
	URL    string `json:"url"`
	Method string `json:"method"`
}
type RouteHitEventJSON struct {
	Timestamp   time.Time                   `json:"timestamp"`
	Request     RequestJSON                 `json:"request"`
	RouteConfig *domain.AppspaceRouteConfig `json:"route_config"` // this might be nil.OK?
	User        *DevAppspaceUser            `json:"user"`         //make nil OK
	IsOwner     bool                        `json:"is_owner"`
	Status      int                         `json:"status"`
}

// RouteHitService forwards route hit events to provided twine instance
type RouteHitService struct {
	RouteHitEvents interface {
		Subscribe(ch chan<- *domain.AppspaceRouteHitEvent)
		Unsubscribe(ch chan<- *domain.AppspaceRouteHitEvent)
	}
	AppspaceUserModels interface {
		GetV0(domain.AppspaceID) domain.V0UserModel
	}
	DevAppspaceContactModel *DevAppspaceContactModel
}

func (s *RouteHitService) Start(t *twine.Twine) {

	routeEventsChan := make(chan *domain.AppspaceRouteHitEvent)
	s.RouteHitEvents.Subscribe(routeEventsChan)
	go func() {
		for routeEvent := range routeEventsChan {
			go s.sendRouteEvent(t, routeEvent)
		}
	}()

	// Wait for twine to close and shut it all down.
	t.WaitClose()

	s.RouteHitEvents.Unsubscribe(routeEventsChan)
	close(routeEventsChan)
}

// HandleMessage is a no-op, error producing function in route hit service
func (s *RouteHitService) HandleMessage(m twine.ReceivedMessageI) {
	panic("did not expect a mesage to route hit service.")
}

func (s *RouteHitService) sendRouteEvent(twine *twine.Twine, routeEvent *domain.AppspaceRouteHitEvent) {
	send := RouteHitEventJSON{
		Timestamp: routeEvent.Timestamp,
		Request: RequestJSON{
			URL:    routeEvent.Request.URL.String(),
			Method: routeEvent.Request.Method},
		RouteConfig: routeEvent.RouteConfig,
		Status:      routeEvent.Status}

	if routeEvent.Credentials.ProxyID != "" {
		userModel := s.AppspaceUserModels.GetV0(appspaceID)
		v0user, err := userModel.Get(routeEvent.Credentials.ProxyID)
		if err != nil {
			panic(err)
		}
		user := V0ToDevApspaceUser(v0user)
		send.User = &user

		contact, err := s.DevAppspaceContactModel.GetByProxy(appspaceID, routeEvent.Credentials.ProxyID)
		if err != nil {
			panic(err)
		}
		send.IsOwner = contact.IsOwner
	}

	bytes, err := json.Marshal(send)
	if err != nil {
		// meh
		fmt.Println("sendRouteEvent json Marshal Error: " + err.Error())
	}

	_, err = twine.SendBlock(routeEventService, routeHitEventCmd, bytes)
	if err != nil {
		//urhg
		fmt.Println("sendRouteEvent SendBlock Error: " + err.Error())
	}

}
