package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/twine"
)

const routeEventService = 11
const appspaceControlService = 12 // incmoing? For appspace control (pause, migrate)
const appspaceStatusService = 13

// DropserverDevServer serves routes at dropserver-dev which control
// the handling of the app server
type DropserverDevServer struct {
	Config        *domain.RuntimeConfig
	AppspaceModel interface {
		Pause(appspaceID domain.AppspaceID, pause bool) domain.Error
	}
	AppspaceStatusEvents interface {
		Subscribe(domain.AppspaceID, chan<- domain.AppspaceStatusEvent)
		Unsubscribe(domain.AppspaceID, chan<- domain.AppspaceStatusEvent)
	}
	RouteEvents interface {
		Subscribe(ch chan<- *domain.AppspaceRouteHitEvent)
		Unsubscribe(ch chan<- *domain.AppspaceRouteHitEvent)
	}

	baseData BaseData
}

func (s *DropserverDevServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	fmt.Println("DropserverDevServer url: " + req.URL.String())

	// switch through possible commands
	// and default to serving frontend files
	// what are some things?
	// - open twine connection over websockets....
	// - get debugger port?
	// - get basics of application?

	// What information do we want to have on frontend:
	// - basic info about app being run (app meta)
	// - appspace meta
	// - appspace routes
	// - explore appspace files
	// - explore appspace dbs
	// - live sandbox status (stopped starting, running; debug mode)
	// - live http hits

	staticHandler := http.FileServer(http.Dir(s.Config.Exec.StaticAssetsDir))

	head, _ := shiftpath.ShiftPath(req.URL.Path)
	switch head {

	case "base-data": // temporary
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		json.NewEncoder(res).Encode(s.baseData)

	case "livedata":
		s.StartLivedata(res, req)

	default:
		// file serve the frontend dist dir
		//http.Error(res, "dropserver-dev route not found", http.StatusNotFound)
		staticHandler.ServeHTTP(res, req)
	}

}

func (s *DropserverDevServer) StartLivedata(res http.ResponseWriter, req *http.Request) {
	t, err := twine.NewWebsocketServer(res, req)
	if err != nil {
		fmt.Println("twine.NewWebsocketServer(res, req) " + err.Error())
		//http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	_, ok := <-t.ReadyChan
	if !ok {
		fmt.Println("failed to start Twine")
	}

	// then subscribe to various events and push them down as new t.Send
	appspaceStatusChan := make(chan domain.AppspaceStatusEvent)
	s.AppspaceStatusEvents.Subscribe(appspaceID, appspaceStatusChan)
	go func() {
		for statusEvent := range appspaceStatusChan {
			go s.sendAppspaceStatusEvent(t, statusEvent)
		}
	}()

	routeEventsChan := make(chan *domain.AppspaceRouteHitEvent)
	s.RouteEvents.Subscribe(routeEventsChan)
	go func() {
		for routeEvent := range routeEventsChan {
			go s.sendRouteEvent(t, routeEvent)
		}
	}()

	// need to receive messages too
	go func() {
		for m := range t.MessageChan {
			switch m.ServiceID() {
			case appspaceControlService:
				go s.handleAppspaceMessage(m)
			default:
				m.SendError("Service not found")
			}

		}
	}()

	// Then handle shutting down
	// the close signal might come from teh client, which will have to propagate via Twine stack to here
	// Or it will come from the host
	// For now just handle remote cloes:
	go func() {
		for err := range t.ErrorChan {
			fmt.Println("twine error chan err " + err.Error())
		}
		// when ErrorChan closes, means Twine connection is down, so unsubscribe
		s.AppspaceStatusEvents.Unsubscribe(appspaceID, appspaceStatusChan)
		close(appspaceStatusChan)

		s.RouteEvents.Unsubscribe(routeEventsChan)
		close(routeEventsChan)

		fmt.Println("unsubscribed")
	}()
}

const pauseAppspaceCmd = 11
const unpauseAppspaceCmd = 12

func (s *DropserverDevServer) handleAppspaceMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case pauseAppspaceCmd:
		dsErr := s.AppspaceModel.Pause(appspaceID, true)
		if dsErr != nil {
			msg := "error pausing appspace " + dsErr.ToStandard().Error()
			fmt.Println(msg)
			m.SendError(msg)
		} else {
			m.SendOK()
		}
	case unpauseAppspaceCmd:
		dsErr := s.AppspaceModel.Pause(appspaceID, false)
		if dsErr != nil {
			msg := "error unpausing appspace " + dsErr.ToStandard().Error()
			fmt.Println(msg)
			m.SendError(msg)
		} else {
			m.SendOK()
		}
	default:
		m.SendError("service not found")
	}
}

const statusEventCmd = 11

func (s *DropserverDevServer) sendAppspaceStatusEvent(twine *twine.Twine, statusEvent domain.AppspaceStatusEvent) {
	bytes, err := json.Marshal(statusEvent)
	if err != nil {
		fmt.Println("sendAppspaceStatusEvent json Marshal Error: " + err.Error())
	}

	fmt.Println("Sending status event")

	_, err = twine.SendBlock(appspaceStatusService, statusEventCmd, bytes)
	if err != nil {
		fmt.Println("sendAppspaceStatusEvent SendBlock Error: " + err.Error())
	}
}

const routeHitEventCmd = 11

type RequestJSON struct {
	URL    string `json:"url"`
	Method string `json:"method"`
}
type RouteHitEventJSON struct {
	Timestamp   time.Time                   `json:"timestamp"`
	Request     RequestJSON                 `json:"request"`
	RouteConfig *domain.AppspaceRouteConfig `json:"route_config"` // this might be nil.OK?
}

func (s *DropserverDevServer) sendRouteEvent(twine *twine.Twine, routeEvent *domain.AppspaceRouteHitEvent) {
	send := RouteHitEventJSON{
		Timestamp: routeEvent.Timestamp,
		Request: RequestJSON{
			URL:    routeEvent.Request.URL.String(),
			Method: routeEvent.Request.Method},
		RouteConfig: routeEvent.RouteConfig}

	bytes, err := json.Marshal(send)
	if err != nil {
		// meh
		fmt.Println("sendRouteEvent json Marshal Error: " + err.Error())
	}

	fmt.Println("Sending route event")

	_, err = twine.SendBlock(routeEventService, routeHitEventCmd, bytes)
	if err != nil {
		//urhg
		fmt.Println("sendRouteEvent SendBlock Error: " + err.Error())
	}

}

// SetBaseData sets the base data that the server will return.
func (s *DropserverDevServer) SetBaseData(bd BaseData) {
	s.baseData = bd
}
