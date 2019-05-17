package sandboxproxy

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// SandboxProxy holds other structs for the proxy
type SandboxProxy struct {
	SandboxManager domain.SandboxManagerI // not needed at server level
	Logger         domain.LogCLientI
	Metrics        domain.MetricsI
}

// ServeHTTP forwards the request to a sandbox
// Could still see splitting this function in two.
func (s *SandboxProxy) ServeHTTP(oRes http.ResponseWriter, oReq *http.Request, routeData *domain.AppspaceRouteData) {
	defer s.Metrics.HostHandleReq(time.Now())

	appName := routeData.App.Name
	appspaceName := routeData.Appspace.Name

	fmt.Println("in request handler", appspaceName, appName)

	sandboxChan := s.SandboxManager.GetForAppSpace(appName, appspaceName)
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
		sb.GetLogClient().Log(domain.ERROR, map[string]string{
			"app-space": appspaceName, "app": appName},
			"http.NewRequest error: "+err.Error())

		fmt.Println("http.NewRequest error", sbName, oReq.Method, sbAddress, err)
		//os.Exit(1)
		// don't exit, but need to think about how to deal with this gracefully.
	}

	cReq.Header = header

	cRes, err := sbTransport.RoundTrip(cReq)
	if err != nil {
		sb.GetLogClient().Log(domain.ERROR, map[string]string{
			"app-space": appspaceName, "app": appName},
			"sb.Transport.RoundTrip(cReq) error: "+err.Error())
		fmt.Println("sb.Transport.RoundTrip(cReq) error", sbName, err)
		//os.Exit(1)
	}

	// futz around with headers
	copyHeader(oRes.Header(), cRes.Header)

	oRes.WriteHeader(cRes.StatusCode)

	io.Copy(oRes, cRes.Body)

	cRes.Body.Close()

	reqTaskCh <- true

	sb.GetLogClient().Log(domain.INFO, map[string]string{
		"app-space": appspaceName, "app": appName},
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