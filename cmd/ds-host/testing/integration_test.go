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

// this def fails until we at least put routes in th system before thrying to call the route.
func TestIntegration1(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	appLoc := "app123"
	appspaceLoc := "appspace123"

	socketsDir := path.Join(dir, "sockets")
	os.MkdirAll(socketsDir, 0700)

	dataDir := path.Join(dir, "data")
	os.MkdirAll(dataDir, 0700)

	os.MkdirAll(filepath.Join(dataDir, "appspaces", appspaceLoc), 0700)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = socketsDir
	cfg.DataDir = dataDir
	cfg.Exec.AppspacesPath = filepath.Join(dataDir, "appspaces")
	cfg.Exec.SandboxCodePath = getJSRuntimePath()

	appsPath := filepath.Join(dataDir, "apps")
	l2p := &l2p{appFiles: appsPath, app: appsPath}

	appspace := &domain.Appspace{DomainName: "as1.ds.dev", AppID: domain.AppID(1), LocationKey: appspaceLoc}
	appVersion := &domain.AppVersion{LocationKey: appLoc}

	tl := &testLogger{
		t: t}

	services := testmocks.NewMockVXServices(mockCtrl)
	services.EXPECT().Get(appspace, domain.APIVersion(0))
	sM := sandbox.Manager{
		Services:       services,
		AppspaceLogger: tl,
		Location2Path:  l2p,
		Config:         cfg}

	sandboxProxy := &sandboxproxy.SandboxProxy{
		SandboxManager: &sM}

	sM.Init()

	routeConfig := domain.V0AppRoute{
		ID:     "<get><end:true>/",
		Method: "get",
		Path:   domain.V0AppRoutePath{Path: "/", End: true},
		Auth:   domain.AppspaceRouteAuth{Allow: "public"},
		Type:   "function",
	}

	// So now we need to put a hello.js file at dataDir/apps/loc123/hello.js

	js := []byte(`
	import AppRouter, {AuthAllow} from '@dropserver/app-router.ts';
	const r = new AppRouter;
	r.add("get", "/", {allow:AuthAllow.public}, (ctx) => {
		ctx.req.respond({status: 200, body: 'Hello World'})
	});
	export default r;
	`)

	os.MkdirAll(l2p.AppFiles(appLoc), 0700)

	err = ioutil.WriteFile(path.Join(l2p.AppFiles(appLoc), "router.ts"), js, 0644)
	if err != nil {
		t.Error(err)
	}

	// Actually call proxy:
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "as1.ds.dev" //not necessary??
	ctx := domain.CtxWithAppVersionData(req.Context(), *appVersion)
	ctx = domain.CtxWithAppspaceData(ctx, *appspace)
	ctx = domain.CtxWithV0RouteConfig(ctx, routeConfig)
	req = req.WithContext(ctx)

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	sandboxProxy.ServeHTTP(rr, req)

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

	jsRuntime := path.Join(dir, "../../../resources/")

	return jsRuntime
}

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Log(_ domain.AppspaceID, source string, message string) {
	l.t.Logf("%v: %v\n", source, message)
}

// l2p Location2Path standin
type l2p struct {
	appFiles string
	app      string
}

func (l *l2p) AppMeta(loc string) string {
	return filepath.Join(l.app, loc)
}
func (l *l2p) AppFiles(loc string) string {
	return filepath.Join(l.appFiles, loc, "app")
}
