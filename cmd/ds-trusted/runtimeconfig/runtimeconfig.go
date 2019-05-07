package runtimeconfig

import (
	"encoding/json"
	"io"
	"os"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// main calls Load with input file name (from cli)
// That gets parsed and put into a struct (type defined in domain)
// For now you can just return a hard coded set of values

const configFile = "/root/ds-trusted-config.json"

var configDefault = []byte(`{
	"loki": {
		"port": 3100
	}
}`)

// Load opens the json passed and merges it with defaults
func Load() *domain.TrustedConfig {

	rtc := loadDefault()

	configHandle, err := os.Open(configFile)
	if err != nil {
		panic("Could not open configuration file: " + configFile + " Error: " + err.Error())
	}
	defer configHandle.Close()

	mergeLocal(rtc, configHandle)

	validateConfig(rtc)

	return rtc
}

func loadDefault() *domain.TrustedConfig {
	var rtc domain.TrustedConfig

	err := json.Unmarshal(configDefault, &rtc)
	if err != nil {
		panic(err)
	}

	return &rtc
}

func mergeLocal(rtc *domain.TrustedConfig, handle io.Reader) {
	dec := json.NewDecoder(handle)
	err := dec.Decode(rtc)
	if err != nil {
		panic("Could not decode json in config file: " + err.Error())
	}
}

func validateConfig(rtc *domain.TrustedConfig) {
	// just panic if it fails.

	// Loki:
	if rtc.Loki.Address == "" { // would be btter if we could disable Loki
		panic("you need an address for Loki")
	}
	if rtc.Loki.Port == 0 {
		panic("you need a port for Loki")
	}
}
