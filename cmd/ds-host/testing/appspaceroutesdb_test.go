package integrationtests

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/sandbox"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
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
	os.MkdirAll(filepath.Join(dataDir, "apps", "app-location", "app"), 0700)
	os.MkdirAll(filepath.Join(dataDir, "appspaces", "appspace-location"), 0700)

	cfg := &domain.RuntimeConfig{}
	cfg.Sandbox.SocketsDir = socketsDir
	cfg.DataDir = dataDir
	cfg.Exec.AppspacesPath = filepath.Join(dataDir, "appspaces")
	cfg.Exec.SandboxCodePath = getSandboxCodePath()

	appsPath := filepath.Join(dataDir, "apps")
	l2p := &l2p{appFiles: appsPath, app: appsPath}

	appVersion := domain.AppVersion{
		LocationKey: "app-location"}
	appspace := domain.Appspace{
		AppspaceID:  13,
		LocationKey: "appspace-location"}

	services := testmocks.NewMockVXServices(mockCtrl)
	services.EXPECT().Get(&appspace, domain.APIVersion(0))

	appspaceLogger := testmocks.NewMockAppspaceLogger(mockCtrl)
	appspaceLogger.EXPECT().Get(appspace.AppspaceID).Return(nil)

	sM := sandbox.Manager{
		AppspaceLogger: appspaceLogger,
		Services:       services,
		Location2Path:  l2p,
		Config:         cfg}

	sM.Init()

	sandboxChan := sM.GetForAppspace(&appVersion, &appspace)
	sb := <-sandboxChan
	if sb == nil {
		t.Error("Sandbox channel closed")
	}
	defer sb.Graceful()

	data := []byte("export default function testFn() { console.log(\"testFn running\"); };")
	err = ioutil.WriteFile(path.Join(l2p.AppFiles(appVersion.LocationKey), "test-file.ts"), data, 0600)
	if err != nil {
		t.Error(err)
	}

	handler := domain.AppspaceRouteHandler{
		File: "@app/test-file.ts"}
	err = sb.ExecFn(handler)
	if err != nil {
		t.Error(err)
	}
}

func getSandboxCodePath() string {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(dir, "../../../denosandboxcode/")
}
