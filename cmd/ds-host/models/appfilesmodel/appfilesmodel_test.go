package appfilesmodel

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	r := []byte(`{ "name":"blah", "version":"0.0.1 }`)
	_, err := unmarshalManifest(r)
	if err == nil {
		t.Error("Error was nil")
	}
}

func TestCreateLocation(t *testing.T) {
	// TODO
}

func TestDecodeAppJSON(t *testing.T) {
	r := []byte(`{
		"name":"blah",
		"version":"0.0.1"
	}`)

	manifest, err := unmarshalManifest(r)
	if err != nil {
		t.Error("got error for well formed json")
	}
	if manifest.Name != "blah" || manifest.Version != "0.0.1" {
		t.Error("got incorrect values for meta")
	}
}

func TestSavePackage(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.DataDir = dir
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")

	m := AppFilesModel{
		Config:           cfg,
		AppLocation2Path: &appl2p{appFiles: cfg.Exec.AppsPath, app: cfg.Exec.AppsPath}}

	contents := bytes.NewBuffer([]byte("hello world"))

	locKey, err := m.SavePackage(contents)
	if err != nil {
		t.Error(err)
	}
	if locKey == "" {
		t.Error("location key should not be empty")
	}

	dat, err := ioutil.ReadFile(filepath.Join(cfg.Exec.AppsPath, locKey, "package.tar.gz"))
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal([]byte("hello world"), dat) {
		t.Error("didn't get the same file data", string(dat))
	}
}

type fileList []struct {
	name     string
	contents []byte
}

func TestExtractPackageLow(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	var files = fileList{
		{"abc.txt", []byte("hello")},
		{"deep/nested/file.txt", []byte("it's dark down here")},
	}

	err = ExtractPackageLow(createPackage(files), dir)
	if err != nil {
		t.Error(err)
	}

	for _, f := range files {
		contents, err := os.ReadFile(filepath.Join(dir, f.name))
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(f.contents, contents) {
			t.Errorf("file contents are different: %s %s", f.contents, contents)
		}
	}
}

func TestExtractBadPackage(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	var files = fileList{
		{"/abc.txt", []byte("hello")},
	}
	err = ExtractPackageLow(createPackage(files), dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Errorf("expected error with absolute, got %s", err.Error())
	}

	files = fileList{
		{"trick/../../abc.txt", []byte("hello")},
	}
	err = ExtractPackageLow(createPackage(files), dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "outside") {
		t.Errorf("expected error with outside, got %s", err.Error())
	}
}

func TestExtractPackage(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	cfg := &domain.RuntimeConfig{}
	cfg.DataDir = dir
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")

	m := AppFilesModel{
		Config:           cfg,
		AppLocation2Path: &appl2p{appFiles: cfg.Exec.AppsPath, app: cfg.Exec.AppsPath}}

	var files = fileList{
		{"abc.txt", []byte("hello")},
		{"deep/nested/file.txt", []byte("it's dark down here")},
	}

	var buf bytes.Buffer
	r := createPackage(files)
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatal(err)
	}
	loc, err := m.SavePackage(&buf)
	if err != nil {
		t.Fatal(err)
	}

	err = m.ExtractPackage(loc)
	if err != nil {
		t.Error(err)
	}

	for _, f := range files {
		contents, err := os.ReadFile(filepath.Join(m.AppLocation2Path.Files(loc), f.name))
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(f.contents, contents) {
			t.Errorf("file contents are different: %s %s", f.contents, contents)
		}
	}
}

func createPackage(files fileList) io.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: 0644,
			Size: int64(len(f.contents))}
		err := tw.WriteHeader(hdr)
		if err != nil {
			panic(err)
		}
		_, err = tw.Write(f.contents)
		if err != nil {
			panic(err)
		}
	}
	tw.Close()

	var outBuf bytes.Buffer
	gzw := gzip.NewWriter(&outBuf)
	// gzw.Name = name
	// gzw.Comment = comment
	// gzw.ModTime = modTime.Round(time.Second)

	_, err := gzw.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}
	gzw.Close()

	return &outBuf
}

func TestWriteAppIconLink(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)
	cfg := &domain.RuntimeConfig{}
	cfg.DataDir = dir
	cfg.Exec.AppsPath = filepath.Join(dir, "apps")

	m := AppFilesModel{
		Config:           cfg,
		AppLocation2Path: &appl2p{appFiles: cfg.Exec.AppsPath, app: cfg.Exec.AppsPath}}

	loc, err := m.createLocation()
	if err != nil {
		t.Error(err)
	}
	err = m.WriteAppIconLink(loc, "")
	if err != nil {
		t.Error(err)
	}

	err = m.WriteAppIconLink(loc, "nonsense")
	if err == nil {
		t.Error("should get an error when attempting to link to a non-existing file")
	}

	err = os.MkdirAll(m.AppLocation2Path.Files(loc), 0766)
	if err != nil {
		t.Error(err)
	}
	icon := "some-icon.txt"
	err = os.WriteFile(filepath.Join(m.AppLocation2Path.Files(loc), icon), []byte("hello"), 0644)
	if err != nil {
		t.Error(err)
	}

	err = m.WriteAppIconLink(loc, icon)
	if err != nil {
		t.Error(err)
	}

	// for now read it directly. replace when we have reader / opener functions.
	b, err := os.ReadFile(filepath.Join(m.AppLocation2Path.Meta(loc), "app-icon"))
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(b, []byte("hello")) {
		t.Error("got wrong conetent for app icon")
	}
}

type appl2p struct {
	appFiles string
	app      string
}

func (l *appl2p) Base(loc string) string {
	return filepath.Join(l.app, loc)
}
func (l *appl2p) Meta(loc string) string {
	return filepath.Join(l.app, loc)
}
func (l *appl2p) Files(loc string) string {
	return filepath.Join(l.appFiles, loc, "app")
}
