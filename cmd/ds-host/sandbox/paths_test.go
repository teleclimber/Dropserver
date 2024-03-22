package sandbox

import (
	"reflect"
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/runtimeconfig"
)

// Note we're using Location2Path from runtimeconfig here

func TestAppImportMapPath(t *testing.T) {
	c := &domain.RuntimeConfig{}
	c.Exec.AppsPath = "/app-f1les-base"
	p := &paths{
		appLoc:           "av-loc-77",
		Config:           c,
		AppLocation2Path: &runtimeconfig.AppLocation2Path{Config: c},
	}
	p.init()

	assertEqStr(t, "/app-f1les-base/av-loc-77/import-paths.json", p.hostPath("import-map"))
	assertEqStr(t, "/app-f1les-base/av-loc-77/import-paths.json", p.sandboxPath("import-map"))

	c.Sandbox.UseBubblewrap = true
	assertEqStr(t, "/import-paths.json", p.sandboxPath("import-map"))
}

func TestAppspaceImportMapPath(t *testing.T) {
	c := &domain.RuntimeConfig{}
	c.Exec.AppsPath = "/app-f1les-base"
	c.Exec.AppspacesPath = "/appspace-f1les-base"
	p := &paths{
		appLoc:                "av-loc-77",
		appspaceLoc:           "as-loc-13",
		Config:                c,
		AppLocation2Path:      &runtimeconfig.AppLocation2Path{Config: c},
		AppspaceLocation2Path: &runtimeconfig.AppspaceLocation2Path{Config: c},
	}
	p.init()

	assertEqStr(t, "/appspace-f1les-base/as-loc-13/import-paths.json", p.hostPath("import-map"))
	assertEqStr(t, "/appspace-f1les-base/as-loc-13/import-paths.json", p.sandboxPath("import-map"))

	c.Sandbox.UseBubblewrap = true
	assertEqStr(t, "/import-paths.json", p.sandboxPath("import-map"))
}

func TestAppDenoAllow(t *testing.T) {
	c := &domain.RuntimeConfig{}
	c.Exec.AppsPath = "/app-f1les-base"
	p := &paths{
		appLoc:           "av-loc-77",
		sockets:          "/s0ckets/",
		Config:           c,
		AppLocation2Path: &runtimeconfig.AppLocation2Path{Config: c},
	}
	p.init()

	assertEqStr(t, "/s0ckets/,/app-f1les-base/av-loc-77/app/", p.denoAllowRead())
	assertEqStr(t, "/s0ckets/", p.denoAllowWrite())

	c.Sandbox.UseBubblewrap = true
	assertEqStr(t, "/sockets/,/app-files/", p.denoAllowRead())
	assertEqStr(t, "/sockets/", p.denoAllowWrite())
}

func TestAppspaceDenoAllow(t *testing.T) {
	c := &domain.RuntimeConfig{}
	c.Exec.AppsPath = "/app-f1les-base"
	c.Exec.AppspacesPath = "/appspace-f1les-base"
	p := &paths{
		appLoc:                "av-loc-77",
		appspaceLoc:           "as-loc-13",
		sockets:               "/s0ckets/",
		Config:                c,
		AppLocation2Path:      &runtimeconfig.AppLocation2Path{Config: c},
		AppspaceLocation2Path: &runtimeconfig.AppspaceLocation2Path{Config: c},
	}
	p.init()

	expected := []string{
		"/s0ckets/",
		"/app-f1les-base/av-loc-77/app/",
		"/appspace-f1les-base/as-loc-13/data/files/",
		"/appspace-f1les-base/as-loc-13/data/avatars/",
	}
	assertEqStr(t, strings.Join(expected, ","), p.denoAllowRead())
	expected = []string{
		"/s0ckets/",
		"/appspace-f1les-base/as-loc-13/data/files/",
	}
	assertEqStr(t, strings.Join(expected, ","), p.denoAllowWrite())

	c.Sandbox.UseBubblewrap = true
	expected = []string{
		"/sockets/",
		"/app-files/",
		"/appspace-data/files/",
		"/appspace-data/avatars/",
	}
	assertEqStr(t, strings.Join(expected, ","), p.denoAllowRead())
	expected = []string{
		"/sockets/",
		"/appspace-data/files/",
	}
	assertEqStr(t, strings.Join(expected, ","), p.denoAllowWrite())
}

