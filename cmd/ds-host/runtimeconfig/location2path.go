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
	checkNotEmpty(locationKey, "app locationkey")
	return filepath.Join(l.Config.Exec.AppsPath, locationKey)
}
func (l *AppLocation2Path) Meta(locationKey string) string {
	return l.Base(locationKey)
}
func (l *AppLocation2Path) Files(locationKey string) string {
	return filepath.Join(l.Meta(locationKey), "app")
}
func (s *AppLocation2Path) DenoDir(locationKey string) string {
	return filepath.Join(s.Meta(locationKey), "deno-dir")
}

type AppspaceLocation2Path struct {
	Config *domain.RuntimeConfig
}

func (s *AppspaceLocation2Path) Base(locationKey string) string {
	checkNotEmpty(locationKey, "appspace locationKey")
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
	checkNotEmpty(avatar, "avatar")
	return filepath.Join(s.Base(locationKey), "data", "avatars", avatar)
}
func (s *AppspaceLocation2Path) Backups(locationKey string) string {
	return filepath.Join(s.Base(locationKey), "backups")
}
func (s *AppspaceLocation2Path) Backup(locationKey string, backupFile string) string {
	checkNotEmpty(backupFile, "backup file")
	return filepath.Join(s.Backups(locationKey), backupFile)
}
func (s *AppspaceLocation2Path) DenoDir(locationKey string) string {
	return filepath.Join(s.Base(locationKey), "deno-dir")
}
func (s *AppspaceLocation2Path) TailscaleNodeStore(locationKey string) string {
	return filepath.Join(s.Base(locationKey), "tailscale-store")
}

func checkNotEmpty(loc string, desc string) {
	if loc == "" {
		panic("checkNotEmpty: " + desc)
	}
}
