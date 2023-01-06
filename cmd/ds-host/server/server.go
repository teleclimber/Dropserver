package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/getcleanhost"
)

// Server struct sets all parameters about the server
type Server struct {
	Config             *domain.RuntimeConfig `checkinject:"required"`
	CertificateManager interface {
		GetTLSConfig() *tls.Config
		GetHTTPChallengeHandler(handler http.Handler) http.Handler
	} `checkinject:"required"`
	// admin routes, user routes, auth routes....
	UserRoutes     http.Handler `checkinject:"required"`
	AppspaceRouter http.Handler `checkinject:"required"`

	server              *http.Server
	httpChallengeServer *http.Server
}

// Start up the server so it listens for connections
func (s *Server) Start() { //return a server type
	cfg := s.Config.Server

	HTTPAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	HTTPSAddr := fmt.Sprintf(":%d", cfg.TLSPort)

	u := fmt.Sprintf("%s://%s%s", s.Config.ExternalAccess.Scheme, s.Config.Exec.UserRoutesDomain, s.Config.Exec.PortString)
	s.getLogger("Start()").Log("Log in at: " + u)

	s.server = &http.Server{
		Handler: s}

	if s.Config.Server.NoTLS {
		// start plain http server and you're done.
		s.server.Addr = HTTPAddr
		go func() {
			err := s.server.ListenAndServe()
			if err != nil {
				s.getLogger("ListenAndServe()").Error(err)
			}
		}()
	} else {
		// main server will be TLS
		s.server.Addr = HTTPSAddr
		cert := ""
		key := ""
		if s.Config.ManageTLSCertificates.Enable {
			s.server.TLSConfig = s.CertificateManager.GetTLSConfig()
		} else if s.Config.Server.SslCert != "" && s.Config.Server.SslKey != "" {
			cert = s.Config.Server.SslCert
			key = s.Config.Server.SslKey
		} else {
			// should never happen because config should be checked such that it does not happen
			panic("no valid configuration to start the server")
		}
		go func() {
			err := s.server.ListenAndServeTLS(cert, key)
			if err != nil {
				s.getLogger("ListenAndServeTLS()").Error(err)
			}
		}()

		if s.Config.ManageTLSCertificates.Enable { // For now only start a plain http server to answer http challnges
			var handler http.Handler
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// this should optionally be a redirect to https
				w.Write([]byte("Hello UN-encrypted world!"))
			})
			handler = s.CertificateManager.GetHTTPChallengeHandler(handler)
			s.httpChallengeServer = &http.Server{
				Addr:    HTTPAddr,
				Handler: handler}
			go func() {
				err := s.httpChallengeServer.ListenAndServe()
				if err != nil {
					s.getLogger("Plain HTTP ListenAndServe()").Error(err)
				}
			}()
		}
	}
}

func (s *Server) Shutdown() {
	if s.server == nil {
		return
	}

	l := s.getLogger("Shutdown()")
	l.Log("stopping main server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		s.getLogger("Cert Managed plain HTTP ListenAndServe()").Error(err)
	}

	if s.httpChallengeServer != nil {
		l.Log("stopping http challenge server")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := s.httpChallengeServer.Shutdown(ctx)
		if err != nil {
			panic(err) // for now.// log it
		}
	}

	l.Log("servers stopped")
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	host, err := getcleanhost.GetCleanHost(req.Host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// This is going to need to account for dropid domains
	// which may double as appspace domains and user domains?

	switch host {
	case s.Config.Exec.UserRoutesDomain:
		s.UserRoutes.ServeHTTP(res, req)
	default:
		// It's an appspace subdomain
		// first filter through blacklist of domains
		s.AppspaceRouter.ServeHTTP(res, req)
	}
}

func (s *Server) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("Server")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
