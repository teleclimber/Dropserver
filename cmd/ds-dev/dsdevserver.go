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
const baseDataService = 13
const migrationStatusService = 14
const appspaceLogService = 15

// DropserverDevServer serves routes at dropserver-dev which control
// the handling of the app server
type DropserverDevServer struct {
	Config        *domain.RuntimeConfig
	DevAppModel   *DevAppModel
	AppFilesModel interface {
		ReadMeta(locationKey string) (*domain.AppFilesMetadata, domain.Error)
	}
	AppspaceFiles interface {
		Reset()
	}
	DevAppspaceModel *DevAppspaceModel
	AppspaceMetaDB   interface {
		CloseConn(domain.AppspaceID) error
	}
	AppspaceDB interface {
		CloseAppspace(domain.AppspaceID)
	}
	AppspaceInfoModels interface {
		GetSchema(domain.AppspaceID) (int, error)
	}
	DevSandboxManager interface {
		StopAppspace(domain.AppspaceID)
		SetInspect(bool)
	}
	MigrationJobModel interface {
		Create(ownerID domain.UserID, appspaceID domain.AppspaceID, toVersion domain.Version, priority bool) (*domain.MigrationJob, error)
	}
	MigrationJobController interface {
		WakeUp()
	}
	DevSandboxMaker interface {
		SetInspect(bool)
	}
	AppspaceStatus interface {
		SetTempPause(domain.AppspaceID, bool)
		WaitStopped(domain.AppspaceID)
	}
	AppVersionEvents interface {
		Subscribe(chan<- domain.AppID)
	}
	MigrationJobsEvents interface {
		Subscribe(chan<- domain.MigrationStatusData)
	}
	AppspaceLogEvents interface {
		Subscribe(domain.AppspaceID, chan<- domain.AppspaceLogEvent)
		Unsubscribe(domain.AppspaceID, chan<- domain.AppspaceLogEvent)
	}
	AppspaceStatusEvents interface {
		Subscribe(domain.AppspaceID, chan<- domain.AppspaceStatusEvent)
		Unsubscribe(domain.AppspaceID, chan<- domain.AppspaceStatusEvent)
	}
	RouteEvents interface {
		Subscribe(ch chan<- *domain.AppspaceRouteHitEvent)
		Unsubscribe(ch chan<- *domain.AppspaceRouteHitEvent)
	}

	appPath      string
	appspacePath string
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

	case "base-data":
		appspaceSchema, err := s.AppspaceInfoModels.GetSchema(appspaceID)
		if err != nil {
			fmt.Println("failed to get appspace schema: " + err.Error())
		}

		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)

		baseData := BaseData{
			AppPath:      s.appPath, // these don't change
			AppspacePath: s.appspacePath,

			AppspaceSchema: appspaceSchema} // this is appspace-related, and should be sent via a different command? Like appspace status event?

		json.NewEncoder(res).Encode(baseData)

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
	appVersionEvent := make(chan domain.AppID)
	s.AppVersionEvents.Subscribe(appVersionEvent)
	go func() {
		for range appVersionEvent {
			go s.sendAppData(t)
		}
	}()
	// push initial app data:
	s.sendAppData(t)

	appspaceStatusChan := make(chan domain.AppspaceStatusEvent)
	s.AppspaceStatusEvents.Subscribe(appspaceID, appspaceStatusChan)
	go func() {
		for statusEvent := range appspaceStatusChan {
			go s.sendAppspaceStatusEvent(t, statusEvent)
		}
	}()

	appspaceLogEventChan := make(chan domain.AppspaceLogEvent)
	s.AppspaceLogEvents.Subscribe(appspaceID, appspaceLogEventChan)
	go func() {
		for logEvent := range appspaceLogEventChan {
			go s.sendAppspaceLogEvent(t, logEvent)
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

		s.AppspaceLogEvents.Unsubscribe(appspaceID, appspaceLogEventChan)
		close(appspaceLogEventChan)

		s.RouteEvents.Unsubscribe(routeEventsChan)
		close(routeEventsChan)

		fmt.Println("unsubscribed")
	}()
}

