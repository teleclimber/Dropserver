package appops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetKey(t *testing.T) {
	g := &AppGetter{}
	g.Init()

	d := g.set(appGetData{locationKey: "abc"})

	data, ok := g.keys[d.key]
	if !ok {
		t.Error("key not set")
	}
	if data.locationKey != "abc" {
		t.Error("data not set correctly")
	}
}

func TestGetMeta(t *testing.T) {
	g := &AppGetter{}
	g.Init()

	keyData := g.set(appGetData{})
	m, ok := g.GetResults(keyData.key)
	if !ok {
		t.Fatal("expected meta should exist")
	}

	g.appendErrorResult(keyData.key, "some error")
	m3, _ := g.GetResults(keyData.key)
	if len(m3.Errors) != 1 || m3.Errors[0] != "some error" {
		t.Fatalf("expected error to be in meta: %v", m.Errors)
	}
}

func TestValidatePackageFile(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"/", "", false},
		{".", "", false},
		{"abc/../def/", "def", true},
		{"abc/../../def/", "", false},
		{"/abc/def", "abc/def", true},
		{"abc\\def", "abc\\def", false},
	}
	for _, c := range cases {
		p, ok := validatePackagePath(c.input)
		if c.ok != ok {
			t.Errorf("mismatchin OK value: expected: %v, got: %v", c.ok, ok)
		}
		if c.ok && c.expected != p {
			t.Errorf("expected entrypoint does not match: %v %v", c.expected, p)
		}
	}
}

type l2p struct {
	appFiles string
}

func (l *l2p) Files(loc string) string {
	return l.appFiles
}

func TestGetDefaultEntryPoint(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := &AppGetter{
		AppLocation2Path: &l2p{appFiles: dir},
	}
	g.Init()
	keyData := g.set(appGetData{})

	// getEntrypoint when there are no files:
	err = g.getEntrypoint(keyData)
	if err != nil {
		t.Error(err)
	}
	result, ok := g.GetResults(keyData.key)
	if !ok {
		t.Fatal("expected there to be results")
	}
	if len(result.Errors) != 1 {
		t.Error("expected Meta.Error because no app js or ts in files")
	}

	// now put app.js
	keyData = g.set(appGetData{})
	err = os.WriteFile(filepath.Join(g.AppLocation2Path.Files(""), "app.js"), []byte("app"), 0644)
	if err != nil {
		t.Error(err)
	}
	err = g.getEntrypoint(keyData)
	if err != nil {
		t.Error(err)
	}
	result, _ = g.GetResults(keyData.key)
	if len(result.Errors) != 0 {
		t.Errorf("expected no Meta.Error %v", result.Errors)
	}
	if result.VersionManifest.Entrypoint != "app.js" {
		t.Error("expected entrypoint to be app.js")
	}

	// now with both app.js and app.ts
	keyData = g.set(appGetData{})
	err = os.WriteFile(filepath.Join(g.AppLocation2Path.Files(""), "app.ts"), []byte("app"), 0644)
	if err != nil {
		t.Error(err)
	}
	err = g.getEntrypoint(keyData)
	if err != nil {
		t.Error(err)
	}
	result, _ = g.GetResults(keyData.key)
	if len(result.Errors) != 1 {
		t.Errorf("expected one Meta.Error")
	}
}
