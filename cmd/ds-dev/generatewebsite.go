package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/blang/semver/v4"
	"github.com/cbroglie/mustache"
	"github.com/teleclimber/DropServer/cmd/ds-host/appops"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type SiteData struct {
	Listing string
	Latest  domain.AppVersionManifest
	Sorted  []domain.AppVersionManifest
}

func generateWebsite(dir string, templatePath string) error {
	var indexHTML []byte
	var err error
	if templatePath != "" {
		templatePath, err = filepath.Abs(templatePath)
		if err != nil {
			return fmt.Errorf("error evaluating absolute path for %s: %w", templatePath, err)
		}
		indexHTML, err = os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("error reading template file at %s: %w", templatePath, err)
		}
	} else {
		indexHTML, err = distsiteFS.ReadFile("distsite/index.html")
		if err != nil {
			return fmt.Errorf("error reading default template file: %w", err)
		}
	}

	listing, err := getAppListing(dir)
	if err != nil {
		return err
	}

	sortedVersions, err := getSortedVersions(listing.Versions)
	if err != nil {
		return err
	}
	if len(sortedVersions) == 0 {
		return errors.New("no versions found in app listing")
	}
	sortedManifests := make([]domain.AppVersionManifest, len(sortedVersions))
	for i, v := range sortedVersions {
		m, err := getManifest(dir, listing.Versions[v])
		if err != nil {
			return err
		}
		sortedManifests[i] = m
	}

	siteData := SiteData{
		Listing: "app-listing.json", // we assume this for now
		Latest:  sortedManifests[0],
		Sorted:  sortedManifests,
	}

	html, err := mustache.Render(string(indexHTML), siteData)
	if err != nil {
		return err
	}

	outFile := filepath.Join(dir, "index.html")
	err = os.WriteFile(outFile, []byte(html), 0644)
	if err != nil {
		return fmt.Errorf("error writing index.html: %w", err)
	}
	fmt.Printf("Wrote %s\n", outFile)

	return nil
}

func getAppListing(dir string) (domain.AppListing, error) {
	var listing domain.AppListing
	data, err := os.ReadFile(filepath.Join(dir, "app-listing.json"))
	if err != nil {
		return listing, fmt.Errorf("error reading app listing: %w", err)
	}
	err = json.Unmarshal(data, &listing)
	if err != nil {
		return listing, fmt.Errorf("error reading JSON in app listing: %w", err)
	}
	return listing, nil
}

func getSortedVersions(versions map[domain.Version]domain.AppListingVersion) ([]domain.Version, error) {
	s := make([]struct {
		v  domain.Version
		sv semver.Version
	}, 0, len(versions))
	ret := make([]domain.Version, len(versions))
	for v := range versions {
		sv, err := semver.Parse(string(v))
		if err != nil {
			return ret, fmt.Errorf("error parsing version %v: %w", v, err)
		}
		s = append(s, struct {
			v  domain.Version
			sv semver.Version
		}{v, sv})
	}
	sort.Slice(s, func(i, j int) bool {
		return s[i].sv.Compare(s[j].sv) == 1
	})
	for i, sorted := range s {
		ret[i] = sorted.v
	}
	return ret, nil
}

func getManifest(dir string, vListing domain.AppListingVersion) (domain.AppVersionManifest, error) {
	var manifest domain.AppVersionManifest
	data, err := os.ReadFile(filepath.Join(dir, vListing.Manifest))
	if err != nil {
		return manifest, fmt.Errorf("error reading manifest at %s: %w", vListing.Manifest, err)
	}
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return manifest, fmt.Errorf("error reading manifest JSON at %s: %w", vListing.Manifest, err)
	}

	if vListing.Changelog == "" {
		return manifest, nil
	}
	fd, err := os.Open(filepath.Join(dir, vListing.Changelog))
	if err != nil {
		return manifest, fmt.Errorf("error reading changelog at %s: %w", vListing.Changelog, err)
	}
	sv, err := semver.Parse(string(manifest.Version))
	if err != nil {
		return manifest, fmt.Errorf("error parsing version in manifest at %s: %w", vListing.Manifest, err)
	}
	manifest.Changelog, err = appops.GetValidChangelog(fd, sv) // cheat and replace the changelog path with text string
	if err != nil {
		return manifest, fmt.Errorf("error getting valid changelog at %s: %w", vListing.Changelog, err)
	}

	return manifest, nil
}