const pauseAppspaceCmd = 11
const unpauseAppspaceCmd = 12
const migrateAppspaceCmd = 13
const setInspect = 14
const stopSandbox = 15
const importAndMigrate = 16

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
		migrateTo := int(binary.BigEndian.Uint16(m.Payload()))
		err := s.migrate(migrateTo)
		if err != nil {
			m.SendError("error migrating: " + err.Error())
		} else {
			m.SendOK()
		}

	case setInspect: // this should really be inspect for everything.
		inspect := true
		p := m.Payload()
		if p[0] == 0x00 {
			inspect = false
		}
		s.DevSandboxMaker.SetInspect(inspect)
		s.DevSandboxManager.StopAppspace(appspaceID)
		s.DevSandboxManager.SetInspect(inspect)
		m.SendOK()
	case stopSandbox:
		// force-kill for unruly scripts?
		s.DevSandboxManager.StopAppspace(appspaceID)
		m.SendOK()
	case importAndMigrate:
		schema, err := s.AppspaceInfoModels.GetSchema(appspaceID)
		if err != nil {
			m.SendError(err.Error())
			return
		}
		fmt.Println(schema)

		m.RefSendBlock(11, []byte("Stopping..."))
		s.AppspaceStatus.SetTempPause(appspaceID, true)
		s.AppspaceStatus.WaitStopped(appspaceID)
		s.DevSandboxManager.StopAppspace(appspaceID)

		m.RefSendBlock(11, []byte("Closing..."))
		err = s.AppspaceMetaDB.CloseConn(appspaceID)
		if err != nil {
			m.SendError(err.Error())
			return
		}

		s.AppspaceDB.CloseAppspace(appspaceID)

		m.RefSendBlock(11, []byte("Copying Files..."))
		s.AppspaceFiles.Reset()

		// run migration to latest
		m.RefSendBlock(11, []byte("Migrating..."))
		err = s.migrate(schema)
		if err != nil {
			m.SendError("error migrating: " + err.Error())
			return
		}

		s.AppspaceStatus.SetTempPause(appspaceID, false)
		m.SendOK()
	default:
		m.SendError("service not found")
	}
}

// If migrating up, create a app version with higher version, and to-schema,
// *or* create a lower version with current schema, and re-create the main app-version with new schema?
// but maybe this takes place automatically on observe of app code?

// If migrating down, then create dummy version with lower version, to-schema.
// Location key and rest is immaterial as it souldn't get used.
func (s *DropserverDevServer) migrate(migrateTo int) error {
	appspaceSchema, err := s.AppspaceInfoModels.GetSchema(appspaceID)
	if err != nil {
		return err
	}
	if migrateTo < appspaceSchema {
		s.DevAppModel.ToVer.Version = domain.Version("0.0.1")
		s.DevAppModel.ToVer.Schema = migrateTo
		s.MigrationJobModel.Create(ownerID, appspaceID, s.DevAppModel.ToVer.Version, true)
		s.MigrationJobController.WakeUp()
	} else if migrateTo > appspaceSchema {
		s.DevAppModel.ToVer.Version = domain.Version("100.0.0")
		s.DevAppModel.ToVer.Schema = migrateTo
		s.MigrationJobModel.Create(ownerID, appspaceID, s.DevAppModel.ToVer.Version, true)
		s.MigrationJobController.WakeUp()
	} else {
		return fmt.Errorf("migrate to scehma same as current appspace schema: to: %v, current: %v", migrateTo, appspaceSchema)
	}
	return nil
}

type AppData struct {
	AppName       string `json:"app_name"`
	AppVersion    string `json:"app_version"`
	AppMigrations []int  `json:"app_migrations"`
	AppSchema     int    `json:"app_version_schema"`
}

// base data service:
const statusEventCmd = 11
const appDataCmd = 12

func (s *DropserverDevServer) sendAppData(twine *twine.Twine) {
	appFilesMeta, dsErr := s.AppFilesModel.ReadMeta("")
	if dsErr != nil {
		panic(dsErr.ToStandard().Error())
	}
	appData := AppData{
		AppName:       appFilesMeta.AppName, // these are app-related, should be re-sent when app is changed
		AppVersion:    string(appFilesMeta.AppVersion),
		AppSchema:     appFilesMeta.SchemaVersion,
		AppMigrations: appFilesMeta.Migrations,
	}
	bytes, err := json.Marshal(appData)
	if err != nil {
		fmt.Println("sendAppData json Marshal Error: " + err.Error())
	}

	fmt.Println("Sending app data")

	_, err = twine.SendBlock(baseDataService, appDataCmd, bytes)
	if err != nil {
		fmt.Println("sendAppData SendBlock Error: " + err.Error())
	}
}

func (s *DropserverDevServer) sendAppspaceStatusEvent(twine *twine.Twine, statusEvent domain.AppspaceStatusEvent) {
	bytes, err := json.Marshal(statusEvent)
	if err != nil {
		fmt.Println("sendAppspaceStatusEvent json Marshal Error: " + err.Error())
	}

	fmt.Println("Sending status event")

	_, err = twine.SendBlock(baseDataService, statusEventCmd, bytes)
	if err != nil {
		fmt.Println("sendAppspaceStatusEvent SendBlock Error: " + err.Error())
	}
}

const sandboxLogEventCmd = 11

func (s *DropserverDevServer) sendAppspaceLogEvent(twine *twine.Twine, statusEvent domain.AppspaceLogEvent) {
	bytes, err := json.Marshal(statusEvent)
	if err != nil {
		fmt.Println("sendAppspaceLogEvent json Marshal Error: " + err.Error())
	}

	_, err = twine.SendBlock(appspaceLogService, sandboxLogEventCmd, bytes)
	if err != nil {
		fmt.Println("sendAppspaceLogEvent SendBlock Error: " + err.Error())
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

// SetPaths sets the paths of the ppa nd appspace so it can be reported on the frontend
func (s *DropserverDevServer) SetPaths(appPath, appspacePath string) {
	s.appPath = appPath
	s.appspacePath = appspacePath
}
