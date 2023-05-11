package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type AppSourceType int

const (
	Directory AppSourceType = iota
	Package
	// URL
)

func makeAbsolute(p string) string {
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	if !filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		p = filepath.Join(wd, p)
	}
	return p
}

func ResolveAppOrigin(appFlag string) AppSourceType {
	sourceType := Directory

	f, err := os.Open(appFlag)
	if err != nil {
		fmt.Println("Error opening app file/directory: " + appFlag)
		os.Exit(1)
	}
	defer f.Close()
	fInfo, err := f.Stat()
	if err != nil {
		fmt.Println("Error getting stat on app file/directory: " + appFlag)
		os.Exit(1)
	}
	if fInfo.IsDir() {
		sourceType = Directory
	} else {
		byteSlice := make([]byte, 512)
		_, err := f.Read(byteSlice)
		if err != nil {
			fmt.Println("Error reading bytes from app file: ", err)
			os.Exit(1)
		}
		contentType := http.DetectContentType(byteSlice)
		if contentType != "application/x-gzip" {
			fmt.Println("Unknown content type for application package: ", contentType)
			os.Exit(1)
		}
		sourceType = Package
	}

	return sourceType
}

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

	rtc.Exec.AppsPath = appPath

	rtc.Exec.SandboxCodePath = filepath.Join(tempDir, "sandbox-code")
	rtc.Exec.AppspacesPath = filepath.Join(tempDir, "appspace")

	return rtc
}
