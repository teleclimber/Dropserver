package main

import (
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type AppLocation2Path struct {
	AppMetaDir string
	Config     *domain.RuntimeConfig `checkinject:"required"`
}

// App return the app files directory as well
// though it could return a temporary directory
func (l *AppLocation2Path) Meta(locationKey string) string { // this is now Base, but now I see Meta has meaning in ds-dev?
	return l.AppMetaDir
}

func (l *AppLocation2Path) Base(locationKey string) string { // unsure if this should be apps dir or meta?
	return l.Config.Exec.AppsPath
}

// AppFiles in ds-dev returns the path of the app
func (l *AppLocation2Path) Files(locationKey string) string {
	return l.Config.Exec.AppsPath
}
func (s *AppLocation2Path) DenoDir(locationKey string) string {
	return filepath.Join(s.Meta(locationKey), "deno-dir")
}

type AppspaceLocation2Path struct {
	Config *domain.RuntimeConfig
}

func (s *AppspaceLocation2Path) Base(locationKey string) string {
	checkNotEmpty(locationKey, "appspace base")
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
func (s *AppspaceLocation2Path) Avatar(locationKey string, avatar string) string {
	return filepath.Join(s.Base(locationKey), "data", "avatars", avatar)
}
func (s *AppspaceLocation2Path) DenoDir(locationKey string) string {
	return filepath.Join(s.Base(locationKey), "deno-dir")
}

func checkNotEmpty(loc string, desc string) {
	if loc == "" {
		panic("checkNotEmpty: " + desc)
	}
}
