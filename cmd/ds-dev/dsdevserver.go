package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/markbates/pkger"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/twine"
)

const routeEventService = 11      // that's hits on a route
const appspaceControlService = 12 // incmoing? For appspace control (pause, migrate)
const baseDataService = 13
const migrationJobService = 14
const appspaceLogService = 15
const appspaceRouteService = 16 // keeps a live list of appspace routes from appspace meta db
const userControlService = 17   //incoming / outgoing

type twineService interface {
	Start(*twine.Twine)
	HandleMessage(twine.ReceivedMessageI)
}

// DropserverDevServer serves routes at dropserver-dev which control
// the handling of the app server
type DropserverDevServer struct {
	Config        *domain.RuntimeConfig
	DevAppModel   *DevAppModel
	AppFilesModel interface {
		ReadMeta(locationKey string) (*domain.AppFilesMetadata, error)
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
	AppspaceLogEvents interface {
		Subscribe(domain.AppspaceID, chan<- domain.AppspaceLogEvent)
		Unsubscribe(domain.AppspaceID, chan<- domain.AppspaceLogEvent)
	}
	AppspaceStatusEvents interface {
		Subscribe(domain.AppspaceID, chan<- domain.AppspaceStatusEvent)
		Unsubscribe(domain.AppspaceID, chan<- domain.AppspaceStatusEvent)
	}

	// Dev Services:
	AppMetaService  twineService
	RoutesService   twineService
	UserService     twineService
	RouteHitService twineService

	// Services:
	MigrationJobService domain.TwineService

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

	staticHandler := http.FileServer(pkger.Dir("/frontend-ds-dev/dist/"))

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

	case "appspacelogin":
		s.appspaceLogin(res, req)

	default:
		// file serve the frontend dist dir
		//http.Error(res, "dropserver-dev route not found", http.StatusNotFound)
		staticHandler.ServeHTTP(res, req)
	}

}

func (s *DropserverDevServer) appspaceLogin(res http.ResponseWriter, req *http.Request) {
	// Here we can get the token, retrieve the corresponding data,

	res.Write([]byte(fmt.Sprintf("try again:")))
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

	go s.AppMetaService.Start(t)
	go s.RoutesService.Start(t)
	go s.UserService.Start(t)
	go s.RouteHitService.Start(t)

	go s.MigrationJobService.Start(ownerID, t)

	appspaceLogEventChan := make(chan domain.AppspaceLogEvent)
	s.AppspaceLogEvents.Subscribe(appspaceID, appspaceLogEventChan)
	go func() {
		for logEvent := range appspaceLogEventChan {
			go s.sendAppspaceLogEvent(t, logEvent)
		}
	}()

	// need to receive messages too
	go func() {
		for m := range t.MessageChan {
			switch m.ServiceID() {
			case appspaceControlService:
				go s.handleAppspaceCtrlMessage(m)
			case userControlService:
				go s.UserService.HandleMessage(m)
			case migrationJobService:
				go s.MigrationJobService.HandleMessage(m)
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
		err := s.DevAppspaceModel.Pause(appspaceID, true)
		if err != nil {
			msg := "error pausing appspace " + err.Error()
			fmt.Println(msg)
			m.SendError(msg)
		} else {
			m.SendOK()
		}
	case unpauseAppspaceCmd:
		err := s.DevAppspaceModel.Pause(appspaceID, false)
		if err != nil {
			msg := "error unpausing appspace " + err.Error()
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
		m.RefSendBlock(11, []byte("Stopping..."))
		s.AppspaceStatus.SetTempPause(appspaceID, true)
		s.AppspaceStatus.WaitStopped(appspaceID)
		s.DevSandboxManager.StopAppspace(appspaceID)

		m.RefSendBlock(11, []byte("Closing..."))
		err := s.AppspaceMetaDB.CloseConn(appspaceID)
		if err != nil {
			m.SendError(err.Error())
			return
		}

		s.AppspaceDB.CloseAppspace(appspaceID)

		m.RefSendBlock(11, []byte("Copying Files..."))
		s.AppspaceFiles.Reset()

		// run migration to latest
		//  First get highest migration level
		appFilesMeta, err := s.AppFilesModel.ReadMeta("")
		if err != nil {
			panic(err)
		}
		schema := 0
		if len(appFilesMeta.Migrations) > 0 {
			ms := appFilesMeta.Migrations
			schema = ms[len(ms)-1]
		}
		m.RefSendBlock(11, []byte("Migrating..."))
		err = s.migrate(schema)
		if err != nil && err != errNoMigrationNeeded {
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

var errNoMigrationNeeded = errors.New("No migration needed")

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
		return errNoMigrationNeeded
	}
	return nil
}

// base data service:
const statusEventCmd = 11

func (s *DropserverDevServer) sendAppspaceStatusEvent(twine *twine.Twine, statusEvent domain.AppspaceStatusEvent) {
	bytes, err := json.Marshal(statusEvent)
	if err != nil {
		fmt.Println("sendAppspaceStatusEvent json Marshal Error: " + err.Error())
	}

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

// SetPaths sets the paths of the ppa nd appspace so it can be reported on the frontend
func (s *DropserverDevServer) SetPaths(appPath, appspacePath string) {
	s.appPath = appPath
	s.appspacePath = appspacePath
}
