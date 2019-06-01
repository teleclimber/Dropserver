package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// Server struct sets all parameters about the server
type Server struct {
	Config *domain.RuntimeConfig

	// admin routes, user routes, auth routes....
	UserRoutes     domain.RouteHandler
	AppspaceRoutes domain.RouteHandler

	Metrics domain.MetricsI
	Logger  domain.LogCLientI

	rootDomainPieces    []string
	publicStaticHandler http.Handler
}

// Start starts up the server so it listens for connections
func (s *Server) Start() { //return a server type
	s.init()

	cfg := s.Config.Server

	// Proxy:
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	})

	addr := ":" + strconv.FormatInt(int64(cfg.Port), 10)

	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (s *Server) init() {
	host := strings.ToLower(s.Config.Server.Host)
	s.rootDomainPieces = strings.Split(host, ".")
	reverse(s.rootDomainPieces)

	// static server
	s.publicStaticHandler = http.FileServer(http.Dir(s.Config.PublicStaticDir))
}

// needed server graceful shutdown
// func (s *Server) Start() {
// }

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// Can we have some application-global iddlewares at work here?
	// Like CSRF? -> Any POST, PUT, PATCH, .. gets checked for a CSRF token?
	// I guess the middleware would have same signature as others, and include reouteData
	//

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
	subdomains, ok := getSubdomains(req.Host, s.rootDomainPieces)
	if !ok {
		http.Error(res, "Error getting subdomains from host string", http.StatusInternalServerError)
		s.Logger.Log(domain.DEBUG, map[string]string{}, "Error getting subdomains from host string: "+req.Host)
		return
	} else if len(subdomains) == 0 {
		// no subdomain. It's the site itself?
		http.Error(res, "Not found", http.StatusNotFound)
		return
	}

	topSub := subdomains[len(subdomains)-1]
	switch topSub {
	case "static":
		s.publicStaticHandler.ServeHTTP(res, req)
	case "user":
		routeData := &domain.AppspaceRouteData{
			URLTail:    req.URL.Path,
			Subdomains: &subdomains}

		s.UserRoutes.ServeHTTP(res, req, routeData)

	case "admin":
		http.Error(res, "admin not implemented", http.StatusNotImplemented)
	default:
		// first filter through blacklist of subdomains
		// ..though probably do that in appspace routes handler, not here.

		routeData := &domain.AppspaceRouteData{
			URLTail:    req.URL.Path,
			Subdomains: &subdomains}

		s.AppspaceRoutes.ServeHTTP(res, req, routeData)
	}
}

func getSubdomains(reqHost string, rootDomainPieces []string) (subdomains []string, ok bool) {
	// also, consider that it might be a third-party domain.
	// so: explode host into pieces,
	// ..walk known host domain pieces [org, dropserver]

	ok = true

	numRoot := len(rootDomainPieces)

	reqHost = strings.ToLower(reqHost)

	reqHost = strings.Split(reqHost, ":")[0] // in case host includes port
	hostPieces := strings.Split(reqHost, ".")
	numPieces := len(hostPieces)

	if numPieces < numRoot {
		ok = false
	} else {
		for i, p := range rootDomainPieces {
			if hostPieces[numPieces-i-1] != p {
				ok = false
				break
			}
		}
	}

	if ok {
		subdomains = hostPieces[:numPieces-numRoot]
	}

	return
}

// util
// from https://stackoverflow.com/questions/34816489/reverse-slice-of-strings
func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}
