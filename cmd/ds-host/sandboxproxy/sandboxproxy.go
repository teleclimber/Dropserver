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
		GetForAppspace(*domain.AppVersion, *domain.Appspace) chan domain.SandboxI
	} `checkinject:"required"`
}

// ServeHTTP forwards the request to a sandbox
// Could still see splitting this function in two.
func (s *SandboxProxy) ServeHTTP(oRes http.ResponseWriter, oReq *http.Request) {
	// The responsibiility for knowing whether an appspace is ready or not, is upstream (in appspaceroutes)

	ctx := oReq.Context()
	appVersion, _ := domain.CtxAppVersionData(ctx)
	appspace, _ := domain.CtxAppspaceData(ctx)

	sandboxChan := s.SandboxManager.GetForAppspace(&appVersion, &appspace) // Change this to more solid IDs
	sb := <-sandboxChan

	if sb == nil {
		// sandbox failed to start or something
		oRes.WriteHeader(http.StatusInternalServerError)
		return
	}

	sbTransport := sb.GetTransport()

	//timetrack.Track(getTime, "getting sandbox "+appSpace+" c"+sbName)

	reqTaskCh := sb.TaskBegin()
	defer func() {
		reqTaskCh <- true
	}()

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

	cRes, err := sbTransport.RoundTrip(cReq)
	if err != nil {
		s.getLogger("ServeHTTP(), sbTransport.RoundTrip()").Error(err)
		oRes.WriteHeader(http.StatusInternalServerError)
		return
	}

	cRes.Header = oRes.Header().Clone()
	// Here we need to delete a bunch of headers, like CSP, CORS, etc...

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
