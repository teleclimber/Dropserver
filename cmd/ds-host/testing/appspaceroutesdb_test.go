package integrationtests

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacedb"
	"github.com/teleclimber/DropServer/cmd/ds-host/appspacemetadb"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
	"github.com/teleclimber/DropServer/cmd/ds-host/vxservices"
	"github.com/teleclimber/DropServer/internal/validator"
)

func TestSandboxExecFn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Before I canstart:
	// - ds-sandbox runner needs to be able to start, connect twine, and act when an incoming "exec fn" call comes in.
	// - means I also need a host-side "exec" function

	// Need:
	// - appspace script that creates route
	// - sandbox
	// - meta db with in-memroy db

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	socketsDir := path.Join(dir, "sockets")
	os.MkdirAll(socketsDir, 0700)
	dataDir := path.Join(dir, "data")
	err = os.MkdirAll(filepath.Join(dataDir, "apps", "app-location"), 0700)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = socketsDir
	cfg.DataDir = dataDir
	cfg.Exec.AppsPath = filepath.Join(dataDir, "apps")
	cfg.Exec.AppspacesPath = filepath.Join(dataDir, "appspaces")
	cfg.Exec.SandboxCodePath = getSandboxCodePath()
	cfg.Exec.SandboxRunnerPath = getSandboxRunnerPath()

	appVersion := domain.AppVersion{
		LocationKey: "app-location"}
	appspace := domain.Appspace{
		AppspaceID:  13,
		LocationKey: "appspace-location"}

	services := testmocks.NewMockVXServices(mockCtrl)
	services.EXPECT().Get(&appspace, domain.APIVersion(0))

	sM := sandbox.Manager{
		Services: services,
		Config:   cfg}

	sM.Init()

	sandboxChan := sM.GetForAppSpace(&appVersion, &appspace)
	sb := <-sandboxChan
	if sb == nil {
		t.Error("Sandbox channel closed")
	}
	defer sb.Graceful()

	data := []byte("export default function testFn() { console.log(\"testFn running\"); };")
	err = ioutil.WriteFile(path.Join(cfg.Exec.AppsPath, "app-location", "test-file.ts"), data, 0600)
	if err != nil {
		t.Error(err)
	}

	handler := domain.AppspaceRouteHandler{
		File: "@app/test-file.ts"}
	dsErr := sb.ExecFn(handler)
	if dsErr != nil {
		t.Error(dsErr)
	}
}

func TestSandboxCreateRoute(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Need:
	// - appspace script that creates route
	// - sandbox
	// - meta db with in-memroy db

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	socketsDir := path.Join(dir, "sockets")
	os.MkdirAll(socketsDir, 0700)
	dataDir := path.Join(dir, "data")
	err = os.MkdirAll(filepath.Join(dataDir, "apps", "app-location"), 0700)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = socketsDir
	cfg.DataDir = dataDir
	cfg.Exec.AppsPath = filepath.Join(dataDir, "apps")
	cfg.Exec.AppspacesPath = filepath.Join(dataDir, "appspaces")
	cfg.Exec.SandboxCodePath = getSandboxCodePath()
	cfg.Exec.SandboxRunnerPath = getSandboxRunnerPath()

	appspaceID := domain.AppspaceID(13)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{}, nil)

	v := &validator.Validator{}
	v.Init()

	appspaceMetaDb := &appspacemetadb.AppspaceMetaDB{
		AppspaceModel: appspaceModel,
		Config:        cfg,
		Validator:     v}
	appspaceMetaDb.Init()
	appspaceRouteModels := &appspacemetadb.AppspaceRouteModels{
		Config:         cfg,
		AppspaceMetaDB: appspaceMetaDb,
		Validator:      v}
	appspaceRouteModels.Init()

	appspaceDB := &appspacedb.AppspaceDB{
		Config: cfg}
	appspaceDB.Init()

	services := &vxservices.VXServices{
		RouteModels:  appspaceRouteModels,
		V0AppspaceDB: appspaceDB.V0}

	sM := sandbox.Manager{
		Services: services,
		Config:   cfg}

	appVersion := domain.AppVersion{
		LocationKey: "app-location"}
	appspace := domain.Appspace{
		AppspaceID:  appspaceID,
		LocationKey: "appspace-location"}

	appspaceMetaDb.Create(appspace.AppspaceID, 0)
	sM.Init()

	sandboxChan := sM.GetForAppSpace(&appVersion, &appspace)
	sb := <-sandboxChan
	defer sb.Graceful()

	ts := `
	import Routes from "@dropserver/appspace-routes-db.ts";
	export default async function createRoute() {
		await Routes.createRoute(["get", "post"], "/some/path", {allow:"owner"}, {file:"file.ts", function:"handleRoute", type:"function"});
	}`

	data := []byte(ts)
	err = ioutil.WriteFile(path.Join(cfg.Exec.AppsPath, "app-location", "test-file.ts"), data, 0600)
	if err != nil {
		t.Error(err)
	}

	handler := domain.AppspaceRouteHandler{
		File: "@app/test-file.ts"}
	err = sb.ExecFn(handler)
	if err != nil {
		t.Error(err)
	}

	// check to see if the route exists in the DB...
	rm := appspaceRouteModels.GetV0(domain.AppspaceID(13))
	route, dsErr := rm.Match("get", "/some/path")
	if dsErr != nil {
		t.Error(dsErr)
	}
	if route == nil {
		t.Error("Expected one route")
	}
}

func getSandboxCodePath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(dir, "../../../resources/")
}
func getSandboxRunnerPath() string {
	return filepath.Join(getSandboxCodePath(), "ds-sandbox-runner.ts")
}
