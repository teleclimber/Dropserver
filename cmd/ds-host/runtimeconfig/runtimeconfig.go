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

var configDefault = []byte(`{
	"server": {
		"port": 3000,
		"host": "localhost"
	},
	"sandbox": {
		"num": 3
	},
	"loki": {
		"port": 3100
	},
	"prometheus": {
		"port": 2112
	}
}`)

// Load opens the json passed and merges it with defaults
func Load(configFile string) *domain.RuntimeConfig {

	rtc := loadDefault()

	// load JSON, and merge it in simply by passing rtc again
	if configFile != "" {
		configHandle, err := os.Open(configFile)
		if err != nil {
			panic("Could not open configuration file: " + configFile + " Error: " + err.Error())
		}
		defer configHandle.Close()

		mergeLocal(rtc, configHandle)
	}

	validateConfig(rtc)

	return rtc
}

func loadDefault() *domain.RuntimeConfig {
	var rtc domain.RuntimeConfig

	err := json.Unmarshal(configDefault, &rtc)
	if err != nil {
		panic(err)
	}

	return &rtc
}

func mergeLocal(rtc *domain.RuntimeConfig, handle io.Reader) {
	dec := json.NewDecoder(handle)
	err := dec.Decode(rtc)
	if err != nil {
		panic("Could not decode json in config file: " + err.Error())
	}
}

func validateConfig(rtc *domain.RuntimeConfig) {
	// just panic if it fails.

	if rtc.DataDir == "" {
		panic("You need to specify a data directory")
	}

	// Server:
	if rtc.Server.Port == 0 {
		panic("Server.port can not be 0")
	}

	host := rtc.Server.Host
	if host == "" {
		panic("host can not be empty")
	}

	// need more validation for host names...

	// Sandbox:
	if rtc.Sandbox.Num == 0 {
		panic("you need at least one sandbox")
	}

	// Loki:
	if rtc.Loki.Address == "" { // would be btter if we could disable Loki
		panic("you need an address for Loki")
	}
	if rtc.Loki.Port == 0 {
		panic("you need a port for Loki")
	}

	// Prometheus:
	if rtc.Prometheus.Port == 0 {
		panic("Prometheus port can not be 0")
	}
}
