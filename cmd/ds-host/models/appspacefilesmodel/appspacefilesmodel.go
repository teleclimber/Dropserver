package appspacefilesmodel

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// performs operations on appspace files

// AppspaceFilesModel is struct for application files manager
type AppspaceFilesModel struct {
	Config *domain.RuntimeConfig
}

// Probably need a create location

// CreateLocation creates a new location for an appspace
func (a *AppspaceFilesModel) CreateLocation() (string, domain.Error) {
	appspacesPath := a.getAppspacesPath()

	err := os.MkdirAll(appspacesPath, 0766)
	if err != nil {
		a.getLogger("CreateLocation(), os.Mkdirall").AddNote(appspacesPath).Error(err)
		return "", dserror.New(dserror.InternalError)
	}

	appspacePath, err := ioutil.TempDir(appspacesPath, "as")
	if err != nil {
		a.getLogger("CreateLocation(), ioutil.TempDir").AddNote(appspacesPath).Error(err)
		return "", dserror.New(dserror.InternalError)
	}

	// Should we log when we create a location?

	return filepath.Base(appspacePath), nil
}

// Delete removes the files from the system
// func (a *AppspaceFilesModel) Delete(locationKey string) domain.Error {
// 	if !a.locationKeyExists(locationKey) {
// 		return nil //is that an error or do we consider this OK?
// 	}

// 	appsPath := a.getAppspacesPath()

// 	err := os.RemoveAll(filepath.Join(appsPath, locationKey))
// 	if err != nil {
// 		return dserror.FromStandard(err)
// 	}

// 	return nil
// }
// Do this later

func (a *AppspaceFilesModel) locationKeyExists(locationKey string) bool {
	appsPath := a.getAppspacesPath()
	_, err := os.Stat(filepath.Join(appsPath, locationKey))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true // OK but there could be aonther problem, like permissions out of whack?
	// Should probably log that as warning at least.
}

func (a *AppspaceFilesModel) getAppspacesPath() string {
	return filepath.Join(a.Config.DataDir, "appspaces") //TODO haven't we moved that over to Config?
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
