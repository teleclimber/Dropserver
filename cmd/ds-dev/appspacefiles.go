package main

import (
	"os"

	"github.com/otiai10/copy"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appspacefilesmodel"
)

// DevAppspaceFiles manages the appspace files as a group.
// May make it into ds-host as part of importing and exporting appsapces
type DevAppspaceFiles struct {
	AppspaceMetaDb interface {
		Create(appspaceID domain.AppspaceID, dsAPIVersion int) error
	}
	AppspaceFilesEvents interface {
		Send(domain.AppspaceID)
	}
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
		err := copy.Copy(a.sourceDir, a.destDir)
		if err != nil {
			panic(err)
		}
	} else {
		// Let's cheat for now: AppspaceFilesModel should really take the place of or be proxied by DevAppspaceFiles
		appspaceFilesModel := &appspacefilesmodel.AppspaceFilesModel{}
		appspaceFilesModel.CreateDirs(a.destDir)

		err = a.AppspaceMetaDb.Create(appspaceID, 0) // that 0 is dsAPI version. Can it stay zero in a blank appspace? Probably not?
		if err != nil {
			panic(err)
		}
	}

	a.AppspaceFilesEvents.Send(appspaceID)
}
