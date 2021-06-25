package sandboxproxy

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
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
	tempDir        string
	sandbox        *domain.MockSandboxI
	sandboxManager *testmocks.MockSandboxManager
	sandboxProxy   *SandboxProxy
	sandboxServer  *http.Server
	appVersion     domain.AppVersion
	appspace       domain.Appspace
	routeConfig    domain.V0AppRoute
}

func TestSandboxBadStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ch := make(chan domain.SandboxI)
	close(ch) //sandbox manager closes the channel to indicate bad start.

	sandboxManager := testmocks.NewMockSandboxManager(mockCtrl)
	sandboxManager.EXPECT().GetForAppspace(gomock.Any(), gomock.Any()).Return(ch)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := req.Context()
	ctx = domain.CtxWithAppVersionData(ctx, domain.AppVersion{})
	ctx = domain.CtxWithAppspaceData(ctx, domain.Appspace{})

	rr := httptest.NewRecorder()

	sandboxProxy := SandboxProxy{
		SandboxManager: sandboxManager,
	}

	sandboxProxy.ServeHTTP(rr, req.WithContext(ctx))

	if rr.Result().StatusCode != http.StatusInternalServerError {
		t.Error("got wrong status " + rr.Result().Status)
	}
}

func TestServeHTTP200(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tm := createMocks(mockCtrl, func(w http.ResponseWriter, r *http.Request) {
		routeID := r.Header.Get("X-Dropserver-Route-ID")
		if routeID != "test-route-id" {
			t.Error("wrong route id: " + routeID)
		}
		w.WriteHeader(200)            // parametrize
		fmt.Fprintf(w, "Hello World") // return w? Or parametrize the handler.
	})
	defer closeMocks(tm)

	// from https://blog.questionable.services/article/testing-http-handlers-go/
	// craft a request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithV0RouteConfig(req.Context(), tm.routeConfig))

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	tm.sandboxProxy.ServeHTTP(rr, req)

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

	tm := createMocks(mockCtrl, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	defer closeMocks(tm)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	tm.sandboxProxy.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if rr.Code != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func createMocks(mockCtrl *gomock.Controller, sbHandler func(http.ResponseWriter, *http.Request)) *testMocks {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	serverSocket := filepath.Join(tempDir, "server.sock")

	sandboxManager := testmocks.NewMockSandboxManager(mockCtrl)

	sandboxProxy := &SandboxProxy{
		SandboxManager: sandboxManager}

	sandbox := domain.NewMockSandboxI(mockCtrl)
	sandbox.EXPECT().GetTransport().Return(&http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", serverSocket)
		},
	})

	// dummy server to stand in for sandbox
	//ts := httptest.NewServer(http.HandlerFunc(sbHandler))

	server := http.Server{
		Handler: http.HandlerFunc(sbHandler)}

	unixListener, err := net.Listen("unix", serverSocket)
	if err != nil {
		panic(err)
	}
	go server.Serve(unixListener)

	taskCh := make(chan bool)
	sandbox.EXPECT().TaskBegin().Return(taskCh)
	go func() {
		<-taskCh
		fmt.Println("task done")
	}()

	sandboxManager.EXPECT().GetForAppspace(gomock.Any(), gomock.Any()).DoAndReturn(func(av *domain.AppVersion, as *domain.Appspace) chan domain.SandboxI {
		sandboxChan := make(chan domain.SandboxI)
		go func() {
			sandboxChan <- sandbox
		}()
		return sandboxChan
	})

	return &testMocks{
		tempDir:        tempDir,
		sandbox:        sandbox,
		sandboxManager: sandboxManager,
		sandboxProxy:   sandboxProxy,
		sandboxServer:  &server,
		appVersion:     domain.AppVersion{},
		appspace:       domain.Appspace{DomainName: "as1.ds.dev"},
		routeConfig: domain.V0AppRoute{
			ID:   "test-route-id",
			Type: "function",
		}}

}

func closeMocks(tm *testMocks) {
	os.RemoveAll(tm.tempDir)
	tm.sandboxServer.Close()
}
