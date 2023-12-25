package main

import (
	"os"
	"path/filepath"

	"github.com/blang/semver/v4"
	"github.com/teleclimber/DropServer/cmd/ds-host/appops"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
)

// DevAppFilesModel embeds AppFilesModel to intercept
// Read/Write Routes to avoid writing to disk
type DevAppFilesModel struct {
	appfilesmodel.AppFilesModel `checkinject:"required"`

	routesData []byte
	manifest   domain.AppVersionManifest
	fileLinks  map[string]string
}

func (a *DevAppFilesModel) Save(files *map[string][]byte) (string, error) {
	panic("Save should never be called!")
}

// WriteRoutes keeps the data in memory instead of writing to disk
func (a *DevAppFilesModel) WriteRoutes(locationKey string, routesData []byte) error {
	a.routesData = routesData
	return nil
}

// ReadRoutes returns the in-memory data
func (a *DevAppFilesModel) ReadRoutes(locationKey string) ([]byte, error) {
	return a.routesData, nil
}

func (a *DevAppFilesModel) WriteFileLink(locationKey string, linkName string, iconPath string) error {
	a.fileLinks[linkName] = iconPath
	return nil
}
func (a *DevAppFilesModel) GetLinkPath(locationKey string, linkName string) string {
	p, ok := a.fileLinks[linkName]
	if !ok {
		return ""
	}
	return filepath.Join(a.AppLocation2Path.Files(locationKey), p)
}

func (a *DevAppFilesModel) GetVersionChangelog(locationKey string, version domain.Version) (string, error) {
	p := a.GetLinkPath(locationKey, "changelog")
	if p == "" { // no changelog file, no changelog.
		return "", nil
	}
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	targetVer, err := semver.ParseTolerant(string(version))
	if err != nil {
		return "", err // definitely a Dropserver error
	}

	ret, err := appops.GetValidChangelog(f, targetVer)
	return ret, err
}

func (a *DevAppFilesModel) Delete(locationKey string) error {
	panic("Delete should never be called!")
}

func extractPackage(packagePath, tempDir string) string {
	appDir := filepath.Join(tempDir, "app-code")

	f, err := os.Open(packagePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = appfilesmodel.ExtractPackageLow(f, appDir, domain.AppExtractedPackageMaxSize)
	if err != nil {
		panic(err)
	}

	return appDir
}
