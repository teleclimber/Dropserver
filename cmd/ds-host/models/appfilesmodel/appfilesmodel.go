package appfilesmodel

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// performs operations on application files

// AppFilesModel is struct for application files manager
type AppFilesModel struct {
	Config *domain.RuntimeConfig
	Logger domain.LogCLientI
}

// Save puts the data passed in files in an apps directory
func (a *AppFilesModel) Save(files *map[string][]byte) (string, domain.Error) {
	appsPath := a.getAppsPath()

	err := os.MkdirAll(appsPath, 0766)
	if err != nil {
		a.Logger.Log(domain.ERROR, nil, "AppFilesModel: failed to create apps directory: "+err.Error())
		return "", dserror.New(dserror.InternalError)
	}

	appPath, err := ioutil.TempDir(appsPath, "app")
	if err != nil {
		a.Logger.Log(domain.ERROR, nil, "AppFilesModel: failed to create app directory: "+err.Error())
		return "", dserror.New(dserror.InternalError)
	}

	writeErr := false
	for f, data := range *files {
		fPath := filepath.Join(appPath, f)
		inside, err := pathInsidePath(fPath, appPath)
		if err != nil {
			a.Logger.Log(domain.ERROR, nil, "AppFilesModel: Error determining if file path outside of app path: "+f+": "+err.Error())
			writeErr = true
			break
		}
		if !inside {
			a.Logger.Log(domain.ERROR, nil, "AppFilesModel: file path outside of app path: "+f)
			writeErr = true
			break
		}

		err = os.MkdirAll(filepath.Dir(fPath), 0766)
		if err != nil {
			a.Logger.Log(domain.ERROR, nil, "AppFilesModel: failed to create app file directory: "+err.Error())
			return "", dserror.New(dserror.InternalError)
		}

		err = ioutil.WriteFile(fPath, data, 0666) // TODO: correct permissions?
		if err != nil {
			a.Logger.Log(domain.ERROR, nil, "AppFilesModel: failed to write app file: "+f+": "+err.Error())
			writeErr = true
			break
		}
	}

	if writeErr {
		return "", dserror.New(dserror.InternalError, err.Error())
	}

	locationKey := filepath.Base(appPath)

	a.Logger.Log(domain.INFO,
		map[string]string{"location-key": locationKey},
		fmt.Sprintf("AppFilesModel: Saved %d files", len(*files)))
	// ^^ here locationKey is good to include in map, but it should really be a type so we can reliably use it

	return locationKey, nil
}

// ReadMeta reads metadata from the files at location key
func (a *AppFilesModel) ReadMeta(locationKey string) (*domain.AppFilesMetadata, domain.Error) {
	appsPath := a.getAppsPath()
	jsonPath := filepath.Join(appsPath, locationKey, "application.json")
	jsonHandle, err := os.Open(jsonPath)
	if err != nil {
		// here the error might be that application.json is not in app?
		// Or it could be a more internal problem, like directory of apps not where it's expected to be.
		// Or it could be a bad location key, like it was deleted but DB doesn't know.
		if !a.locationKeyExists(locationKey) {
			a.Logger.Log(domain.ERROR,
				map[string]string{"location-key": locationKey},
				"AppFilesModel: Locationkey does not exist")
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

	return meta, nil
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

	//TODO: validate what comes out of JSON against valid values
	// auth keywords, routes can be empty strings, handler type is enum...

	return nil
}

func (a *AppFilesModel) locationKeyExists(locationKey string) bool {
	appsPath := a.getAppsPath()
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

func (a *AppFilesModel) getAppsPath() string {
	return filepath.Join(a.Config.DataDir, "apps")
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
