package appfilesmodel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// performs operations on application files

// AppFilesModel is struct for application files manager
type AppFilesModel struct {
	AppLocation2Path interface {
		Base(string) string
		Meta(string) string
		Files(string) string
	} `checkinject:"required"`
	Config *domain.RuntimeConfig `checkinject:"required"`
}

// Save puts the data passed in files in an apps directory
// Wonder if we will need to save additional files here if for ex the manifest points to a file as URL?
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

// ReadManifest reads metadata from the files at location key
// this just reads the JSON file. OK.
// In next iteration this will presumably read teh normalized JSON that was evaluated prior.
// OR, this is the original file from the package? And we store the augmented / normalized version in the DB?
// ^^ This is correct. The app is stored unmodified, and the normalized/processed/evaluated data is stashed in the DB.
func (a *AppFilesModel) ReadManifest(locationKey string) (domain.AppVersionManifest, error) {
	jsonPath := filepath.Join(a.AppLocation2Path.Files(locationKey), "dropapp.json")
	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		// here the error might be that dropapp.json is not in app?
		// Or it could be a more internal problem, like directory of apps not where it's expected to be.
		// Or it could be a bad location key, like it was deleted but DB doesn't know.
		if !a.locationKeyExists(locationKey) {
			a.getLogger(fmt.Sprintf("ReadManifest(), location key: %v", locationKey)).Error(err)
			return domain.AppVersionManifest{}, errors.New("internal error reading app meta data")
		}
		return domain.AppVersionManifest{}, domain.ErrAppManifestNotFound // rename to app manifest
	}

	meta, err := unmarshalManifest(jsonBytes)
	if err != nil {
		return domain.AppVersionManifest{}, err
	}

	return meta, nil
}

func (a *AppFilesModel) WriteRoutes(locationKey string, routesData []byte) error {
	routesFile := filepath.Join(a.AppLocation2Path.Meta(locationKey), "routes.json")
	err := ioutil.WriteFile(routesFile, routesData, 0666) // TODO: correct permissions?
	if err != nil {
		a.getLogger(fmt.Sprintf("WriteRoutes(), location key: %v", locationKey)).Error(err)
		return err
	}
	return nil
}

func (a *AppFilesModel) ReadRoutes(locationKey string) ([]byte, error) {
	routesFile := filepath.Join(a.AppLocation2Path.Meta(locationKey), "routes.json")
	routesData, err := ioutil.ReadFile(routesFile)
	if err != nil {
		a.getLogger(fmt.Sprintf("ReadRoutes(), location key: %v", locationKey)).Error(err)
		return nil, err
	}
	return routesData, nil
}

func (a *AppFilesModel) WriteMigrations(locationKey string, routesData []byte) error {
	migrationsFile := filepath.Join(a.AppLocation2Path.Meta(locationKey), "migrations.json")
	err := ioutil.WriteFile(migrationsFile, routesData, 0666) // TODO: correct permissions?
	if err != nil {
		a.getLogger(fmt.Sprintf("WriteMigrations(), location key: %v", locationKey)).Error(err)
		return err
	}
	return nil
}

func (a *AppFilesModel) ReadMigrations(locationKey string) ([]byte, error) {
	migrationsFile := filepath.Join(a.AppLocation2Path.Meta(locationKey), "migrations.json")
	migrationsData, err := ioutil.ReadFile(migrationsFile)
	if err != nil {
		a.getLogger(fmt.Sprintf("ReadMigrations(), location key: %v", locationKey)).Error(err)
		return nil, err
	}
	return migrationsData, nil
}

// should we have the equivalent for reading and writing migrations?

// Delete removes the files from the system
func (a *AppFilesModel) Delete(locationKey string) error {
	if !a.locationKeyExists(locationKey) {
		return nil //is that an error or do we consider this OK?
	}

	err := os.RemoveAll(a.AppLocation2Path.Base(locationKey))
	if err != nil {
		a.getLogger("Delete()").Error(err)
		return err
	}

	return nil
}

func unmarshalManifest(b []byte) (domain.AppVersionManifest, error) {
	var manifest domain.AppVersionManifest

	err := json.Unmarshal(b, &manifest)
	if err != nil {
		return manifest, err
	}

	return manifest, nil
}

func (a *AppFilesModel) locationKeyExists(locationKey string) bool {
	_, err := os.Stat(a.AppLocation2Path.Base(locationKey))
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
