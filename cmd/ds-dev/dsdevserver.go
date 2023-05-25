package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/twine-go/twine"
)

const routeEventService = 11      // that's hits on a route
const appspaceControlService = 12 // incmoing? For appspace control (pause, migrate)
const appDataService = 13
const migrationJobService = 14
const appspaceLogService = 15
const appRoutesService = 16      // keeps a live list of app routes
const userControlService = 17    //incoming / outgoing
const appspaceStatusService = 18 // in/out? or just out? I think just out.
const sandboxControlService = 19 // in/out, inspect, kill sandbox

type twineService interface {
	Start(*twine.Twine)
	HandleMessage(twine.ReceivedMessageI)
}

// DropserverDevServer serves routes at dropserver-dev which control
// the handling of the app server
type DropserverDevServer struct {
	Config      *domain.RuntimeConfig `checkinject:"required"`
	DevAppModel *DevAppModel          `checkinject:"required"`
	AppGetter   interface {
		ValidateMigrationSteps(migrations []domain.MigrationStep) ([]int, error)
	} `checkinject:"required"`
	AppFilesModel interface {
		GetAppIconPath(string) string
	} `checkinject:"required"`
	AppspaceFiles interface {
		Reset()
	} `checkinject:"required"`
	AppspaceMetaDB interface {
		CloseConn(domain.AppspaceID) error
	} `checkinject:"required"`
	AppspaceDB interface {
		CloseAppspace(domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceLogger interface {
		Close(domain.AppspaceID)
		Open(domain.AppspaceID) domain.LoggerI
	} `checkinject:"required"`
	DevSandboxManager interface {
		StopAppspace(domain.AppspaceID)
	} `checkinject:"required"`
	MigrationJobModel interface {
		CreateFromSchema(migrateTo int) error
	} `checkinject:"required"`
	MigrationJobController interface {
		WakeUp()
	} `checkinject:"required"`
	PauseAppspace interface {
		Pause(appspaceID domain.AppspaceID, pause bool) error
	} `checkinject:"required"`
	AppspaceStatus interface {
		WaitTempPaused(domain.AppspaceID, string) chan struct{}
	} `checkinject:"required"`

	// Dev Services:
	SandboxControlService twineService `checkinject:"required"`
	AppspaceStatusService twineService `checkinject:"required"`
	AppMetaService        twineService `checkinject:"required"`
	AppRoutesService      twineService `checkinject:"required"`
	UserService           twineService `checkinject:"required"`
	RouteHitService       twineService `checkinject:"required"`

	// Services:
	MigrationJobService domain.TwineService2 `checkinject:"required"`
	AppspaceLogService  domain.TwineService2 `checkinject:"required"`

	appPath      string // this is app original location, not where the app files necessarily are.
	appspacePath string
}

func (s *DropserverDevServer) GetBaseData(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	baseData := BaseData{
		AppPath:            s.appPath, // these don't change
		AppspacePath:       s.appspacePath,
		AppspaceWorkingDir: s.Config.Exec.AppspacesPath}

	json.NewEncoder(res).Encode(baseData)

}
func (s *DropserverDevServer) GetAppIcon(res http.ResponseWriter, req *http.Request) {
	p := s.AppFilesModel.GetAppIconPath("")
	if p == "" {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	http.ServeFile(res, req, p)
}

func (s *DropserverDevServer) StartLivedata(res http.ResponseWriter, req *http.Request) {
	t, err := twine.NewWebsocketServer(res, req)
	if err != nil {
		fmt.Println("twine.NewWebsocketServer(res, req) " + err.Error() + " origin: " + req.Header.Get("origin"))
		//http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	_, ok := <-t.ReadyChan
	if !ok {
		fmt.Println("failed to start Twine")
	}

	go s.SandboxControlService.Start(t)
	go s.AppspaceStatusService.Start(t)
	go s.AppMetaService.Start(t)
	go s.AppRoutesService.Start(t)
	go s.UserService.Start(t)
	go s.RouteHitService.Start(t)

	migrationJobTwine := s.MigrationJobService.Start(ownerID, t)
	appspaceLogTwine := s.AppspaceLogService.Start(ownerID, t)

	// need to receive messages too
	go func() {
		for m := range t.MessageChan {
			switch m.ServiceID() {
			case sandboxControlService:
				go s.SandboxControlService.HandleMessage(m)
			case appspaceControlService:
				go s.handleAppspaceCtrlMessage(m)
			case userControlService:
				go s.UserService.HandleMessage(m)
			case migrationJobService:
				go migrationJobTwine.HandleMessage(m)
			case appspaceLogService:
				go appspaceLogTwine.HandleMessage(m)
			case appRoutesService:
				go s.AppRoutesService.HandleMessage(m)
			default:
				fmt.Println("Service not found ")
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
		err := s.PauseAppspace.Pause(appspaceID, true)
		if err != nil {
			msg := "error pausing appspace " + err.Error()
			fmt.Println(msg)
			m.SendError(msg)
		} else {
			m.SendOK()
		}
	case unpauseAppspaceCmd:
		err := s.PauseAppspace.Pause(appspaceID, false)
		if err != nil {
			msg := "error unpausing appspace " + err.Error()
			fmt.Println(msg)
			m.SendError(msg)
		} else {
			m.SendOK()
		}
	case migrateAppspaceCmd:
		migrateTo := int(binary.BigEndian.Uint16(m.Payload()))
		err := s.MigrationJobModel.CreateFromSchema(migrateTo)
		if err != nil {
			m.SendError("error creating migration job: " + err.Error())
		} else {
			m.SendOK()
		}
	case importAndMigrate:
		m.RefSendBlock(11, []byte("Stopping..."))
		tempPauseCh := s.AppspaceStatus.WaitTempPaused(appspaceID, "import & migrate")
		s.DevSandboxManager.StopAppspace(appspaceID)

		m.RefSendBlock(11, []byte("Closing..."))
		err := s.AppspaceMetaDB.CloseConn(appspaceID)
		if err != nil {
			m.SendError(err.Error())
			return
		}

		s.AppspaceDB.CloseAppspace(appspaceID)
		s.AppspaceLogger.Close(appspaceID)

		m.RefSendBlock(11, []byte("Copying Files..."))
		s.AppspaceFiles.Reset()

		close(tempPauseCh) // close here so migration system can obtain a pause

		m.RefSendBlock(11, []byte("Migrating..."))
		err = s.MigrationJobModel.CreateFromSchema(s.DevAppModel.Ver.Schema) // migrate the appspace to the app's schema.
		if err != nil && err != errNoMigrationNeeded {
			m.SendError("error migrating: " + err.Error())
			return
		}

		m.SendOK()

		// Reopen log after the work is complete so tahtthe frontend can get current log view.
		// Is this really necessary? -> maybe, since we don't have locks on apps, need to explicitly open log?
		s.AppspaceLogger.Open(appspaceID)

	default:
		m.SendError("service not found")
	}
}

// SetPaths sets the paths of the ppa nd appspace so it can be reported on the frontend
func (s *DropserverDevServer) SetPaths(appPath, appspacePath string) {
	s.appPath = appPath
	s.appspacePath = appspacePath
}
