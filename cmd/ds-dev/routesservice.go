package main

import (
	"encoding/json"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

// RoutesService is a twine service that keeps up to date with
// an appspace's route table.
type RoutesService struct {
	AppspaceRouteModels interface {
		GetV0(domain.AppspaceID) domain.V0RouteModel
	}
	AppspaceRouteEvents interface {
		Subscribe(domain.AppspaceID, chan<- domain.AppspaceRouteEvent)
		Unsubscribe(domain.AppspaceID, chan<- domain.AppspaceRouteEvent)
	}
	AppspaceFilesEvents interface {
		Subscribe(chan<- domain.AppspaceID)
		Unsubscribe(chan<- domain.AppspaceID)
	}
}

func (s *RoutesService) Start(t *twine.Twine) {
	appspaceRouteEvent := make(chan domain.AppspaceRouteEvent)
	s.AppspaceRouteEvents.Subscribe(appspaceID, appspaceRouteEvent)
	go func() {
		for routeEvent := range appspaceRouteEvent {
			go s.sendAppspaceRoutesPatch(t, routeEvent)
		}
	}()

	// When appspace files are changed externally, resend all routes
	asFilesCh := make(chan domain.AppspaceID)
	s.AppspaceFilesEvents.Subscribe(asFilesCh)
	go func() {
		for range asFilesCh {
			go s.sendAppspaceRoutes(t)
		}
	}()

	// push current routes:
	s.sendAppspaceRoutes(t)

	// Wait for twine to close and shut it all down.
	t.WaitClose()

	fmt.Println("closing route service")

	s.AppspaceRouteEvents.Unsubscribe(appspaceID, appspaceRouteEvent)
	close(appspaceRouteEvent)

	s.AppspaceFilesEvents.Unsubscribe(asFilesCh)
	close(asFilesCh)
}

// HandleMessage handles incoming mesages for this service.
func (s *RoutesService) HandleMessage(m twine.ReceivedMessageI) {
	// we don't expect an imcping message, so return an error
	m.SendError("did not expect an incoming message on routes service")
}

const loadAllRoutesCmd = 11
const patchRoutesCmd = 12

type appspaceRoutes struct {
	Path   string                        `json:"path"`
	Routes *[]domain.AppspaceRouteConfig `json:"routes"`
}

func (s *RoutesService) sendAppspaceRoutes(twine *twine.Twine) {
	v0routeModel := s.AppspaceRouteModels.GetV0(appspaceID)
	routes, err := v0routeModel.GetAll()
	if err != nil {
		fmt.Println("sendAppspaceRoutes error getting routes: " + err.Error())
	}

	data := appspaceRoutes{
		Path:   "",
		Routes: routes}

	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println("sendAppspaceRoutes json Marshal Error: " + err.Error())
	}

	_, err = twine.SendBlock(appspaceRouteService, loadAllRoutesCmd, bytes)
	if err != nil {
		fmt.Println("sendAppspaceRoutes SendBlock Error: " + err.Error())
	}
}

func (s *RoutesService) sendAppspaceRoutesPatch(twine *twine.Twine, routeEvent domain.AppspaceRouteEvent) {
	v0routeModel := s.AppspaceRouteModels.GetV0(appspaceID)
	routes, err := v0routeModel.GetPath(routeEvent.Path)
	if err != nil {
		fmt.Println("sendAppspaceRoutesPatch error getting routes: " + err.Error())
	}

	data := appspaceRoutes{
		Path:   routeEvent.Path,
		Routes: routes}

	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println("sendAppspaceRoutesPatch json Marshal Error: " + err.Error())
	}

	_, err = twine.SendBlock(appspaceRouteService, patchRoutesCmd, bytes)
	if err != nil {
		fmt.Println("sendAppspaceRoutesPatch SendBlock Error: " + err.Error())
	}
}
