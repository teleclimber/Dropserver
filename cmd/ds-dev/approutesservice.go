package main

import (
	"fmt"

	"github.com/teleclimber/twine-go/twine"
)

// remote service:
// const appRoutesService = 16

// outgoing commands:
const allRoutesData = 11

type AppRoutesService struct {
	AppFilesModel interface {
		ReadRoutes(locationKey string) ([]byte, error)
	} `checkinject:"required"`
	AppVersionEvents interface {
		Subscribe(chan<- string)
		Unsubscribe(chan<- string)
	} `checkinject:"required"`
}

//HandleMessage handles incoming twine message
func (s *AppRoutesService) HandleMessage(m twine.ReceivedMessageI) {
}

func (r *AppRoutesService) Start(t *twine.Twine) {
	appVersionCh := make(chan string)
	r.AppVersionEvents.Subscribe(appVersionCh)
	go func(ch chan string) {
		for state := range ch {
			if state == "ready" {
				r.sendRoutes(t)
			} else if state == "error" {
				r.sendNoRoutes(t)
			}
		}
	}(appVersionCh)

	r.sendRoutes(t)

	t.WaitClose()
	r.AppVersionEvents.Unsubscribe(appVersionCh)
}

func (r *AppRoutesService) sendRoutes(t *twine.Twine) {
	routesData, err := r.AppFilesModel.ReadRoutes("")
	if err != nil {
		panic(err)
	}
	if routesData == nil {
		r.sendNoRoutes(t)
		return
	}
	_, err = t.SendBlock(appRoutesService, allRoutesData, routesData)
	if err != nil {
		fmt.Println("sendRoutes SendBlock Error: " + err.Error())
	}
}

func (r *AppRoutesService) sendNoRoutes(t *twine.Twine) {
	_, err := t.SendBlock(appRoutesService, allRoutesData, []byte("[]"))
	if err != nil {
		fmt.Println("sendRoutes SendBlock Error: " + err.Error())
	}
}
