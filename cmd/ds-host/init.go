package main

import (
	"os"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/denosandboxcode"
	"github.com/teleclimber/DropServer/internal/embedutils"
)

// copyEmbeddedFiles writes embedded files to disk
func copyEmbeddedFiles(rtc domain.RuntimeConfig) {
	err := os.RemoveAll(rtc.Exec.SandboxCodePath)
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
