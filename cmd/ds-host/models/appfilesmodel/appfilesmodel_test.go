package appfilesmodel

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestPathInsidePath(t *testing.T) {
	cases := []struct {
		p      string
		root   string
		inside bool
	}{
		{"/foo/bar/baz", "/foo/bar/baz", true},
		{"/foo/bar/zoink", "/foo/bar/baz", false},
		{"/foo/bar/baz", "/foo/bar/baz", true},
		{"/foo/bar/baz/../zoink", "/foo/bar/baz/", false},
		{"/foo/bar/baz/..", "/foo/bar/baz/", false},
	}

	for _, c := range cases {
		inside, err := pathInsidePath(c.p, c.root)
		if err != nil {
			t.Error(err)
		}
		if inside != c.inside {
			t.Error("mismatched inside", c.p, c.root)
		}
	}
}

func TestDecodeAppJsonError(t *testing.T) {
	// check that passing a json does return the struct as expceted
	// check that badly formed json returns the correct error code.
	r := strings.NewReader(`{ "name":"blah", "version":"0.0.1 }`)
	_, err := decodeAppJSON(r)
	if err == nil {
		t.Error("Error was nil")
	}
}

func TestDecodeAppJSON(t *testing.T) {
	r := strings.NewReader(`{
		"name":"blah",
		"version":"0.0.1"
	}`)

	meta, err := decodeAppJSON(r)
	if err != nil {
		t.Error("got error for well formed json")
	}
	if meta.AppName != "blah" || meta.AppVersion != "0.0.1" {
		t.Error("got incorrect values for meta")
	}
}

func TestSave(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// create temp dir and put that in runtime config.
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.DataDir = dir
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")

	m := AppFilesModel{
		Config: cfg}

	// for files, create dummy data
	// will read back to check it's there
	files := map[string][]byte{
		"file1":             []byte("hello world"),
		"bar/baz/file2.txt": []byte("oink oink oink"),
	}

	locKey, dsErr := m.Save(&files)
	if dsErr != nil {
		t.Error(dsErr)
	}

	dat, err := ioutil.ReadFile(filepath.Join(cfg.Exec.AppsPath, locKey, "file1"))
	if err != nil {
		t.Error(err)
	}
	if string(dat) != "hello world" {
		t.Error("didn't get the same file data", string(dat))
	}

	dat, err = ioutil.ReadFile(filepath.Join(cfg.Exec.AppsPath, locKey, "bar/baz/file2.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(dat) != "oink oink oink" {
		t.Error("didn't get the same file2 data", string(dat))
	}

}

func TestGetMigrationDirs(t *testing.T) {
	// create temp dir and put that in runtime config.
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.DataDir = dir
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")

	m := AppFilesModel{
		Config: cfg}

	for _, d := range []string{"boo", "0", "5", "zoink", "2b", "3"} {
		os.MkdirAll(filepath.Join(cfg.Exec.AppsPath, "abc", "migrations", d), 0766)
	}

	mInts, dsErr := m.getMigrationDirs("abc")
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	if len(mInts) != 2 {
		t.Fatal("wrong length for migration ints", mInts)
	}
	if mInts[0] != 3 {
		t.Fatal("wrong order for migrations", mInts)
	}

}

// test removal please
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
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")

	m := AppFilesModel{
		Config: cfg}

	files := map[string][]byte{
		"file1":             []byte("hello world"),
		"bar/baz/file2.txt": []byte("oink oink oink"),
	}

	locKey, dsErr := m.Save(&files)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	_, err = ioutil.ReadFile(filepath.Join(cfg.Exec.AppsPath, locKey, "file1"))
	if err != nil {
		t.Fatal(err)
	}

	dsErr = m.Delete(locKey)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	_, err = os.Stat(filepath.Join(cfg.Exec.AppsPath, locKey))
	if err == nil || !os.IsNotExist(err) {
		t.Fatal("expected not exist error", err)
	}
}
