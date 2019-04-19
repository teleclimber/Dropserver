package runtimeconfig

import (
	"bytes"
	"testing"
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
