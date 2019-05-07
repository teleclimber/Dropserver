package runtimeconfig

import (
	"bytes"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	rtc := loadDefault()

	if rtc.Loki.Port != 3100 {
		t.Error("port didn't register correctly. Expected 3000")
	}
}

func TestMergeLocal(t *testing.T) {
	rtc := loadDefault()

	var localJSON = bytes.NewReader([]byte(`{
		"loki": {
			"port": 3999
		}
	}`))

	mergeLocal(rtc, localJSON)

	if rtc.Loki.Port != 3999 {
		t.Error("port wasn't overriden by local config. Expected 3999")
	}
}
