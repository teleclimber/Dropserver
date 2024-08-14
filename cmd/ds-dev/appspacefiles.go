package main

import (
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacefilesmodel"
)

// DevAppspaceFiles manages the appspace files as a group.
// May make it into ds-host as part of importing and exporting appsapces
type DevAppspaceFiles struct {
	AppspaceMetaDb interface {
		Create(appspaceID domain.AppspaceID) error
		Migrate(appspaceID domain.AppspaceID) error
	} `checkinject:"required"`
	AppspaceFilesEvents interface {
		Send(domain.AppspaceID)
	} `checkinject:"required"`
	sourceDir string
	destDir   string
}

// Reset recreates the appspace files directory
func (a *DevAppspaceFiles) Reset() {
	err := os.RemoveAll(a.destDir)
	if err != nil {
		panic(err)
	}

	if a.sourceDir != "" {
		// Copy appspace files
		err := copy.Copy(a.sourceDir, filepath.Join(a.destDir, "data"))
		if err != nil {
			panic(err)
		}
		// After copying, migrate appspace meta DB.
		// It will no-op if it's unnecessary
		err = a.AppspaceMetaDb.Migrate(appspaceID)
		if err != nil {
			panic(err)
		}
	} else {
		// Let's cheat for now: AppspaceFilesModel should really take the place of or be proxied by DevAppspaceFiles
		appspaceFilesModel := &appspacefilesmodel.AppspaceFilesModel{}
		appspaceFilesModel.CreateDirs(a.destDir)

		err = a.AppspaceMetaDb.Create(appspaceID)
		if err != nil {
			panic(err)
		}
	}

	a.AppspaceFilesEvents.Send(appspaceID)
}
