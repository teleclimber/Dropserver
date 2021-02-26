package runtimeconfig

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestLoadDefault(t *testing.T) {
	rtc := loadDefault()

	if rtc.Server.Port != 3000 {
		t.Error("port didn't register correctly. Expected 3000")
	}
}

func TestMergeLocal(t *testing.T) {
	rtc := loadDefault()

	var localJSON = bytes.NewReader([]byte(`{
		"server": {
			"port": 3999
		}
	}`))

	mergeLocal(rtc, localJSON)

	if rtc.Server.Port != 3999 {
		t.Error("port wasn't overriden by local config. Expected 3999")
	}
}

func TestSetExecValues(t *testing.T) {
	rtc := loadDefault()

	setExecValues(rtc, "/abc/def/bin/")

	if rtc.Exec.UserRoutesDomain != "dropid.localhost:3000" {
		t.Error("user routes domain not as expected", rtc.Exec)
	}
	if rtc.Exec.PublicStaticDomain != "static.localhost:3000" {
		t.Error("pubilc assets domain not as expected", rtc.Exec)
	}
}

func TestValidateHost(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	rtc := getPassingDefault(dir)
	tv(t, rtc, "default", false)

	rtc = getPassingDefault(dir)
	cases := []struct {
		host        string
		shouldPanic bool
	}{
		{"", true},
		{"abc.def", false},
		{"//abc.def", true},
		{"abc.def/", true},
		{".abc.def", true},
		{" .abc.def", true},
		{"abc.def.", true},
		{"abc.def:3000", true},
		{"10.255.5.11", true},
	}
	for _, c := range cases {
		rtc.Server.Host = c.host
		tv(t, rtc, "server host: "+c.host, c.shouldPanic)
	}

	// mostly testing fo rproblems with host validation
	// because it impacts a lot of things.
}
func TestValidateSsl(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	rtc := getPassingDefault(dir)
	tv(t, rtc, "default", false)

	rtc.Server.NoSsl = false
	tv(t, rtc, "no cert or key", true)

	rtc.Server.SslCert = "some.crt"
	rtc.Server.SslKey = "the.key"
	tv(t, rtc, "ssl with cert and key", false)
}
func getPassingDefault(dir string) *domain.RuntimeConfig {
	rtc := loadDefault()
	rtc.DataDir = dir
	rtc.Sandbox.SocketsDir = "blah"
	rtc.Server.NoSsl = true
	return rtc
}
func tv(t *testing.T, rtc *domain.RuntimeConfig, hintStr string, shouldPanic bool) {
	defer func() {
		r := recover()
		if shouldPanic && r == nil {
			t.Error(hintStr + ": should panic but didn't")
		} else if !shouldPanic && r != nil {
			t.Error(hintStr+": panic though it shouldn't have.", r)
		}
	}()
	validateConfig(rtc)

}
