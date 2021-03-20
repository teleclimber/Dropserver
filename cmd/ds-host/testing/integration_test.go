package integrationtests

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandboxproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

// want to try to test
// - server?
// - proxy
// - sandbox manager
// - sandbox
// - runner (JS)
// ..by sending a request to proxy (server?) that should trickle all the way to app code and back.

func TestIntegration1(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	socketsDir := path.Join(dir, "sockets")
	os.MkdirAll(socketsDir, 0700)

	dataDir := path.Join(dir, "data")
	os.MkdirAll(dataDir, 0700)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = socketsDir
	cfg.DataDir = dataDir
	cfg.Exec.AppsPath = filepath.Join(dataDir, "apps")
	cfg.Exec.AppspacesPath = filepath.Join(dataDir, "appspaces")
	cfg.Exec.SandboxRunnerPath = getJSRuntimePath()

	appspace := &domain.Appspace{DomainName: "as1.ds.dev", AppID: domain.AppID(1)}
	appVersion := &domain.AppVersion{LocationKey: "loc123"}

	metrics := domain.NewMockMetricsI(mockCtrl)
	metrics.EXPECT().HostHandleReq(gomock.Any())

	tl := &testLogger{
		t: t}

	services := testmocks.NewMockVXServices(mockCtrl)
	services.EXPECT().Get(appspace, domain.APIVersion(0))
	sM := sandbox.Manager{
		Services:       services,
		AppspaceLogger: tl,
		Config:         cfg}

	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: &sM,
		Metrics:        metrics}

	sM.Init()

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/abc", // parametrize
		App:        &domain.App{Name: "app1"},
		AppVersion: appVersion,
		Appspace:   appspace,
		RouteConfig: &domain.AppspaceRouteConfig{
			Handler: domain.AppspaceRouteHandler{
				File:     "@app/hello.js",
				Function: "hello",
			}}}

	// So now we need to put a hello.js file at dataDir/apps/loc123/hello.js

	js := []byte(`
	export function hello(req) {
		req.respond({status: 200, body: 'Hello World'})
	}
	`)

	appDir := path.Join(dataDir, "apps", "loc123")
	os.MkdirAll(appDir, 0700)

	err = ioutil.WriteFile(path.Join(appDir, "hello.js"), js, 0644)
	if err != nil {
		t.Error(err)
	}

	// Actually call proxy:
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "as1.ds.dev" //not necessary??

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	sandboxProxy.ServeHTTP(rr, req, routeData)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `Hello World`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	sM.StopAll()
}

func getJSRuntimePath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}

	jsRuntime := path.Join(dir, "../../../resources/ds-sandbox-runner.ts")

	return jsRuntime
}

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Log(_ domain.AppspaceID, source string, message string) {
	l.t.Logf("%v: %v\n", source, message)
}