func TestAppBwrapMaps(t *testing.T) {
	c := &domain.RuntimeConfig{}
	c.Sandbox.UseBubblewrap = true
	c.Exec.DenoFullPath = "/path/to/deno"
	c.Exec.AppsPath = "/app-f1les-base"
	c.Exec.SandboxCodePath = "/sandb0x-runner"
	p := &paths{
		appLoc:           "av-loc-77",
		sockets:          "/s0ckets/",
		Config:           c,
		AppLocation2Path: &runtimeconfig.AppLocation2Path{Config: c},
	}
	p.init()

	expected := []string{
		"--ro-bind", "/path/to/deno", "/deno",
		"--ro-bind", "/app-f1les-base/av-loc-77/bootstrap.js", "/bootstrap.js",
		"--ro-bind", "goproxy-ca-cert.pem", "/goproxy-ca-cert.pem",
		"--ro-bind", "/sandb0x-runner/", "/deno-sandbox-runner/",
		"--bind", "/s0ckets/", "/sockets/",
		"--ro-bind", "/app-f1les-base/av-loc-77/app/", "/app-files/",
		"--ro-bind", "/app-f1les-base/av-loc-77/import-paths.json", "/import-paths.json",
		"--bind", "/app-f1les-base/av-loc-77/deno-dir", "/deno-dir/",
	}
	actual := p.getBwrapPathMaps()
	if !reflect.DeepEqual(expected, actual) {
		t.Log("\nexpected:\n", strings.Join(expected, "\n"), "\nactual:\n", strings.Join(actual, "\n"))
		t.Error("expected and actual different")
	}
}

func TestAppspaceBwrapMaps(t *testing.T) {
	c := &domain.RuntimeConfig{}
	c.Sandbox.UseBubblewrap = true
	c.Exec.DenoFullPath = "/path/to/deno"
	c.Exec.AppsPath = "/app-f1les-base"
	c.Exec.AppspacesPath = "/appspace-f1les-base"
	c.Exec.SandboxCodePath = "/sandb0x-runner"
	p := &paths{
		appLoc:                "av-loc-77",
		appspaceLoc:           "as-loc-13",
		sockets:               "/s0ckets/",
		Config:                c,
		AppLocation2Path:      &runtimeconfig.AppLocation2Path{Config: c},
		AppspaceLocation2Path: &runtimeconfig.AppspaceLocation2Path{Config: c},
	}
	p.init()

	expected := []string{
		"--ro-bind", "/path/to/deno", "/deno",
		"--ro-bind", "/app-f1les-base/av-loc-77/bootstrap.js", "/bootstrap.js",
		"--ro-bind", "goproxy-ca-cert.pem", "/goproxy-ca-cert.pem",
		"--ro-bind", "/sandb0x-runner/", "/deno-sandbox-runner/",
		"--bind", "/s0ckets/", "/sockets/",
		"--ro-bind", "/app-f1les-base/av-loc-77/app/", "/app-files/",
		"--ro-bind", "/appspace-f1les-base/as-loc-13/import-paths.json", "/import-paths.json",
		"--bind", "/appspace-f1les-base/as-loc-13/deno-dir", "/deno-dir/",
		"--bind", "/appspace-f1les-base/as-loc-13/data/files/", "/appspace-data/files/",
		"--ro-bind", "/appspace-f1les-base/as-loc-13/data/avatars/", "/appspace-data/avatars/",
	}
	actual := p.getBwrapPathMaps()
	if !reflect.DeepEqual(expected, actual) {
		//t.Log("\nexpected:\n", strings.Join(expected, "\n"), "\nactual:\n", strings.Join(actual, "\n"))
		for i, e := range expected {
			a := actual[i]
			if e != a {
				t.Logf("Different: actual %s, expected %s", a, e)
			}
		}
		t.Error("expected and actual different")
	}
}

func assertEqStr(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("expected: %s actual: %s", expected, actual)
	}
}
