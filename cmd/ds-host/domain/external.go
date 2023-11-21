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
