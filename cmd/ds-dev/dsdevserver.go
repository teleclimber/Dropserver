package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// DropserverDevServer serves routes at dropserver-dev which control
// the handling of the app server
type DropserverDevServer struct {
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

	head, _ := shiftpath.ShiftPath(req.URL.Path)
	switch head {
	case "":
		// serve ds-dev.html
		http.Error(res, "not done", http.StatusNotImplemented)
	case "base-data": // temporary
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		json.NewEncoder(res).Encode(s.baseData)

	default:
		// file serve the frontend dist dir
		http.Error(res, "dropserver-dev route not found", http.StatusNotFound)
	}

}

// SetBaseData sets the base data that the server will return.
func (s *DropserverDevServer) SetBaseData(bd BaseData) {
	s.baseData = bd
}
