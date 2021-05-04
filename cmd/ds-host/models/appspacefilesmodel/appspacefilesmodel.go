package appspacefilesmodel

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

// dir structure:
// - asXYZLOCATION/
//   - backups/
//   - data/
//     - files/
//     - logs/
//     - dbs/ ? We should consider putting appspace dbs in the regular data files?
//     - appspacemeta.db
//     - [appspace-export.json]
//   - import-paths.json

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

	err = a.CreateDirs(appspacePath)
	if err != nil {
		return "", err
	}

	// TODO should create ocation create the subdirectories?

	// Should we log when we create a location?

	return filepath.Base(appspacePath), nil
}

func (a *AppspaceFilesModel) CreateDirs(base string) error {
	err := os.MkdirAll(filepath.Join(base, "backups"), 0766)
	if err != nil {
		a.getLogger("CreateDirs(), os.Mkdirall for backups").Error(err)
		return err
	}

	err = os.MkdirAll(filepath.Join(base, "data", "files"), 0766)
	if err != nil {
		a.getLogger("CreateDirs(), os.Mkdirall for files").Error(err)
		return err
	}

	err = os.MkdirAll(filepath.Join(base, "data", "logs"), 0766)
	if err != nil {
		a.getLogger("CreateDirs(), os.Mkdirall for logs").Error(err)
		return err
	}

	err = os.MkdirAll(filepath.Join(base, "data", "dbs"), 0766)
	if err != nil {
		a.getLogger("CreateDirs(), os.Mkdirall for dbs").Error(err)
		return err
	}

	return nil
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

// GetBackups retuns list of backup files at location key
// Should at least include the file size for each.
func (a *AppspaceFilesModel) GetBackups(locationKey string) ([]string, error) {
	if !a.locationKeyExists(locationKey) {
		return []string{}, errors.New("location key does not exist")
	}

	f, err := os.Open(filepath.Join(a.Config.Exec.AppspacesPath, locationKey, "backups"))
	if err != nil {
		a.getLogger("GetBackups, os.Open").Error(err)
		return []string{}, err
	}

	entries, err := f.Readdirnames(-1)
	if err != nil {
		a.getLogger("GetBackups, f.Readdirnames").Error(err)
		return []string{}, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i] > entries[j]
	})

	return entries, nil
}

func (a *AppspaceFilesModel) DeleteBackup(locationKey string, filename string) error {
	if !a.locationKeyExists(locationKey) {
		return errors.New("location key does not exist")
	}

	// let's validate that file name again
	err := validator.AppspaceBackupFile(filename)
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(a.Config.Exec.AppspacesPath, locationKey, "backups", filename))

	return err
}

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
