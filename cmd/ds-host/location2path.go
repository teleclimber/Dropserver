package main

import (
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type Location2Path struct {
	Config domain.RuntimeConfig
}

// App returns the path to the App's generated files
func (l *Location2Path) AppMeta(locationKey string) string {
	return filepath.Join(l.Config.Exec.AppsPath, locationKey)
}

func (l *Location2Path) AppFiles(locationKey string) string {
	return filepath.Join(l.Config.Exec.AppsPath, locationKey, "app")
}
