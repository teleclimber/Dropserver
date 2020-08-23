package main

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// DevAppModel can return a predetermined app and app version
type DevAppModel struct {
	app domain.App
	ver domain.AppVersion
}

// Set the app and app version to return for all calls
func (m *DevAppModel) Set(app domain.App, ver domain.AppVersion) {
	m.app = app
	m.ver = ver
}

// GetFromID always returns the same app
func (m *DevAppModel) GetFromID(appID domain.AppID) (*domain.App, domain.Error) {
	return &m.app, nil
}

// GetVersion always returns the same version
func (m *DevAppModel) GetVersion(appID domain.AppID, version domain.Version) (*domain.AppVersion, domain.Error) {
	return &m.ver, nil
}

// DevAppspaceModel can return an appspace struct as needed
type DevAppspaceModel struct {
	appspace domain.Appspace
}

// Set the appspace to return for all calls
func (m *DevAppspaceModel) Set(appspace domain.Appspace) {
	m.appspace = appspace
}

// GetFromSubdomain always returns the same appspace
func (m *DevAppspaceModel) GetFromSubdomain(subdomain string) (*domain.Appspace, domain.Error) {
	return &m.appspace, nil
}

// GetFromID always returns the same appspace
func (m *DevAppspaceModel) GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, domain.Error) {
	return &m.appspace, nil
}
