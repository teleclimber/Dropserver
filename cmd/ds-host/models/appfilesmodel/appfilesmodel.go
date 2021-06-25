package appfilesmodel

import (
	"encoding/json"
	"errors"
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
)

// performs operations on application files

// AppFilesModel is struct for application files manager
type AppFilesModel struct {
	Location2Path interface {
		App(string) string
		AppFiles(string) string
	}
	Config *domain.RuntimeConfig
}

// Save puts the data passed in files in an apps directory
func (a *AppFilesModel) Save(files *map[string][]byte) (string, error) {
	logger := a.getLogger("Save()")
	appsPath := a.Config.Exec.AppsPath

	err := os.MkdirAll(appsPath, 0766)
	if err != nil {
		logger.AddNote("os.MkdirAll() appsPath").Error(err)
		return "", errors.New("internal error saving app files")
	}

	appPath, err := ioutil.TempDir(appsPath, "app")
	if err != nil {
		logger.AddNote("ioutil.TempDir()").Error(err)
		return "", errors.New("internal error saving app files")
	}

	locationKey := filepath.Base(appPath)

	appPath = filepath.Join(appPath, "app")
	err = os.MkdirAll(appPath, 0766) // omg permissions!
	if err != nil {
		logger.AddNote("os.MkdirAll() app").Error(err)
		return "", errors.New("internal error saving app files")
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
			return "", errors.New("internal error saving app files")
		}

		err = ioutil.WriteFile(fPath, data, 0666) // TODO: correct permissions?
		if err != nil {
			logger.AddNote(fmt.Sprintf("ioutil.WriteFile(): %v", f)).Error(err)
			writeErr = true
			break
		}
	}

	if writeErr {
		return "", errors.New("internal error saving app files")
	}

	return locationKey, nil
}

// ReadMeta reads metadata from the files at location key
func (a *AppFilesModel) ReadMeta(locationKey string) (*domain.AppFilesMetadata, error) {
	jsonPath := filepath.Join(a.Location2Path.AppFiles(locationKey), "dropapp.json")
	jsonHandle, err := os.Open(jsonPath)
	if err != nil {
		// here the error might be that dropapp.json is not in app?
		// Or it could be a more internal problem, like directory of apps not where it's expected to be.
		// Or it could be a bad location key, like it was deleted but DB doesn't know.
		if !a.locationKeyExists(locationKey) {
			a.getLogger(fmt.Sprintf("ReadMeta(), location key: %v", locationKey)).Error(err)
			return nil, errors.New("internal error reading app meta data")
		}
		return nil, domain.ErrAppConfigNotFound
	}
	defer jsonHandle.Close()

	meta, err := decodeAppJSON(jsonHandle)
	if err != nil {
		return nil, err
	}

	// Other metadata:
	// in application.json: app name, author, ...
	// overall: num files, total size of all files
	// migrations: migration levels available, or at least the latest one

	// Migration level:
	mInts, err := a.getMigrationDirs(locationKey)
	if err != nil {
		return nil, err
	}
	meta.Migrations = mInts

	if len(mInts) == 0 {
		meta.SchemaVersion = 0
	} else {
		meta.SchemaVersion = mInts[len(mInts)-1]
	}

	return meta, nil
}

func (a *AppFilesModel) WriteRoutes(locationKey string, routesData []byte) error {
	routesFile := filepath.Join(a.Location2Path.App(locationKey), "routes.json")
	err := ioutil.WriteFile(routesFile, routesData, 0666) // TODO: correct permissions?
	if err != nil {
		a.getLogger(fmt.Sprintf("WriteRoutes(), location key: %v", locationKey)).Error(err)
		return err
	}
	return nil
}

func (a *AppFilesModel) ReadRoutes(locationKey string) ([]byte, error) {
	routesFile := filepath.Join(a.Location2Path.App(locationKey), "routes.json")
	routesData, err := ioutil.ReadFile(routesFile)
	if err != nil {
		a.getLogger(fmt.Sprintf("ReadRoutes(), location key: %v", locationKey)).Error(err)
		return nil, err
	}
	return routesData, nil
}

// Delete removes the files from the system
func (a *AppFilesModel) Delete(locationKey string) error {
	if !a.locationKeyExists(locationKey) {
		return nil //is that an error or do we consider this OK?
	}

	err := os.RemoveAll(a.Location2Path.App(locationKey))
	if err != nil {
		a.getLogger("Delete()").Error(err)
		return err
	}

	return nil
}

func decodeAppJSON(r io.Reader) (*domain.AppFilesMetadata, error) {
	var meta domain.AppFilesMetadata
	dec := json.NewDecoder(r)
	err := dec.Decode(&meta)
	if err != nil {
		return nil, err
	}

	// TODO: clean up data too, like trim whitespace on strings
	// ..could lowercap on keywords?

	for i, p := range meta.UserPermissions {
		meta.UserPermissions[i].Key = strings.TrimSpace(p.Key)
	}

	return &meta, nil
}

func (a *AppFilesModel) getMigrationDirs(locationKey string) (ret []int, err error) {
	mPath := filepath.Join(a.Location2Path.AppFiles(locationKey), "migrations")

	mDir, err := os.Open(mPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}

	list, err := mDir.Readdir(-1)
	if err != nil {
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
	_, err := os.Stat(a.Location2Path.App(locationKey))
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
