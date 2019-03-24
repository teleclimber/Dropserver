package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// Server struct sets all parameters about the server
type Server struct {
	// logger?
	// mux?
	SandboxManager domain.SandboxManagerI
	Metrics        domain.MetricsI
	HostAppSpace   *map[string]string
	AppSpaceApp    *map[string]string
	// this is going to get annoying with too many models and other things
	// Other things that are needed:
	// - record (logging and metrics)
	// - timetrack (ok that's metrics.)
}

// Start starts up the server so it listens for connections
func (s *Server) Start() { //return a server type
	// take a config please with:
	// - port
	// - ... all sorts of other things I'm sure.

	// Proxy:
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.handleRequest(w, r)
	})
	if err := http.ListenAndServe(":3000", nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// needed server graceful shutdown
// func (s *Server) Start() {
// }

/////////////////////////////////////////////////
// proxy
// This desperately needs to be refactored and broken into pieces
func (s *Server) handleRequest(oRes http.ResponseWriter, oReq *http.Request) {
	defer s.Metrics.HostHandleReq(time.Now())

	hostAppSpace := *s.HostAppSpace
	appSpaceApp := *s.AppSpaceApp

	host := strings.Split(oReq.Host, ":")[0] //in case the port was included in host
	appSpace, ok := hostAppSpace[host]
	if !ok {
		fmt.Println("app space not found for host", host) // Should log with request id, host
		oRes.WriteHeader(404)
		return
	}
	app, ok := appSpaceApp[appSpace]
	if !ok {
		fmt.Println("app not found for app space", appSpace) //request id, appspace
		oRes.WriteHeader(500)
		return
	}

	// then gotta run through auth and route...

	fmt.Println("in request handler", host, appSpace, app)

	sandboxChan := s.SandboxManager.GetForAppSpace(app, appSpace)
	sb := <-sandboxChan

	sbName := sb.GetName()
	sbAddress := sb.GetAddress()
	sbTransport := sb.GetTransport()

	//timetrack.Track(getTime, "getting sandbox "+appSpace+" c"+sbName)

	reqTaskCh := sb.TaskBegin()

	header := cloneHeader(oReq.Header)
	//header["ds-user-id"] = []string{"teleclimber"}
	header["app-space-script"] = []string{"hello.js"}
	header["app-space-fn"] = []string{"hello"}

	cReq, err := http.NewRequest(oReq.Method, sbAddress, oReq.Body)
	if err != nil {
		fmt.Println("http.NewRequest error", sbName, oReq.Method, sbAddress, err)
		os.Exit(1)
	}

	cReq.Header = header

	cRes, err := sbTransport.RoundTrip(cReq)
	if err != nil {
		fmt.Println("sb.Transport.RoundTrip(cReq) error", sbName, err)
		os.Exit(1)
	}

	// futz around with headers
	copyHeader(oRes.Header(), cRes.Header)

	oRes.WriteHeader(cRes.StatusCode)

	io.Copy(oRes, cRes.Body)

	cRes.Body.Close()

	reqTaskCh <- true

	sb.GetLogClient().Log(domain.INFO, map[string]string{
		"app-space": appSpace, "app": app},
		"Request handled")
}

// From https://golang.org/src/net/http/httputil/reverseproxy.go
// ..later we can pick and choose the headers and values we forward to the app
func cloneHeader(h http.Header) http.Header {
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
