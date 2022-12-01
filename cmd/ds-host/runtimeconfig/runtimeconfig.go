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
		"tls-port": 5050,
		"host": "localhost"
	},
	"port-string": ":5050",
	"subdomains": {
		"user-accounts": "dropid",
		"static-assets": "static"
	},
	"sandbox": {
		"num": 3,
		"use-cgroups": true,
		"cgroup-mount": "/sys/fs/cgroup",
		"memory-high-mb": 512
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
	checkDataDir(rtc.DataDir)
	setExec(rtc)

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
	// TODO verify path is absolute, convert to abs?

	// Server:
	if rtc.Server.HTTPPort == 0 {
		rtc.Server.HTTPPort = 80
	}
	if rtc.Server.TLSPort == 0 {
		rtc.Server.TLSPort = 443
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
		panic("host can not contain a :")
	}
	if addr := net.ParseIP(host); addr != nil {
		panic("host can not be an IP")
	}

	// Let's make sure there is a valid config for the server:
	if !rtc.Server.NoTLS && rtc.Server.SslCert == "" && !rtc.ManageTLSCertificates.Enable {
		panic("config error: no TLS cert specified and TLS Certificate Management disabled")
	}
	if rtc.Server.NoTLS && rtc.Server.SslCert != "" {
		panic("config error: server.ssl-cert is specified while no-tls is set to true")
	}
	if rtc.Server.NoTLS && rtc.ManageTLSCertificates.Enable {
		panic("config error: TLS Certificate Management enabled while no-tls is set to true")
	}
	if rtc.Server.SslCert != "" && rtc.ManageTLSCertificates.Enable {
		panic("config error: ssl-cert is set and TLS Certificate Management is enabled")
	}

	// ManageTLSCertificates
	if rtc.ManageTLSCertificates.Enable {
		m := rtc.ManageTLSCertificates
		if m.Email == "" {
			panic("enter an email address for the ACME account for TLS management")
		}
		// validate that the issuer endpoint looks like a URL?
	}

	// Sandbox:
	if rtc.Sandbox.Num == 0 {
		panic("you need at least one sandbox")
	}
	if rtc.Sandbox.SocketsDir == "" {
		panic("sockets dir can not be blank")
	}
}

func checkDataDir(dir string) {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			panic("data directory does not exist: " + dir)
		} else {
			panic(err)
		}
	}
}

func setExec(rtc *domain.RuntimeConfig) {
	// set up runtime paths
	rtc.Exec.SandboxCodePath = filepath.Join(rtc.DataDir, "sandbox-code")

	// set up user data paths:
	rtc.Exec.AppsPath = filepath.Join(rtc.DataDir, "apps")
	rtc.Exec.AppspacesPath = filepath.Join(rtc.DataDir, "appspaces")

	rtc.Exec.CertificatesPath = filepath.Join(rtc.DataDir, "certificates")

	rtc.Exec.UserRoutesDomain = rtc.Server.Host
	if rtc.Subdomains.UserAccounts != "" {
		rtc.Exec.UserRoutesDomain = rtc.Subdomains.UserAccounts + "." + rtc.Server.Host
	}
}
