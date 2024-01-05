package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// generateListing generates a listing json for the pacakges found in pacakgesDir
// It outputs the listing in that directory.
func generateListing(packagesDir string) error {
	versions, err := getVersions(packagesDir)
	if err != nil {
		return err
	}

	// validate version sequence here? or in getVersions?

	listing := domain.AppListing{
		Base:     "", // TODO
		Versions: make(map[domain.Version]domain.AppListingVersion),
	}

	for _, v := range versions {
		listing.Versions[v.version] = domain.AppListingVersion{
			Package:   v.packagePath,
			Manifest:  v.manifestPath,
			Changelog: v.changelogPath,
			Icon:      v.iconPath,
		}
	}

	outBytes, err := json.Marshal(listing)
	if err != nil {
		return err
	}
	listingFullPath := filepath.Join(packagesDir, "app-listing.json")
	err = os.WriteFile(listingFullPath, outBytes, 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Wrote %s \n", listingFullPath)

	return nil
}

type versionData struct {
	version       domain.Version
	packagePath   string
	manifestPath  string
	iconPath      string
	changelogPath string
}

// TODO we should validate version sequence too.
// if we include the schema in versionData that should be doable
// however we don't have a function that does this already it seems.

func getVersions(packagesDir string) (map[domain.Version]versionData, error) {
	files, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, err
	}

	versions := make(map[domain.Version]versionData)
	for _, f := range files {
		n := f.Name()
		if strings.HasSuffix(n, ".tar.gz") {
			d, err := getPackageData(packagesDir, n)
			if err != nil {
				return nil, err
			}
			if exsitingData, exists := versions[d.version]; exists {
				return nil, fmt.Errorf("version %v from package at %v already found in package at %v", d.version, d.packagePath, exsitingData.packagePath)
			}
			versions[d.version] = d
		}
	}
	if len(versions) == 0 {
		return versions, errors.New("found zero versions in directory")
	}

	for _, f := range files {
		n := f.Name()
		if strings.HasSuffix(n, ".json") {
			manifest, err := getFullManifest(filepath.Join(packagesDir, n))
			if err != nil {
				return nil, err
			}
			version := manifest.Version
			vData, ok := versions[version]
			if ok {
				if vData.manifestPath == "" {
					vData.manifestPath = n
					versions[version] = addExtraPaths(packagesDir, manifest, vData)
				} else {
					fmt.Printf("another manifest found for version %v: %v", version, n)
				}
			} else if version != domain.Version("") {
				fmt.Printf("manifest found for which there is no package: %v", n)
			}
		}
	}

	for _, d := range versions {
		if d.manifestPath == "" {
			return nil, fmt.Errorf("manifest missing for version %v", d.version)
		}
	}

	return versions, nil
}

func getPackageData(basePath, packagePath string) (versionData, error) {
	packageFD, err := os.Open(filepath.Join(basePath, packagePath))
	if err != nil {
		return versionData{}, err
	}
	defer packageFD.Close()

	packagedManifest, err := getPackagedManifest(packageFD)
	if err != nil {
		return versionData{}, fmt.Errorf("error getting manifest from package at %v: %v", packagePath, err)
	}
	if packagedManifest.Version == domain.Version("") {
		return versionData{}, fmt.Errorf("error version string is empty for package at %v", packagePath)
	}

	ret := versionData{
		version:     packagedManifest.Version,
		packagePath: packagePath}

	return ret, nil
}

func getPackageFile(r io.Reader, name string) ([]byte, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == name {
			contents, err := io.ReadAll(tr)
			return contents, err
		}
	}
	return nil, fmt.Errorf("file not found in package")
}

func getPackagedManifest(packageFD io.Reader) (domain.AppVersionManifest, error) {
	manifestBytes, err := getPackageFile(packageFD, "dropapp.json")
	if err != nil {
		return domain.AppVersionManifest{}, fmt.Errorf("error reading manifest: %v", err)
	}
	var packagedManifest domain.AppVersionManifest
	err = json.Unmarshal(manifestBytes, &packagedManifest)
	if err != nil {
		return domain.AppVersionManifest{}, fmt.Errorf("JSON error in manifest: %v", err)
	}

	return packagedManifest, nil
}

func getFullManifest(manifestPath string) (domain.AppVersionManifest, error) {
	jsonBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return domain.AppVersionManifest{}, err
	}
	var manifest domain.AppVersionManifest
	err = json.Unmarshal(jsonBytes, &manifest)
	if err != nil {
		return manifest, err
	}
	return manifest, nil
}

func addExtraPaths(packagesDir string, manifest domain.AppVersionManifest, vData versionData) versionData {
	if fileExists(filepath.Join(packagesDir, manifest.Changelog)) {
		vData.changelogPath = manifest.Changelog
	} else {
		fmt.Printf("could not find changelog for version %s: %s\n", manifest.Version, manifest.Changelog)
	}
	if fileExists(filepath.Join(packagesDir, manifest.Icon)) {
		vData.iconPath = manifest.Icon
	} else {
		fmt.Printf("could not find icon for version %s: %s\n", manifest.Version, manifest.Icon)
	}
	return vData
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return false
	} else if err != nil {
		panic(err)
	}
	return true
}
