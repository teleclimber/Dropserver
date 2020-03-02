package runtimeconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

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

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	setExecValues(rtc, cwd)

	return rtc
}

func setExecValues(rtc *domain.RuntimeConfig, binDir string) {
	// set up runtime paths
	rtc.Exec.GoTemplatesDir = filepath.Join(binDir, "../resources/go-templates")
	rtc.Exec.WebpackTemplatesDir = filepath.Join(binDir, "../resources/webpack-html")
	rtc.Exec.StaticAssetsDir = filepath.Join(binDir, "../static")
	rtc.Exec.JSRunnerPath = filepath.Join(binDir, "../resources/ds-sandbox-runner.js") //todo: rename because it won't be JS one day
	rtc.Exec.MigratorScriptPath = filepath.Join(binDir, "../resources/ds-appspace-migrator.js")

	// set up user data paths:
	rtc.Exec.AppsPath = filepath.Join(rtc.DataDir, "apps")
	rtc.Exec.AppspacesMetaPath = filepath.Join(rtc.DataDir, "appspaces-meta")
	rtc.Exec.AppspacesFilesPath = filepath.Join(rtc.DataDir, "appspaces-files")

	//  subdomain sorting out:
	host := rtc.Server.Host
	port := rtc.Server.Port
	if port != 80 && port != 443 {
		host += fmt.Sprintf(":%d", port)
	}
	rtc.Exec.PublicStaticAddress = "//static." + host
	rtc.Exec.UserRoutesAddress = "//user." + host
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
