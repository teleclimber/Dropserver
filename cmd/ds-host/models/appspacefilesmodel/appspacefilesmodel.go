package appspacefilesmodel

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/otiai10/copy"

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
//     - avatars (readonly from sandbox?)
//     - appspacemeta.db
//     - [appspace-export.json]
//   - import-paths.json

// AppspaceFilesModel is struct for appspace data directory manager
type AppspaceFilesModel struct {
	Config              *domain.RuntimeConfig `checkinject:"required"`
	AppspaceFilesEvents interface {
		Send(appspaceID domain.AppspaceID)
	} `checkinject:"required"`
}

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

	return filepath.Base(appspacePath), nil
}

func (a *AppspaceFilesModel) DeleteLocation(loc string) error {
	// safety checks on that location please.

	err := validator.LocationKey(loc)
	if err != nil {
		err := errors.New("invalid location key")
		a.getLogger("DeleteLocation()").AddNote(a.Config.Exec.AppspacesPath).Error(err)
		return err
	}

	fullPath := filepath.Join(a.Config.Exec.AppspacesPath, loc)

	// The below check may be overkill givsn the location string checks above?
	rel, err := filepath.Rel(a.Config.Exec.AppspacesPath, fullPath)
	if err != nil {
		a.getLogger("DeleteLocation(), filepath.Rel").AddNote(a.Config.Exec.AppspacesPath).Error(err)
		return err
	}
	if strings.HasPrefix(rel, "..") {
		err = errors.New("invalid location key")
		a.getLogger("DeleteLocation()").AddNote(a.Config.Exec.AppspacesPath).Error(err)
		return err
	}

	err = os.RemoveAll(fullPath)
	if err != nil {
		a.getLogger("DeleteLocation(), os.RemoveAll").AddNote(a.Config.Exec.AppspacesPath).Error(err)
		return err
	}

	return nil
}

// CreateDirs creates an appspace directory structure
// and also creates the files necessary for a new appspace
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

	// let's also create an empty log file to prevent log errors:
	logFile := filepath.Join(base, "data", "logs", "log.txt")
	file, err := os.Create(logFile)
	if err != nil {
		a.getLogger("CreateDirs(), os.Create(logFile)").Error(err)
		return err
	}
	file.Close()

	err = os.MkdirAll(filepath.Join(base, "data", "dbs"), 0766)
	if err != nil {
		a.getLogger("CreateDirs(), os.Mkdirall for dbs").Error(err)
		return err
	}

	err = os.MkdirAll(filepath.Join(base, "data", "avatars"), 0766)
	if err != nil {
		a.getLogger("CreateDirs(), os.Mkdirall for avatars").Error(err)
		return err
	}

	return nil
}

var expectedFiles = []string{
	"appspace-meta.db",
	"avatars",
	"dbs",
	"files",
	"logs"}

// badZip implements domain.BadRestoreZip interface
type badZip struct {
	missingFiles []string
	zipFiles     []string
}

func (b *badZip) Error() string {
	errStr := "Files or directories missing from appspace data: "
	for _, m := range b.missingFiles {
		errStr += m + " "
	}
	errStr += "Files found: "
	for _, f := range b.zipFiles {
		errStr += f + " "
	}
	return errStr
}
func (b *badZip) MissingFiles() []string {
	return b.missingFiles
}
func (b *badZip) ZipFiles() []string {
	return b.zipFiles
}

//CheckDataFiles verifies that the directories and files that
// we expect to see in an appspace data dir are present.
func (a *AppspaceFilesModel) CheckDataFiles(dataDir string) error {
	files, err := ioutil.ReadDir(dataDir)
	if err != nil {
		return err
	}

	dataFiles := make([]string, 0)
	for _, file := range files {
		fName := file.Name()
		if fName != "." && fName != ".." {
			dataFiles = append(dataFiles, fName)
		}
	}

	missingFiles := make([]string, 0)
	for _, ef := range expectedFiles {
		found := false
		for _, df := range dataFiles {
			if ef == df {
				found = true
				break
			}
		}
		if !found {
			missingFiles = append(missingFiles, ef)
		}
	}

	if len(missingFiles) > 0 {
		return &badZip{
			missingFiles: missingFiles,
			zipFiles:     dataFiles}
	}

	return nil
}

func (a *AppspaceFilesModel) ReplaceData(appspace domain.Appspace, source string) error {
	// validate appspace location since we're deleting stuff.
	err := validator.LocationKey(appspace.LocationKey)
	if err != nil {
		err := errors.New("invalid location key")
		a.getLogger("ReplaceData(), validator.LocationKey").Error(err)
		return err
	}

	defer a.AppspaceFilesEvents.Send(appspace.AppspaceID)

	dataDir := filepath.Join(a.Config.Exec.AppspacesPath, appspace.LocationKey, "data")
	err = os.RemoveAll(dataDir)
	if err != nil {
		a.getLogger("ReplaceData(), os.RemoveAll").Error(err)
		return err
	}
	err = copy.Copy(source, dataDir)
	if err != nil {
		a.getLogger("ReplaceData(), copy.Copy").Error(err)
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
