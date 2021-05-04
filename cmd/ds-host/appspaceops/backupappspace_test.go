package appspaceops

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestGetZipFilename(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	e := &BackupAppspace{}
	_, err = e.getZipFilename(dir)
	if err != nil {
		t.Error(err)
	}
}

func TestGetZipFilenameDupe(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	fileDateFormat = "abc"

	emptyFile, err := os.Create(filepath.Join(dir, "abc.zip"))
	if err != nil {
		t.Error(err)
	}
	emptyFile.Close()

	e := &BackupAppspace{}

	fn, err := e.getZipFilename(dir)
	if err != nil {
		t.Error(err)
	}
	if fn != filepath.Join(dir, "abc_1.zip") {
		t.Error("unexpected file name: " + fn)
	}
}
