package domain

// AppListingVersion contains relative paths to app files
// for one version.
type AppListingVersion struct {
	// Manifest is relative path to the manifest JSON file
	Manifest string `json:"manifest"`
	// Manifest is relative path to the package file (required)
	Package string `json:"package"`
	// Manifest is relative path to the changelog text file for this version
	Changelog string `json:"changelog"`
	// Icon is relative path of the application icon.
	// It should be the same file as manifest.Icon
	Icon string `json:"icon"`
}

// AppListing describes a set of application versions and
// Urls and paths for requesting additional files
type AppListing struct {
	// Base URL for all relative paths in listing
	// If not specified the Base is determined from the listing URL
	Base string `json:"base,omitempty"`
	// Versions contains data for each version
	// The field name must match the version in the manifest
	Versions map[Version]AppListingVersion `json:"versions"`
}
