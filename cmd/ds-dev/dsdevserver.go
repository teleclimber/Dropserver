package main

import (
	"encoding/binary"
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
const migrationStatusService = 14

// DropserverDevServer serves routes at dropserver-dev which control
// the handling of the app server
type DropserverDevServer struct {
	Config             *domain.RuntimeConfig
	DevAppModel        *DevAppModel
	DevAppspaceModel   *DevAppspaceModel
	AppspaceInfoModels interface {
		GetSchema(domain.AppspaceID) (int, error)
	}
	MigrationJobModel interface {
		Create(ownerID domain.UserID, appspaceID domain.AppspaceID, toVersion domain.Version, priority bool) (*domain.MigrationJob, error)
	}
	MigrationJobController interface {
		WakeUp()
	}
	MigrationJobsEvents interface {
		Subscribe(chan<- domain.MigrationStatusData)
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

	migrationEventsChan := make(chan domain.MigrationStatusData)
	s.MigrationJobsEvents.Subscribe(migrationEventsChan)
	go func() {
		for migrationEvent := range migrationEventsChan {
			go s.sendMigrationEvent(t, migrationEvent)
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
				go s.handleAppspaceCtrlMessage(m)
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
const migrateAppspaceCmd = 13

func (s *DropserverDevServer) handleAppspaceCtrlMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case pauseAppspaceCmd:
		dsErr := s.DevAppspaceModel.Pause(appspaceID, true)
		if dsErr != nil {
			msg := "error pausing appspace " + dsErr.ToStandard().Error()
			fmt.Println(msg)
			m.SendError(msg)
		} else {
			m.SendOK()
		}
	case unpauseAppspaceCmd:
		dsErr := s.DevAppspaceModel.Pause(appspaceID, false)
		if dsErr != nil {
			msg := "error unpausing appspace " + dsErr.ToStandard().Error()
			fmt.Println(msg)
			m.SendError(msg)
		} else {
			m.SendOK()
		}
	case migrateAppspaceCmd:
		// first read the payload
		migrateTo := int(binary.BigEndian.Uint16(m.Payload()))

		// If migrating up, create a app version with higher version, and to-schema,
		// *or* create a lower version with current schema, and re-create the main app-version with new schema?
		// but maybe this takes place automatically on observe of app code?

		// If migrating down, then create dummy version with lower version, to-schema.
		// Location key and rest is immaterial as it souldn't get used.

		appspaceSchema, err := s.AppspaceInfoModels.GetSchema(appspaceID)
		if err != nil {
			fmt.Println("failed to get appspace schema: " + err.Error())
			m.SendError("failed to get appspace schema")
			return
		}

		// Assume we are migrating down
		if migrateTo < appspaceSchema {
			s.DevAppModel.ToVer.Version = domain.Version("0.0.1")
			s.DevAppModel.ToVer.Schema = migrateTo
			s.MigrationJobModel.Create(ownerID, appspaceID, s.DevAppModel.ToVer.Version, true)
			s.MigrationJobController.WakeUp()
			m.SendOK()
		} else if migrateTo > appspaceSchema {
			s.DevAppModel.ToVer.Version = domain.Version("100.0.0")
			s.DevAppModel.ToVer.Schema = migrateTo
			s.MigrationJobModel.Create(ownerID, appspaceID, s.DevAppModel.ToVer.Version, true)
			s.MigrationJobController.WakeUp()
			m.SendOK()
		} else {
			m.SendError("migrate to scehma same as current appspace schema")
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

const migrationStatusEventCmd = 11

func (s *DropserverDevServer) sendMigrationEvent(twine *twine.Twine, migrationEvent domain.MigrationStatusData) {
	bytes, err := json.Marshal(migrationEvent)
	if err != nil {
		fmt.Println("sendMigrationEvent json Marshal Error: " + err.Error())
	}

	fmt.Println("Sending migration job event")

	_, err = twine.SendBlock(migrationStatusService, migrationStatusEventCmd, bytes)
	if err != nil {
		fmt.Println("sendMigrationEvent SendBlock Error: " + err.Error())
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
