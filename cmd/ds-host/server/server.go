package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// Server struct sets all parameters about the server
type Server struct {
	// admin routes, user routes, auth routes....
	AppspaceRoutes domain.RouteHandler

	// TODO logger!
	Metrics domain.MetricsI
}

// Start starts up the server so it listens for connections
func (s *Server) Start() { //return a server type
	// take a config please with:
	// - port
	// - ... all sorts of other things I'm sure.

	// Proxy:
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	})
	if err := http.ListenAndServe(":3000", nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// needed server graceful shutdown
// func (s *Server) Start() {
// }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// switch on top level routes:
	// - admin
	// - user
	// - auth
	// - appspace...

	// Here I think we need to split url into subdomains
	// ..remove our root domain if present
	// ..check agains known subdomains (user, admin...) and route accordingly
	// ..check against our blacklist subdomains and drop accordingly
	// ..then pass remainder to appspace routes.

	// for now just create the RouteMeta and pass to appspace routes
	routeData := &domain.AppspaceRouteData{
		URLTail: r.URL.Path}

	s.AppspaceRoutes.ServeHTTP(w, r, routeData)
}
