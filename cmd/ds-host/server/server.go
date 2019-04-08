package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// Server struct sets all parameters about the server
type Server struct {
	// admin routes, user routes, auth routes....
	AppspaceRoutes domain.RouteHandler

	Metrics domain.MetricsI
	Logger  domain.LogCLientI
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

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
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
	subdomains, ok := getSubdomains(req.Host)
	if !ok {
		http.Error(res, "Error getting subdomains from host string", http.StatusInternalServerError)
		s.Logger.Log(domain.DEBUG, map[string]string{}, "Error getting subdomains from host string: "+req.Host)
	} else {
		if len(subdomains) == 0 {
			// no subdomain. It's the site itself?
			http.Error(res, "Not found", http.StatusNotFound)
			ok = false
		}
	}

	if ok {
		topSub := subdomains[len(subdomains)-1]
		switch topSub {
		case "user":
			http.Error(res, "user not implemented", http.StatusNotImplemented)
		case "admin":
			http.Error(res, "admin not implemented", http.StatusNotImplemented)
		default:
			// first filter through blacklist of subdomains

			routeData := &domain.AppspaceRouteData{
				URLTail:    req.URL.Path,
				Subdomains: &subdomains}

			s.AppspaceRoutes.ServeHTTP(res, req, routeData)
		}
	}
}

func getSubdomains(host string) (subdomains []string, ok bool) {
	// here we need to know something about the configuration
	// how many domain levels to ignore?
	// also, consider that it might be a third-party domain.
	// so: explode host into pieces,
	// ..walk known host domain pieces [org, dropserver]

	ok = true

	rootHost := [2]string{"develop", "dropserver"} //TODO do not hard-code this, obviously
	numRoot := 2

	host = strings.Split(host, ":")[0] // in case host includes port
	hostPieces := strings.Split(host, ".")
	numPieces := len(hostPieces)

	if numPieces < numRoot {
		ok = false
	} else {
		for i, p := range rootHost {
			if hostPieces[numPieces-i-1] != p {
				ok = false
				break
			}
		}
	}

	if ok {
		subdomains = hostPieces[:numPieces-2]
	}

	return
}
