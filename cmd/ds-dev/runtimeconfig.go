package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// GetConfig returns a runtime config set up for ds-dev
func GetConfig(execPath string, appPath string, appspacePath string) *domain.RuntimeConfig {

	rtc := &domain.RuntimeConfig{}
	rtc.Server.Host = "localhost"
	rtc.Server.Port = 3003
	rtc.Server.NoSsl = true

	if execPath == "" {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		execPath = filepath.Dir(ex)
	}

	setExecValues(rtc, execPath)

	rtc.Exec.AppsPath = appPath
	rtc.Exec.AppspacesPath = appspacePath

	return rtc
}

func setExecValues(rtc *domain.RuntimeConfig, binDir string) {
	rtc.Exec.GoTemplatesDir = filepath.Join(binDir, "../resources/go-templates")
	rtc.Exec.WebpackTemplatesDir = filepath.Join(binDir, "../resources/webpack-html")
	rtc.Exec.StaticAssetsDir = filepath.Join(binDir, "../static/ds-dev/")
	rtc.Exec.SandboxCodePath = filepath.Join(binDir, "../resources/")
	rtc.Exec.SandboxRunnerPath = filepath.Join(binDir, "../resources/ds-sandbox-runner.ts")
	// UserRoutesAddress has to be a bit different from what it is on ds-host
	rtc.Exec.PortString = fmt.Sprintf(":%v", rtc.Server.Port)
	//rtc.Exec.UserRoutesDomain = fmt.Sprintf("%v:%v/dropserver-dev", rtc.Server.Host, "")
	//^^^^ this is going to be a problem. Maybe serve user routes on a different port?

	// Sockets?
}