package runtimeconfig

import (
	"bytes"
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

	if rtc.Exec.PublicStaticAddress != "//static.localhost:3000" {
		t.Error("pubilc assets dir not as expected", rtc.Exec)
	}
}

func TestValidateConfig(t *testing.T) {
	rtc := getPassingDefault()
	tv(t, rtc, "default", false)

	rtc = getPassingDefault()
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
func getPassingDefault() *domain.RuntimeConfig {
	rtc := loadDefault()
	rtc.DataDir = "/abc/def"
	rtc.Loki.Address = "yada"
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
