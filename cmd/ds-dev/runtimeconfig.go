package main

import (
	"os"
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// GetConfig returns a runtime config set up for ds-dev
func GetConfig(appPath string, tempDir string) *domain.RuntimeConfig {

	rtc := &domain.RuntimeConfig{}
	rtc.Server.HTTPPort = 3003
	rtc.Server.NoTLS = true

	rtc.ExternalAccess.Scheme = "http"
	rtc.ExternalAccess.Domain = "localhost"
	rtc.ExternalAccess.Port = 3003

	rtc.Sandbox.SocketsDir = filepath.Join(tempDir, "sockets")

	rtc.Exec.PortString = ":3003"

	if !filepath.IsAbs(appPath) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		appPath = filepath.Join(wd, appPath)
	}
	rtc.Exec.AppsPath = appPath

	rtc.Exec.SandboxCodePath = filepath.Join(tempDir, "sandbox-code")
	rtc.Exec.AppspacesPath = filepath.Join(tempDir, "appspace")

	return rtc
}
