package appfilesmodel

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
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
	} else if err.Code() != dserror.AppConfigParseFailed {
		t.Error("wrong error code", err.Code())
	}
}

func TestDecodeAppJSON(t *testing.T) {
	r := strings.NewReader(`{
		"name":"blah",
		"version":"0.0.1",
		"routes": [{
			"route":	"/",
			"method":	"get",
			"authorize": "owner",
			"handler": {
				"type":		"static",
				"file":		"index.html"
			}
		},{
			"route": 	"/hit",
			"method":	"post",
			"authorize":"owner",
			"handler": {
				"type":		"function",
				"file":		"routes.js",
				"function":	"postHit"
			}
		},
		{
			"route": 	"/hit",
			"method":	"get",
			"authorize":"owner",
			"handler": {
				"type":		"function",
				"file":		"routes.js",
				"function":	"getHit"
			}
		}
	]
	}`)

	meta, err := decodeAppJSON(r)
	if err != nil {
		t.Error("got error for well formed json")
	}
	if meta.AppName != "blah" || meta.AppVersion != "0.0.1" {
		t.Error("got incorrect values for meta")
	}
	if len(meta.Routes) != 3 {
		t.Error("expecte 3 routes", meta)
	}

	route := meta.Routes[1]
	expectedRoute := domain.JSONRoute{
		Route:     "/hit",
		Method:    "post",
		Authorize: "owner",
		Handler: domain.JSONRouteHandler{
			Type:     "function",
			File:     "routes.js",
			Function: "postHit"}}

	if route != expectedRoute {
		fmt.Println("expetced / got:", expectedRoute, route)
		t.Error("route doesn't match expected")
	}

}

func TestValidateAppMeta(t *testing.T) {
	cases := []struct {
		json string
		err  bool
	}{
		{`{ "name":"blah", "version":"0.0.1" }`, false},
		{`{ "version":"0.0.1" }`, true},
		{`{ "name":"blah", "foo":"0.0.1" }`, true},
	}

	for _, c := range cases {
		r := strings.NewReader(c.json)
		meta, _ := decodeAppJSON(r)

		dsErr := validateAppMeta(meta)
		hasErr := dsErr != nil
		if hasErr != c.err {
			t.Error("error mismatch", meta, dsErr.ExtraMessage())
		}
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

	logger := domain.NewMockLogCLientI(mockCtrl)
	logger.EXPECT().Log(domain.INFO, gomock.Any(), gomock.Any())

	m := AppFilesModel{
		Config: cfg,
		Logger: logger}

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

	appsPath := m.getAppsPath()

	dat, err := ioutil.ReadFile(filepath.Join(appsPath, locKey, "file1"))
	if err != nil {
		t.Error(err)
	}
	if string(dat) != "hello world" {
		t.Error("didn't get the same file data", string(dat))
	}

	dat, err = ioutil.ReadFile(filepath.Join(appsPath, locKey, "bar/baz/file2.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(dat) != "oink oink oink" {
		t.Error("didn't get the same file2 data", string(dat))
	}

}
