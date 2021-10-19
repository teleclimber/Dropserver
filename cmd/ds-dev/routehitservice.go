package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/twine-go/twine"
)

const routeHitEventCmd = 11

type RequestJSON struct {
	URL    string `json:"url"`
	Method string `json:"method"`
}
type RouteHitEventJSON struct {
	Timestamp     time.Time            `json:"timestamp"`
	Request       RequestJSON          `json:"request"`
	V0RouteConfig *domain.V0AppRoute   `json:"v0_route_config"` // this might be nil.OK?
	User          *domain.AppspaceUser `json:"user"`            //make nil OK
	Authorized    bool                 `json:"authorized"`
	Status        int                  `json:"status"`
}

// RouteHitService forwards route hit events to provided twine instance
type RouteHitService struct {
	RouteHitEvents interface {
		Subscribe(ch chan<- *domain.AppspaceRouteHitEvent)
		Unsubscribe(ch chan<- *domain.AppspaceRouteHitEvent)
	} `checkinject:"required"`
	AppspaceUsersModelV0 interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
	} `checkinject:"required"`
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
	panic("did not expect a message to route hit service.")
}

func (s *RouteHitService) sendRouteEvent(twine *twine.Twine, routeEvent *domain.AppspaceRouteHitEvent) {
	send := RouteHitEventJSON{
		Timestamp: routeEvent.Timestamp,
		Request: RequestJSON{
			URL:    routeEvent.Request.URL.String(),
			Method: routeEvent.Request.Method},
		V0RouteConfig: routeEvent.V0RouteConfig,
		Authorized:    routeEvent.Authorized,
		Status:        routeEvent.Status}

	if routeEvent.Credentials.ProxyID != "" {
		user, err := s.AppspaceUsersModelV0.Get(appspaceID, routeEvent.Credentials.ProxyID)
		if err != nil {
			panic(err)
		}
		send.User = &user
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
