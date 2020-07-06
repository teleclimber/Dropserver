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
)

// want to try to test
// - server?
// - proxy
// - sandbox manager
// - sandbox
// - runner (JS)
// ..by sending a request to proxy (server?) that should trickle all the way to app code and back.

// TODO this needs fixing up after deno can serve http over unix sockets.
func TestIntegration1(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// I think we need a data dir too?
	// This is actually something we haven't touched yet.
	// We need a place to put app code and appspace data.
	// specified in config, but we'll make temp dirs and we will write files there directly.

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
	cfg.Exec.AppspacesFilesPath = filepath.Join(dataDir, "appspaces-files")
	cfg.Exec.SandboxRunnerPath = getJSRuntimePath()
	cfg.Exec.AppspacesMetaPath = dataDir

	metrics := domain.NewMockMetricsI(mockCtrl)
	metrics.EXPECT().HostHandleReq(gomock.Any())

	sM := sandbox.Manager{
		Config: cfg} //create with convenient data

	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: &sM,
		Metrics:        metrics}

	sM.Init()

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/abc",           // parametrize
		Subdomains: &[]string{"as1"}, // parametrize, or override in test fn.
		App:        &domain.App{Name: "app1"},
		AppVersion: &domain.AppVersion{LocationKey: "loc123"},
		Appspace:   &domain.Appspace{Subdomain: "as1", AppID: domain.AppID(1)},
		RouteConfig: &domain.AppspaceRouteConfig{
			Handler: domain.AppspaceRouteHandler{
				File:     "hello.js",
				Function: "hello",
			}}}

	// So now we need to put a hello.js file at dataDir/apps/loc123/hello.js

	js := []byte(`
	function hello(req, res) {
		str = 'Hello World';
		res.writeHead(200, {
			'Content-Length': Buffer.byteLength(str),
			'Content-Type': 'text/plain' 
		});
		res.end(str)
	}
	module.exports = {
		hello
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
	req.Host = "as1.teleclimber.dropserver.org" //not necessary??

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

	// Then kill the sandbox so you don't have a running node process
	sM.StopAll()
}

func getJSRuntimePath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}

	jsRuntime := path.Join(dir, "../../../resources/ds-sandbox-runner.js")

	return jsRuntime
}
