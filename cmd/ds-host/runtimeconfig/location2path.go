package runtimeconfig

import (
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type AppLocation2Path struct {
	Config *domain.RuntimeConfig
}

// App returns the path to the App's generated files
func (l *AppLocation2Path) Base(locationKey string) string { // should this be Base?
	checkAppLocationKey(locationKey)
	return filepath.Join(l.Config.Exec.AppsPath, locationKey)
}
func (l *AppLocation2Path) Meta(locationKey string) string {
	return l.Base(locationKey)
}
func (l *AppLocation2Path) Files(locationKey string) string {
	return filepath.Join(l.Meta(locationKey), "app")
}

type AppspaceLocation2Path struct {
	Config *domain.RuntimeConfig
}

func (s *AppspaceLocation2Path) Base(locationKey string) string {
	checkAppLocationKey(locationKey)
	return filepath.Join(s.Config.Exec.AppspacesPath, locationKey)
}
func (s *AppspaceLocation2Path) Data(locationKey string) string {
	return filepath.Join(s.Base(locationKey), "data")
}
func (s *AppspaceLocation2Path) Files(locationKey string) string {
	return filepath.Join(s.Base(locationKey), "data", "files")
}
func (s *AppspaceLocation2Path) Avatars(locationKey string) string {
	return filepath.Join(s.Base(locationKey), "data", "avatars")
}
func (s *AppspaceLocation2Path) DenoDir(locationKey string) string {
	return filepath.Join(s.Base(locationKey), "deno-dir")
}

func checkAppLocationKey(loc string) {
	if loc == "" {
		panic("Trying to get location with empty location key.")
	}
}
