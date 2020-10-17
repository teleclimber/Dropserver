package appfilesmodel

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// performs operations on application files

// AppFilesModel is struct for application files manager
type AppFilesModel struct {
	Config *domain.RuntimeConfig
}

// Save puts the data passed in files in an apps directory
func (a *AppFilesModel) Save(files *map[string][]byte) (string, domain.Error) {
	logger := a.getLogger("Save()")
	appsPath := a.Config.Exec.AppsPath

	err := os.MkdirAll(appsPath, 0766)
	if err != nil {
		logger.AddNote("os.MkdirAll()").Error(err)
		return "", dserror.New(dserror.InternalError) // user-friendly error
	}

	appPath, err := ioutil.TempDir(appsPath, "app")
	if err != nil {
		logger.AddNote("ioutil.TempDir()").Error(err)
		return "", dserror.New(dserror.InternalError)
	}

	logger.AddNote("files loop")
	writeErr := false
	for f, data := range *files {
		fPath := filepath.Join(appPath, f)
		inside, err := pathInsidePath(fPath, appPath)
		if err != nil {
			logger.AddNote("pathInsidePath()").Error(err)
			writeErr = true
			break
		}
		if !inside {
			logger.Log(fmt.Sprintf("file path outside of app path: %v", f))
			writeErr = true
			break
		}

		err = os.MkdirAll(filepath.Dir(fPath), 0766)
		if err != nil {
			logger.AddNote(fmt.Sprintf("os.MkdirAll(): %v", f)).Error(err)
			return "", dserror.New(dserror.InternalError)
		}

		err = ioutil.WriteFile(fPath, data, 0666) // TODO: correct permissions?
		if err != nil {
			logger.AddNote(fmt.Sprintf("ioutil.WriteFile(): %v", f)).Error(err)
			writeErr = true
			break
		}
	}

	if writeErr {
		return "", dserror.New(dserror.InternalError, err.Error())
	}

	locationKey := filepath.Base(appPath)

	return locationKey, nil
}

// ReadMeta reads metadata from the files at location key
func (a *AppFilesModel) ReadMeta(locationKey string) (*domain.AppFilesMetadata, domain.Error) {
	jsonPath := filepath.Join(a.Config.Exec.AppsPath, locationKey, "drop-app.json")
	jsonHandle, err := os.Open(jsonPath)
	if err != nil {
		// here the error might be that application.json is not in app?
		// Or it could be a more internal problem, like directory of apps not where it's expected to be.
		// Or it could be a bad location key, like it was deleted but DB doesn't know.
		if !a.locationKeyExists(locationKey) {
			a.getLogger(fmt.Sprintf("ReadMeta(), location key: %v", locationKey)).Error(err)
			return nil, dserror.New(dserror.InternalError, "ReadMeta: Location key not found "+locationKey)
		}
		return nil, dserror.New(dserror.AppConfigNotFound)
	}
	defer jsonHandle.Close()

	meta, dsErr := decodeAppJSON(jsonHandle)
	if dsErr != nil {
		return nil, dsErr
	}

	dsErr = validateAppMeta(meta)
	if dsErr != nil {
		return nil, dsErr
	}

	// Other metadata:
	// in application.json: app name, author, ...
	// overall: num files, total size of all files
	// migrations: migration levels available, or at least the latest one

	// Migration level:
	mInts, dsErr := a.getMigrationDirs(locationKey)
	if dsErr != nil {
		return nil, dsErr
	}
	meta.Migrations = mInts

	if len(mInts) == 0 {
		meta.SchemaVersion = 0
	} else {
		meta.SchemaVersion = mInts[len(mInts)-1]
	}

	return meta, nil
}

// Delete removes the files from the system
func (a *AppFilesModel) Delete(locationKey string) domain.Error {
	if !a.locationKeyExists(locationKey) {
		return nil //is that an error or do we consider this OK?
	}

	err := os.RemoveAll(filepath.Join(a.Config.Exec.AppsPath, locationKey))
	if err != nil {
		a.getLogger("Delete()").Error(err)
		return dserror.FromStandard(err)
	}

	return nil
}

func decodeAppJSON(r io.Reader) (*domain.AppFilesMetadata, domain.Error) {
	var meta domain.AppFilesMetadata
	dec := json.NewDecoder(r)
	err := dec.Decode(&meta)
	if err != nil {
		return nil, dserror.New(dserror.AppConfigParseFailed, err.Error())
	}

	// TODO: clean up data too, like trim whitespace on trings
	// ..could lowercap on keywords?

	return &meta, nil
}

func validateAppMeta(meta *domain.AppFilesMetadata) domain.Error {
	if meta.AppName == "" {
		return dserror.New(dserror.AppConfigProblem, "Name can not be blank")
	}
	if meta.AppVersion == "" {
		return dserror.New(dserror.AppConfigProblem, "Version can not be blank")
	}

	return nil
}

func (a *AppFilesModel) getMigrationDirs(locationKey string) (ret []int, dsErr domain.Error) {
	mPath := filepath.Join(a.Config.Exec.AppsPath, locationKey, "migrations")

	mDir, err := os.Open(mPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		dsErr = dserror.FromStandard(err)
		return
	}

	list, err := mDir.Readdir(-1)
	if err != nil {
		dsErr = dserror.FromStandard(err)
		return
	}

	for _, f := range list {
		if !f.IsDir() {
			continue
		}

		dirInt, err := strconv.Atoi(filepath.Base(f.Name()))
		if err != nil {
			continue
		}

		if dirInt == 0 { // first legit number is 1 (the 0 state is "not yet installed")
			continue
		}

		ret = append(ret, dirInt)
	}

	sort.Ints(ret)

	return
}

func (a *AppFilesModel) locationKeyExists(locationKey string) bool {
	_, err := os.Stat(filepath.Join(a.Config.Exec.AppsPath, locationKey))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true // OK but there could be aonther problem, like permissions out of whack?
	// Should probably log that as warning at least.
}

func (a *AppFilesModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppFilesModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

// pathInsidePath determines if A path is inside (contained within) path B
func pathInsidePath(p, root string) (bool, error) {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false, err // not clear that is an error that is actually an error. Errors I think just mean "can't be inside"
	}

	if strings.Contains(rel, "..") {
		return false, nil
	}
	return true, nil
}
