package main

import (
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type Location2Path struct {
	Config *domain.RuntimeConfig
}

// App return the app files directory as well
// though it could return a temporary directory
func (l *Location2Path) App(locationKey string) string {
	return filepath.Join(l.Config.Exec.AppsPath)
}

// AppFiles in des-dev returns the path of the app
func (l *Location2Path) AppFiles(locationKey string) string {
	return l.Config.Exec.AppsPath
}
