package main

import (
	"os"
	"path/filepath"

	"github.com/elazarl/goproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/denosandboxcode"
	"github.com/teleclimber/DropServer/internal/embedutils"
)

// copyEmbeddedFiles writes embedded files to disk
func copyEmbeddedFiles(rtc domain.RuntimeConfig) {
	err := os.RemoveAll(rtc.Exec.RuntimeFilesPath)
	if err != nil {
		panic(err)
	}

	// Remove sandbox-code from original location to keep things tidy
	// See commit 89e0d37
	err = os.RemoveAll(filepath.Join(rtc.DataDir, "sandbox-code"))
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(rtc.Exec.RuntimeFilesPath, 0744)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filepath.Join(rtc.Exec.RuntimeFilesPath, "goproxy-ca-cert.pem"), goproxy.CA_CERT, 0644)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(rtc.Exec.SandboxCodePath, 0744)
	if err != nil {
		panic(err)
	}
	err = embedutils.DirToDisk(denosandboxcode.SandboxCode, ".", rtc.Exec.SandboxCodePath)
	if err != nil {
		panic(err)
	}
}
