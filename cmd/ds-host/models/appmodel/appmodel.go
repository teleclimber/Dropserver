package appmodel

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// inject DB, logger, ...

// AppModel represents the model for app spaces
type AppModel struct {
	apps map[string]domain.App
}

// NewAppModel creates and initializes AppModel
func NewAppModel() *AppModel {
	return &AppModel{
		apps: make(map[string]domain.App),
	}
}

// GetForName returns the apps space data by app space name
func (m *AppModel) GetForName(appName string) (app *domain.App, ok bool) {
	a, ok := m.apps[appName]

	if ok {
		return &a, ok
	}
	return nil, false
}

// GetForUser


// Create adds an appspace to the database
func (m *AppModel) Create(app *domain.App) {
	m.apps[app.Name] = *app
}

