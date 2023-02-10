package runtimeconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// default config values.
var configDefault = []byte(`{
	"server": {
		"tls-port": 443,
		"http-port": 80
	},
	"external-access": {
		"scheme": "https",
		"subdomain": "dropid",
		"port": 443
	},
	"manage-certificates": {
		"issuer-endpoint": "https://acme-v02.api.letsencrypt.org/directory"
	},
	"sandbox": {
		"use-bubblewrap" : true,
		"bwrap-map-paths": ["/usr/lib", "/etc", "/lib64"],
		"use-cgroups": true,
		"cgroup-mount": "/sys/fs/cgroup",
		"memory-high-mb": 512,
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
	checkDirExists(rtc.DataDir, "data")
	for _, p := range rtc.Sandbox.BwrapMapPaths {
		checkDirExists(p, "bwrap-map-paths")
	}
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

	// do a little cleaning up on domain:
	rtc.ExternalAccess.Domain = strings.TrimSpace(rtc.ExternalAccess.Domain)

	dom := rtc.ExternalAccess.Domain
	if dom == "" {
		panic("domain can not be empty")
	}
	if strings.HasPrefix(dom, ".") {
		panic("domain can not start with a .")
	}
	if strings.HasSuffix(dom, ".") {
		panic("domain can not end with a .")
	}
	if strings.Contains(dom, "/") {
		panic("domain can not contain a /")
	}
	if strings.Contains(dom, ":") {
		panic("domain can not contain a :")
	}
	if addr := net.ParseIP(dom); addr != nil {
		panic("domain can not be an IP")
	}

	// Let's make sure there is a valid config for the server:
	if !rtc.Server.NoTLS && rtc.Server.TLSCert == "" && !rtc.ManageTLSCertificates.Enable {
		panic("config error: no TLS cert specified and TLS Certificate Management disabled")
	}
	if rtc.Server.NoTLS && rtc.Server.TLSCert != "" {
		panic("config error: server.tls-cert is specified while no-tls is set to true")
	}
	if rtc.Server.NoTLS && rtc.ManageTLSCertificates.Enable {
		panic("config error: TLS Certificate Management enabled while no-tls is set to true")
	}
	if rtc.Server.TLSCert != "" && rtc.ManageTLSCertificates.Enable {
		panic("config error: ssl-cert is set and TLS Certificate Management is enabled")
	}

	scheme := rtc.ExternalAccess.Scheme
	if scheme != "http" && scheme != "https" {
		panic("config error: external-access.scheme should be http or https")
	}
	if rtc.ExternalAccess.Domain == "" {
		panic("Domain can not be empty")
	}
	if rtc.ExternalAccess.Subdomain == "" {
		panic("Subdomain can not be empty")
	}
	if rtc.ExternalAccess.Port == 0 {
		panic("port can not be zero")
	}

	// ManageTLSCertificates
	if rtc.ManageTLSCertificates.Enable {
		m := rtc.ManageTLSCertificates
		if m.Email == "" {
			panic("Enter an email address for the ACME account for TLS management")
		}
		if m.IssuerEndpoint == "" {
			panic("An empty issuer endpoint is not allowed")
		}
	}

	// Sandbox:
	if rtc.Sandbox.SocketsDir == "" {
		panic("sockets dir can not be blank")
	}
	for i, op := range rtc.Sandbox.BwrapMapPaths {
		p := filepath.Clean(op)
		if !filepath.IsAbs(p) {
			panic(fmt.Sprintf("bwrap-map-paths invalid: %s", op))
		}
		rtc.Sandbox.BwrapMapPaths[i] = p
	}
	if rtc.Sandbox.Num == 0 {
		panic("you need at least one sandbox")
	}
}

func checkDirExists(dir string, name string) {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			panic(fmt.Sprintf("%s directory does not exist: %s", name, dir))
		} else {
			panic(err)
		}
	}
}

func setExec(rtc *domain.RuntimeConfig) {

	rtc.Exec.PortString = ""
	port := rtc.ExternalAccess.Port
	if port != 80 && port != 443 {
		rtc.Exec.PortString = fmt.Sprintf(":%d", port)
	}

	deno, err := getDenoAbsPath()
	if err != nil && rtc.Sandbox.UseBubblewrap {
		getLogger("setExec, getDenoAbsPath, Error getting deno absolute path:").Error(err)
		panic("error getting deno absolute path: " + err.Error())
	}
	rtc.Exec.DenoFullPath = deno

	// set up runtime paths
	rtc.Exec.SandboxCodePath = filepath.Join(rtc.DataDir, "sandbox-code")

	// set up user data paths:
	rtc.Exec.AppsPath = filepath.Join(rtc.DataDir, "apps")
	rtc.Exec.AppspacesPath = filepath.Join(rtc.DataDir, "appspaces")

	rtc.Exec.CertificatesPath = filepath.Join(rtc.DataDir, "certificates")

	rtc.Exec.UserRoutesDomain = rtc.ExternalAccess.Domain
	if rtc.ExternalAccess.Subdomain != "" {
		rtc.Exec.UserRoutesDomain = rtc.ExternalAccess.Subdomain + "." + rtc.ExternalAccess.Domain
	}
}

func getDenoAbsPath() (string, error) {
	fname, err := exec.LookPath("deno")
	if err != nil {
		return "", err
	}
	fname, err = filepath.Abs(fname)
	if err != nil {
		return "", err
	}
	getLogger("getDenoAbsPath").Log("Found deno abs path: " + fname)
	return fname, nil
}

func getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("runtimeconfig")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
