package main

import (
	"encoding/json"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

// remote service:
// const appRoutesService = 16

// remote commands:
const allRoutesData = 11
const routesError = 12
const routesDirty = 13
const loadingRoutes = 14

// local commands:
const loadRoutes = 11
const setAutoLoad = 12

// AppRoutesService calls sandbox to get app routes and reutrns them to ds-dev frontend.
// - listen to app file changes and mark routes as potentially dirty
// - have a "reload routes automatically" flag
// - have a manual "load routes now" command
// - if any errors in routes, these are preserved here and relayed to ds-dev ffronted

type AppRoutesService struct {
	AppFilesModel interface {
		ReadRoutes(locationKey string) ([]byte, error)
		WriteRoutes(locationKey string, routesData []byte) error
	}
	AppGetter interface {
		GetRouterData(loc string) ([]domain.V0AppRoute, error)
	}
	AppVersionEvents interface {
		Subscribe(chan<- domain.AppID)
		Unsubscribe(chan<- domain.AppID)
	}

	routeError error

	autoReload bool

	twine *twine.Twine //this would be fine if we created a new service for each client. But we don't.
}

func (r *AppRoutesService) Start(t *twine.Twine) {
	r.twine = t
	appChangeCh := make(chan domain.AppID)
	r.AppVersionEvents.Subscribe(appChangeCh)
	go func() {
		for range appChangeCh {
			go r.appChanged()
		}
	}()

	// push current routes:
	//s.sendAppspaceRoutes(t)

	// Wait for twine to close and shut it all down.
	t.WaitClose()

	fmt.Println("closing app route service")

	r.AppVersionEvents.Unsubscribe(appChangeCh)
	close(appChangeCh)
}

func (r *AppRoutesService) appChanged() {
	r.sendDirty()

	if r.autoReload {
		r.loadRoutes()
	}
}

func (r *AppRoutesService) sendDirty() {
	_, err := r.twine.SendBlock(appRoutesService, routesDirty, nil)
	if err != nil {
		fmt.Println("AppRoutesService sendDirty SendBlock Error: " + err.Error())
	}
}

// HandleMessage handles incoming mesages for this service.
func (r *AppRoutesService) HandleMessage(m twine.ReceivedMessageI) {
	switch m.CommandID() {
	case loadRoutes:
		r.handleLoadRoutes(m)
	default:
		m.SendError(fmt.Sprintf("command not recognized %v", m.CommandID()))
	}
}

func (r *AppRoutesService) sendRoutes() {
	if r.routeError != nil {
		_, err := r.twine.SendBlock(appRoutesService, routesError, []byte(r.routeError.Error()))
		if err != nil {
			fmt.Println("sendRoutes routesError SendBlock Error: " + err.Error())
		}
	} else {
		routesData, _ := r.AppFilesModel.ReadRoutes("")
		_, err := r.twine.SendBlock(appRoutesService, allRoutesData, routesData)
		if err != nil {
			fmt.Println("sendRoutes SendBlock Error: " + err.Error())
		}
	}
}

func (r *AppRoutesService) handleLoadRoutes(m twine.ReceivedMessageI) {
	m.SendOK()

	r.loadRoutes()

	r.sendRoutes()
}

func (r *AppRoutesService) loadRoutes() {
	r.routeError = nil

	// later send "loading routes"
	routes, err := r.AppGetter.GetRouterData("")
	if err != nil {
		r.routeError = err
		r.AppFilesModel.WriteRoutes("", nil)
		// like, what do we do?
		fmt.Println("Error in routes: " + err.Error())
		return
	}
	fmt.Println("got routes!", routes)
	routesData, err := json.Marshal(routes)
	if err != nil {
		fmt.Println("Error json marshalling routes: " + err.Error())
		return
	}
	r.AppFilesModel.WriteRoutes("", routesData)
}
