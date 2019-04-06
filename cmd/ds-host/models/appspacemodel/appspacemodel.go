package appspacemodel

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// inject DB, logger, ...

// AppspaceModel represents the model for app spaces
type AppspaceModel struct {
	appspaces map[string]domain.Appspace
}

// NewAppspaceModel creates and initializes AppspaceModel
func NewAppspaceModel() *AppspaceModel {
	return &AppspaceModel{
		appspaces: make(map[string]domain.Appspace),
	}
}

// GetForName returns the apps space data by app space name
func (m *AppspaceModel) GetForName(appspaceName string) (appspace *domain.Appspace, ok bool) {
	as, ok := m.appspaces[appspaceName]

	if ok {
		return &as, ok
	}
	return nil, false
}

// GetForUser...


// Create adds an appspace to the database
func (m *AppspaceModel) Create(appspace *domain.Appspace) {
	m.appspaces[appspace.Name] = *appspace
}

