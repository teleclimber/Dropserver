package sandboxproxy

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// SandboxProxy holds other structs for the proxy
type SandboxProxy struct {
	SandboxManager interface {
		GetForAppspace(*domain.AppVersion, *domain.Appspace) (domain.SandboxI, chan struct{})
	} `checkinject:"required"`
}

// ServeHTTP forwards the request to a sandbox
// Could still see splitting this function in two.
func (s *SandboxProxy) ServeHTTP(oRes http.ResponseWriter, oReq *http.Request) {
	// The responsibiility for knowing whether an appspace is ready or not, is upstream (in appspaceroutes)

	ctx := oReq.Context()
	appVersion, _ := domain.CtxAppVersionData(ctx)
	appspace, _ := domain.CtxAppspaceData(ctx)

	sb, taskCh := s.SandboxManager.GetForAppspace(&appVersion, &appspace) // Change this to more solid IDs
	defer close(taskCh)                                                   // signal end of task

	sb.WaitFor(domain.SandboxReady)
	if sb.Status() != domain.SandboxReady {
		oRes.WriteHeader(http.StatusInternalServerError)
		return
	}

	sbTransport := sb.GetTransport()

	routeConfig, _ := domain.CtxV0RouteConfig(ctx)

	header := oReq.Header.Clone()
	header.Set("X-Dropserver-Request-URL", getURLString(*oReq.URL))
	header.Set("X-Dropserver-Route-ID", routeConfig.ID)

	proxyID, ok := domain.CtxAppspaceUserProxyID(ctx)
	if ok {
		header.Set("X-Dropserver-User-ProxyID", string(proxyID))
	}

	cReq, err := http.NewRequest(oReq.Method, "http://unix/", oReq.Body)
	if err != nil {
		s.getLogger("ServeHTTP(), http.NewRequest()").Error(err)
		// Maybe add app id and appspace id?
		oRes.WriteHeader(http.StatusInternalServerError)
		return
	}

	cReq.Header = header

	taskCh <- struct{}{} // signal start of task

	cRes, err := sbTransport.RoundTrip(cReq)
	if err != nil {
		s.getLogger("ServeHTTP(), sbTransport.RoundTrip()").Error(err)
		oRes.WriteHeader(http.StatusInternalServerError)
		return
	}

	cspKey := http.CanonicalHeaderKey("Content-Security-Policy")
	resHeader := oRes.Header()
	for k, vv := range cRes.Header {
		k = http.CanonicalHeaderKey(k)
		// filter out CSP, CORS, etc...
		// Furthermore we could compare against a list of known and acceptable headers,
		// and if not known mandate that it start with "X-"
		if k == cspKey {
			continue
		}
		for _, v := range vv {
			resHeader.Add(k, v)
		}
	}

	oRes.WriteHeader(cRes.StatusCode)

	io.Copy(oRes, cRes.Body)
	cRes.Body.Close()
}

func (s *SandboxProxy) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("SandboxProxy")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

func getURLString(u url.URL) string {
	// copied in part from url.URL.String() implementation in golang std lib
	// We assume no scheme/user/host
	// We do not encode url.

	var buf strings.Builder
	path := u.Path

	// RFC 3986 §4.2
	// A path segment that contains a colon character (e.g., "this:that")
	// cannot be used as the first segment of a relative-path reference, as
	// it would be mistaken for a scheme name. Such a segment must be
	// preceded by a dot-segment (e.g., "./this:that") to make a relative-
	// path reference.
	// (We should not run into this here, but leaving anyways)
	if i := strings.IndexByte(path, ':'); i > -1 && strings.IndexByte(path[:i], '/') == -1 {
		buf.WriteString("./")
	}
	buf.WriteString(path)

	if u.RawQuery != "" {
		buf.WriteByte('?')
		buf.WriteString(u.RawQuery)
	}
	if u.Fragment != "" {
		buf.WriteByte('#')
		buf.WriteString(u.Fragment)
	}
	return buf.String()
}
