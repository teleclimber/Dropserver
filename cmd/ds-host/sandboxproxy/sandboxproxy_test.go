package sandboxproxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// can we test more situations?
// what are inputs that would vary, and what are the outputs?
// In:
// - req (method, path, Host, ...headers, body, ...)
// - routeData (urlTail, subdomains, App?, Appspace?)
// Out:
// - dummy server (response code, headers, body...)
// - log message level (might be more than one if error?)
// - ...?
//
// Not sure what the best way to do this is.
// Maybe have a function that creates the Mocks, with a few overridable parameters?
// ..and sets EXPECTS that are common everywhere
//

type testMocks struct {
	sandboxProxy  *SandboxProxy
	sandboxServer *httptest.Server
	sbLogger      *domain.MockLogCLientI
	routeData     *domain.AppspaceRouteData
}

func TestServeHTTP200(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tm := createMocks(mockCtrl, sandboxHandler200)
	defer tm.sandboxServer.Close()

	tm.sbLogger.EXPECT().Log(domain.INFO, gomock.Any(), gomock.Any())

	// from https://blog.questionable.services/article/testing-http-handlers-go/
	// craft a request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "as1.teleclimber.dropserver.org" //not necessary??

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	tm.sandboxProxy.ServeHTTP(rr, req, tm.routeData)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `Hello World`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestServeHTTP404(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tm := createMocks(mockCtrl, sandboxHandler404)
	defer tm.sandboxServer.Close()

	tm.sbLogger.EXPECT().Log(domain.INFO, gomock.Any(), gomock.Any())

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	tm.sandboxProxy.ServeHTTP(rr, req, tm.routeData)

	// Check the status code is what we expect.
	if rr.Code != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func createMocks(mockCtrl *gomock.Controller, sbHandler func(http.ResponseWriter, *http.Request)) testMocks {
	sandboxManager := domain.NewMockSandboxManagerI(mockCtrl)
	logger := domain.NewMockLogCLientI(mockCtrl)
	metrics := domain.NewMockMetricsI(mockCtrl)

	sandboxProxy := &SandboxProxy{
		SandboxManager: sandboxManager,
		Logger:         logger,
		Metrics:        metrics}

	routeData := &domain.AppspaceRouteData{
		URLTail:     "/abc",           // parametrize
		Subdomains:  &[]string{"as1"}, // parametrize, or override in test fn.
		App:         &domain.App{Name: "app1"},
		AppVersion:  &domain.AppVersion{},
		Appspace:    &domain.Appspace{Subdomain: "as1", AppID: domain.AppID(1)},
		RouteConfig: &domain.RouteConfig{}}

	metrics.EXPECT().HostHandleReq(gomock.Any())

	sandbox := domain.NewMockSandboxI(mockCtrl)
	sandbox.EXPECT().GetTransport().Return(http.DefaultTransport)

	// dummy server to stand in for sandbox
	ts := httptest.NewServer(http.HandlerFunc(sbHandler))

	sandbox.EXPECT().GetPort().Return(ts.Listener.Addr().(*net.TCPAddr).Port)

	sbLogger := domain.NewMockLogCLientI(mockCtrl)
	sandbox.EXPECT().GetLogClient().Return(sbLogger)

	taskCh := make(chan bool)
	sandbox.EXPECT().TaskBegin().Return(taskCh)
	go func() {
		<-taskCh
		fmt.Println("task done")
	}()

	sandboxManager.EXPECT().GetForAppSpace(gomock.Any(), gomock.Any()).DoAndReturn(func(av *domain.AppVersion, as *domain.Appspace) chan domain.SandboxI {
		sandboxChan := make(chan domain.SandboxI)
		go func() {
			sandboxChan <- sandbox
		}()
		return sandboxChan
	})

	return testMocks{
		sandboxProxy:  sandboxProxy,
		sandboxServer: ts,
		sbLogger:      sbLogger,
		routeData:     routeData,
	}

}

func sandboxHandler200(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got request in dummy sandbox server")
	w.WriteHeader(200)            // parametrize
	fmt.Fprintf(w, "Hello World") // return w? Or parametrize the handler.
}

func sandboxHandler404(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got request in dummy sandbox server")
	w.WriteHeader(404)
}
