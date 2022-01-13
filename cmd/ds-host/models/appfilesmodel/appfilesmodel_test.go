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

	dat, err := ioutil.ReadFile(filepath.Join(cfg.Exec.AppsPath, locKey, "app", "file1"))
	if err != nil {
		t.Error(err)
	}
	if string(dat) != "hello world" {
		t.Error("didn't get the same file data", string(dat))
	}

	dat, err = ioutil.ReadFile(filepath.Join(cfg.Exec.AppsPath, locKey, "app", "bar/baz/file2.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(dat) != "oink oink oink" {
		t.Error("didn't get the same file2 data", string(dat))
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
	cfg.Exec.AppsPath = dir

	m := AppFilesModel{
		Location2Path: &l2p{appFiles: dir, app: dir},
		Config:        cfg}

	files := map[string][]byte{
		"file1":             []byte("hello world"),
		"bar/baz/file2.txt": []byte("oink oink oink"),
	}

	locKey, err := m.Save(&files)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ioutil.ReadFile(filepath.Join(m.Location2Path.AppFiles(locKey), "file1"))
	if err != nil {
		t.Fatal(err)
	}

	err = m.Delete(locKey)
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(filepath.Join(cfg.Exec.AppsPath, locKey))
	if err == nil || !os.IsNotExist(err) {
		t.Fatal("expected not exist error", err)
	}
}

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
