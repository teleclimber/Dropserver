package main

import (
	"os"
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// GetConfig returns a runtime config set up for ds-dev
func GetConfig(execPath string, appPath string, tempDir string) *domain.RuntimeConfig {

	rtc := &domain.RuntimeConfig{}
	rtc.Server.Host = "localhost"
	rtc.Server.Port = 3003
	rtc.NoTLS = true
	rtc.PortString = ":3003"

	if execPath == "" {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		execPath = filepath.Dir(ex)
	}

	rtc.Exec.GoTemplatesDir = filepath.Join(execPath, "../resources/go-templates")
	rtc.Exec.WebpackTemplatesDir = filepath.Join(execPath, "../resources/webpack-html")
	rtc.Exec.StaticAssetsDir = filepath.Join(execPath, "../static/ds-dev/")

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
	rtc.Sandbox.SocketsDir = filepath.Join(tempDir, "sockets")

	return rtc
}
