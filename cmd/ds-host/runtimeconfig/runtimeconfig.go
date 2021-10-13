package runtimeconfig

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// default config values.
// Fake paths are set for SSL, forcing user to either override to make non-ssl,
// or set the correct paths.
var configDefault = []byte(`{
	"server": {
		"port": 5050,
		"host": "localhost",
		"ssl-cert": "/path/to/ssl-cert.pem",
		"ssl-key": "/path/to/ssl-key.pem"
	},
	"port-string": ":5050",
	"subdomains": {
		"user-accounts": "dropid",
		"static-assets": "static"
	},
	"sandbox": {
		"num": 3
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

	// set up runtime paths
	rtc.Exec.SandboxCodePath = filepath.Join(rtc.DataDir, "sandbox-code")

	// set up user data paths:
	rtc.Exec.AppsPath = filepath.Join(rtc.DataDir, "apps")
	rtc.Exec.AppspacesPath = filepath.Join(rtc.DataDir, "appspaces")

	rtc.Exec.UserRoutesDomain = rtc.Server.Host
	if rtc.Subdomains.UserAccounts != "" {
		rtc.Exec.UserRoutesDomain = rtc.Subdomains.UserAccounts + "." + rtc.Server.Host
	}

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
	_, err := os.Stat(rtc.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			panic("data directory does not exist: " + rtc.DataDir)
		} else {
			panic(err)
		}
	}

	// Server:
	if rtc.Server.Port == 0 {
		panic("Server.port can not be 0")
	}

	// do a little cleaning up on host:
	rtc.Server.Host = strings.TrimSpace(rtc.Server.Host)

	host := rtc.Server.Host
	if host == "" {
		panic("host can not be empty")
	}
	if strings.HasPrefix(host, ".") {
		panic("host can not start with a .")
	}
	if strings.HasSuffix(host, ".") {
		panic("host can not end with a .")
	}
	if strings.Contains(host, "/") {
		panic("host can not contain a /")
	}
	if strings.Contains(host, ":") {
		panic("host can not contain a :.")
	}
	if addr := net.ParseIP(host); addr != nil {
		panic("host can not be an IP")
	}

	// Sandbox:
	if rtc.Sandbox.Num == 0 {
		panic("you need at least one sandbox")
	}
	if rtc.Sandbox.SocketsDir == "" {
		panic("sockets dir can not be blank")
	}
}

// getters:
