package appspacefilesmodel

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// create temp dir and put that in runtime config.
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.DataDir = dir

	m := AppspaceFilesModel{
		Config: cfg}

	locKey, dsErr := m.CreateLocation()
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	if locKey == "" {
		t.Fatal("location key can not be empty string")
	}

	appspacePath := filepath.Join(m.getAppspacesPath(), locKey)

	if _, err := os.Stat(appspacePath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		t.Fatal("appspace path should exist")
	}

	// dsErr = m.Delete(locKey)
	// if dsErr != nil {
	// 	t.Fatal(dsErr)
	// }

	// _, err = os.Stat(filepath.Join(appsPath, locKey))
	// if err == nil || !os.IsNotExist(err) {
	// 	t.Fatal("expected not exist error", err)
	// }
}
