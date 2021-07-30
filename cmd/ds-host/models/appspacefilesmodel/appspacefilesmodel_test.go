package appspacefilesmodel

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
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
	cfg.Exec.AppspacesPath = dir

	m := AppspaceFilesModel{
		Config: cfg}

	locKey, err := m.CreateLocation()
	if err != nil {
		t.Fatal(err)
	}

	if locKey == "" {
		t.Fatal("location key can not be empty string")
	}

	appspacePath := filepath.Join(cfg.Exec.AppspacesPath, locKey)

	if _, err := os.Stat(appspacePath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		t.Fatal("appspace path should exist")
	}

	err = m.DeleteLocation(locKey)
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(filepath.Join(appspacePath, locKey))
	if err == nil || !os.IsNotExist(err) {
		t.Fatal("expected not exist error", err)
	}
}

func TestDeleteBadLocations(t *testing.T) {
	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = "/path/to/appspaces"

	m := AppspaceFilesModel{
		Config: cfg}

	cases := []string{"/an/absolute/path/", "../relative", "astricky/../../../relative/path/"}

	for _, c := range cases {
		err := m.DeleteLocation(c)
		if err == nil {
			t.Error("no error when we expected one: " + c)

		} else if err.Error() != "invalid location key" {
			t.Error("expected invalid location key " + err.Error())
		}
	}
}

func TestReplaceData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// create temp dir and put that in runtime config.
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	appspacesDir := filepath.Join(dir, "appspaces")
	err = os.MkdirAll(appspacesDir, 0766)
	if err != nil {
		t.Fatal(err)
	}
	replaceDir := filepath.Join(dir, "replace")
	err = os.MkdirAll(replaceDir, 0766)
	if err != nil {
		t.Fatal(err)
	}
	ioutil.WriteFile(filepath.Join(replaceDir, "hello.txt"), []byte("Hello World!"), 0644)

	cfg := &domain.RuntimeConfig{}
	cfg.Exec.AppspacesPath = appspacesDir

	asfEvents := testmocks.NewMockAppspaceFilesEvents(mockCtrl)
	asfEvents.EXPECT().Send(domain.AppspaceID(7))

	m := AppspaceFilesModel{
		Config:              cfg,
		AppspaceFilesEvents: asfEvents}

	locKey, err := m.CreateLocation()
	if err != nil {
		t.Fatal(err)
	}

	appspace := domain.Appspace{AppspaceID: domain.AppspaceID(7), LocationKey: locKey}

	err = m.ReplaceData(appspace, replaceDir)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(filepath.Join(appspacesDir, locKey, "data", "hello.txt")); os.IsNotExist(err) {
		// path/to/whatever does not exist
		t.Fatal("hello.txt should exist")
	}
}

func TestGetBackups(t *testing.T) {
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
	cfg.Exec.AppspacesPath = dir

	m := AppspaceFilesModel{
		Config: cfg}

	loc := "abcLOC"

	backupsDir := filepath.Join(dir, loc, "backups")
	err = os.MkdirAll(backupsDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := m.GetBackups(loc)
	if err != nil {
		t.Error(err)
	}
	if len(entries) != 0 {
		t.Error("expected zero entries")
	}

	file1 := "1234-56-78_7890.zip"
	file2 := "9999-56-78_7890_1.zip"

	err = ioutil.WriteFile(filepath.Join(backupsDir, file2), []byte("test data"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(backupsDir, file1), []byte("test data"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	entries, err = m.GetBackups(loc)
	if err != nil {
		t.Error(err)
	}
	if len(entries) != 2 {
		t.Error("expected 2 entries")
	}
	if entries[0] != file2 {
		t.Error("expected 9999-* entry first " + entries[0])
	}

	// I'm just going to test delete while we're all set up:
	err = m.DeleteBackup(loc, file2)
	if err != nil {
		t.Error(err)
	}
	entries, err = m.GetBackups(loc)
	if err != nil {
		t.Error(err)
	}
	if len(entries) != 1 {
		t.Error("expected 1 entries")
	}
	if entries[0] != file1 {
		t.Error("expected 1234-* entry " + entries[0])
	}

}
