package domain

// AppListingVersion contains relative paths to app files
// for one version.
type AppListingVersion struct {
	// Manifest is relative path to the manifest JSON file
	Manifest string `json:"manifest"`
	// Package is relative path to the package file (required)
	Package string `json:"package"`
	// Changelog is relative path to the changelog text file for this version
	Changelog string `json:"changelog,omitempty"`
	// Icon is relative path of the application icon.
	// It should be the same file as manifest.Icon
	Icon string `json:"icon,omitempty"`
}

// AppListing describes a set of application versions and
// Urls and paths for requesting additional files
type AppListing struct {
	// NewURL of the app listing.
	// If this is non-empty the rest of the listing data is ignored
	NewURL string `json:"new-url,omitempty"`
	// Base URL for all relative paths in listing
	// If not specified the Base is determined from the listing URL
	Base string `json:"base,omitempty"`
	// Versions contains data for each version
	// The field name must match the version in the manifest
	Versions map[Version]AppListingVersion `json:"versions"`
}

type ManifestAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

type AppVersionManifest struct {
	// Name of the application. Optional.
	Name string `json:"name"` // L10N? Also should have omitempty?
	// ShortDescription is a 10-15 words used to tell prsopective users what this is does.
	ShortDescription string `json:"short-description"` // I18N string.
	// Version in semver format. Required.
	Version Version `json:"version"`
	// Entrypoint is the script that runs the app. Optional. If ommitted system will look for app.ts or app.js.
	Entrypoint string `json:"entrypoint"`
	// Schema is the verion of the appspace data schema.
	// This is determined automatically by the system.
	Schema int `json:"schema"`
	// Migrations is list of migrations provided by this app version
	Migrations []MigrationStep `json:"migrations"`

	// Icon is a package-relative path to an icon file to display within the installer instance UI.
	Icon string `json:"icon"`
	//AccentColor is a CSS color used to differentiate the app in the Dropserver UI
	AccentColor string `json:"accent-color"`

	// Both of these are not currently handled.
	// Description  string `json:"description"`   // link to markdown file? I18N??

	// Changelog is a package-relative path to a text file that contains release notes
	Changelog string `json:"changelog"`

	// Authors
	Authors []ManifestAuthor `json:"authors"`

	// Code is the URL of the code repository
	Code string `json:"code"`
	// Website for the app
	Website string `json:"website"`
	// Funding website or site where funding situation is explained
	Funding string `json:"funding"` // should maybe not be a string only...
	// License in SPDX string form
	License string `json:"license"`
	// LicenseFile is a package-relative path to a txt file containing the license text.
	LicenseFile string `json:"license-file"` // Rel path to license file within package.

	//ReleaseDate YYYY-MM-DD of software release date. Should be set automatically by packaging code.
	ReleaseDate string `json:"release-date"` // date of packaging.

	// Size of the installed package in bytes (except that additional space will be taken up when fetching remote modules if applicable)
	// Although maybe the actual installed size can be measured by the packaging system?
	// Size int `json:"size"`
}
