package main

import "github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"

// DevAppFilesModel embeds AppFilesModel to intercept
// Read/Write Routes to avoid writing to disk
type DevAppFilesModel struct {
	appfilesmodel.AppFilesModel

	routesData []byte
}

// WriteRoutes keeps the data in memory instead of writing to disk
func (a *DevAppFilesModel) WriteRoutes(locationKey string, routesData []byte) error {
	a.routesData = routesData
	return nil
}

// ReadRoutes returns the in-memory data
func (a *DevAppFilesModel) ReadRoutes(locationKey string) ([]byte, error) {
	return a.routesData, nil
}
