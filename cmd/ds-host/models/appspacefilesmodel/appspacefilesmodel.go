package appspacefilesmodel

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// performs operations on appspace files
// - backup
// - import
// - export

// Also need ability to take output of metadata taht is stored host-side
// ..if there is any such data
// ..and place it in the appspace files.

// AppspaceFilesModel is struct for appspace data directory manager
type AppspaceFilesModel struct {
	Config *domain.RuntimeConfig
}

// TODO: add appspace files event interface and call upon change.

// Probably need a create location

// CreateLocation creates a new location for an appspace
// This will need more subtlety when we import appspace files ( don't create "files", for ex)
func (a *AppspaceFilesModel) CreateLocation() (string, error) {
	err := os.MkdirAll(a.Config.Exec.AppspacesPath, 0766) // This base dir for all appspaces should probably be created at ds-host migration time
	if err != nil {
		a.getLogger("CreateLocation(), os.Mkdirall").AddNote(a.Config.Exec.AppspacesPath).Error(err)
		return "", err
	}

	appspacePath, err := ioutil.TempDir(a.Config.Exec.AppspacesPath, "as")
	if err != nil {
		a.getLogger("CreateLocation(), ioutil.TempDir").AddNote(a.Config.Exec.AppspacesPath).Error(err)
		return "", err
	}

	err = os.MkdirAll(filepath.Join(appspacePath, "files"), 0766)
	if err != nil {
		a.getLogger("CreateLocation(), os.Mkdirall for files").Error(err)
		return "", err
	}

	// Should we log when we create a location?

	return filepath.Base(appspacePath), nil
}

// Delete removes the files from the system
// func (a *AppspaceFilesModel) Delete(locationKey string) error {
// 	if !a.locationKeyExists(locationKey) {
// 		return nil //is that an error or do we consider this OK?
// 	}

// 	appsPath := a.getAppspacesPath()

// 	err := os.RemoveAll(filepath.Join(appsPath, locationKey))
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
// Do this later

func (a *AppspaceFilesModel) locationKeyExists(locationKey string) bool {
	_, err := os.Stat(filepath.Join(a.Config.Exec.AppspacesPath, locationKey))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true // OK but there could be aonther problem, like permissions out of whack?
	// Should probably log that as warning at least.
}

// pathInsidePath determines if A path is inside (contained within) path B
// func pathInsidePath(p, root string) (bool, error) {
// 	rel, err := filepath.Rel(root, p)
// 	if err != nil {
// 		return false, err // not clear that is an error that is actually an error. Errors I think just mean "can't be inside"
// 	}

// 	if strings.Contains(rel, "..") {
// 		return false, nil
// 	}
// 	return true, nil
// }
// unused here so far.

func (a *AppspaceFilesModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceFilesModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
