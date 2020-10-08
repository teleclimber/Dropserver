package sandboxproxy

import (
	"io"
	"net/http"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// SandboxProxy holds other structs for the proxy
type SandboxProxy struct {
	SandboxManager interface {
		GetForAppSpace(*domain.AppVersion, *domain.Appspace) chan domain.SandboxI
	} // not needed at server level
	Metrics domain.MetricsI
}

// ServeHTTP forwards the request to a sandbox
// Could still see splitting this function in two.
func (s *SandboxProxy) ServeHTTP(oRes http.ResponseWriter, oReq *http.Request, routeData *domain.AppspaceRouteData) {
	defer s.Metrics.HostHandleReq(time.Now())

	// The responsibiility for knowing whether an appspace is ready or not, is upstream (in appspaceroutes)

	sandboxChan := s.SandboxManager.GetForAppSpace(routeData.AppVersion, routeData.Appspace) // Change this to more solid IDs
	sb := <-sandboxChan

	if sb == nil {
		// sandbox failed to start or something
		oRes.WriteHeader(http.StatusInternalServerError)
		return
	}

	sbTransport := sb.GetTransport()

	//timetrack.Track(getTime, "getting sandbox "+appSpace+" c"+sbName)

	reqTaskCh := sb.TaskBegin()

	header := cloneHeader(oReq.Header)
	header["appspace-module"] = []string{routeData.RouteConfig.Handler.File} // verify routeData has a route config, otherwise this fails hard.
	header["appspace-fn"] = []string{routeData.RouteConfig.Handler.Function}

	cReq, err := http.NewRequest(oReq.Method, "http://unix/", oReq.Body)
	if err != nil {
		s.getLogger("ServeHTTP(), http.NewRequest()").Error(err)
		// Maybe add app id and appspace id?
		// probably need to bail if we can't create a requst
	}

	cReq.Header = header

	cRes, err := sbTransport.RoundTrip(cReq)
	if err != nil {
		s.getLogger("ServeHTTP(), sbTransport.RoundTrip()").Error(err)
	}

	// futz around with headers
	copyHeader(oRes.Header(), cRes.Header)

	oRes.WriteHeader(cRes.StatusCode)

	io.Copy(oRes, cRes.Body)

	cRes.Body.Close()

	reqTaskCh <- true
}

func (s *SandboxProxy) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("SandboxProxy")
	if note != "" {
		r.AddNote(note)
	}
	return r
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
