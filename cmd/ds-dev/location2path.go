package main

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type Location2Path struct {
	AppMetaDir string
	Config     *domain.RuntimeConfig `checkinject:"required"`
}

// App return the app files directory as well
// though it could return a temporary directory
func (l *Location2Path) AppMeta(locationKey string) string {
	return l.AppMetaDir
}

// AppFiles in des-dev returns the path of the app
func (l *Location2Path) AppFiles(locationKey string) string {
	return l.Config.Exec.AppsPath
}
