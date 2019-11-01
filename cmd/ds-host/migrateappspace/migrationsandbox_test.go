package migrateappspace

import (
	"log"
	"os"
	"path"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type paths struct {
	migratorScript string
	dataDir        string
}

func TestStart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger := domain.NewMockLogCLientI(mockCtrl)

	// dir, err := ioutil.TempDir("", "")
	// if err != nil {
	// 	t.Error(err)
	// }
	// defer os.RemoveAll(dir)

	p := getJSRuntimePaths()

	cfg := &domain.RuntimeConfig{}
	//cfg.Sandbox.SocketsDir = dir	// will need later when we have rev listener sockets
	cfg.DataDir = p.dataDir
	cfg.Exec.MigratorScriptPath = p.migratorScript

	s := &migrationSandbox{
		Logger: logger,
		Config: cfg}

	logger.EXPECT().Log(domain.INFO, nil, gomock.Any())

	dsErr := s.Start("workingapp", "appspace-loc-5", 1, 2)
	if dsErr != nil {
		t.Error(dsErr)
	}
}

func TestStartFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger := domain.NewMockLogCLientI(mockCtrl)

	// dir, err := ioutil.TempDir("", "")
	// if err != nil {
	// 	t.Error(err)
	// }
	// defer os.RemoveAll(dir)

	p := getJSRuntimePaths()

	cfg := &domain.RuntimeConfig{}
	//cfg.Sandbox.SocketsDir = dir	// will need later when we have rev listener sockets
	cfg.DataDir = p.dataDir
	cfg.Exec.MigratorScriptPath = p.migratorScript

	s := &migrationSandbox{
		Logger: logger,
		Config: cfg}

	logger.EXPECT().Log(domain.INFO, nil, gomock.Any())
	logger.EXPECT().Log(domain.ERROR, nil, gomock.Any())

	dsErr := s.Start("workingapp", "appspace-loc-5", 2, 3)
	if dsErr == nil {
		t.Error("expected an error")
	}
}
func getJSRuntimePaths() (ret paths) {
	dir, err := os.Getwd() // Apparently the CWD of tests is the package dir
	if err != nil {
		log.Fatal(err)
	}

	ret.migratorScript = path.Join(dir, "../../../install/files/ds-appspace-migrator.js")

	ret.dataDir = path.Join(dir, "../../../testbench/appspacemigration/")

	return
}
