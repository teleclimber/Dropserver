package appops

import (
	"fmt"

	"github.com/blang/semver/v4"
	"github.com/mazznoer/csscolorparser"
	"github.com/rivo/uniseg"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

// validate that app version has a name
// validate that the DS API is usabel in this version of DS
// Bit of a misnomer? It validates app name, api version, app version, user permissions.// it doenst nor shoud it.
// Oh wait validate "version" here means app code version.
func validateVersion(meta *domain.AppGetMeta) error {
	manifest := meta.VersionManifest
	if manifest.Name == "" {
		meta.Errors = append(meta.Errors, "App name can not be blank") // TODO I thought we said name wasn't required?
	}

	parsedVer, err := semver.ParseTolerant(string(manifest.Version))
	if err != nil {
		meta.Errors = append(meta.Errors, err.Error()) // TODO clarify it's a semver error
		return nil
	}

	meta.VersionManifest.Version = domain.Version(parsedVer.String())

	return nil
}

func validateAccentColor(meta *domain.AppGetMeta) error {
	if meta.VersionManifest.AccentColor == "" {
		return nil
	}
	c, err := csscolorparser.Parse(meta.VersionManifest.AccentColor)
	if err != nil {
		meta.Warnings["accent-color"] = fmt.Sprintf("Unable to parse %s: invalid CSS color.", meta.VersionManifest.AccentColor)
		meta.VersionManifest.AccentColor = ""
		return nil
	}
	meta.VersionManifest.AccentColor = c.HexString()
	return nil
}

func validateSoftData(meta *domain.AppGetMeta) {
	c := uniseg.GraphemeClusterCount(meta.VersionManifest.Name)
	if c > domain.AppNameMaxLength {
		meta.Warnings["name"] = fmt.Sprintf("App name is over %v characters (%v). It may be difficult to display.", domain.AppNameMaxLength, c)
	}
	c = uniseg.GraphemeClusterCount(meta.VersionManifest.ShortDescription)
	if c > domain.AppShortDescriptionMaxLength {
		meta.Warnings["short-description"] = fmt.Sprintf("Short description is over %v characters (%v). It may be difficult to display.", domain.AppShortDescriptionMaxLength, c)
	}

	for i, a := range meta.VersionManifest.Authors {
		if a.Email != "" {
			err := validator.Email(a.Email)
			if err != nil {
				meta.Warnings["authors"] = "Invalid author email: " + a.Email
				meta.VersionManifest.Authors[i].Email = ""
			}
		}
		if a.URL != "" {
			err := validator.HttpURL(a.URL)
			if err != nil {
				meta.Warnings["authors"] = "Invalid author URL: " + a.URL
				meta.VersionManifest.Authors[i].URL = ""
			}
		}
	}

	if meta.VersionManifest.Website != "" {
		err := validator.HttpURL(meta.VersionManifest.Website)
		if err != nil {
			meta.Warnings["website"] = "Removed invalid website URL: " + meta.VersionManifest.Website
			meta.VersionManifest.Website = ""
		}
	}
	if meta.VersionManifest.Code != "" {
		err := validator.HttpURL(meta.VersionManifest.Code)
		if err != nil {
			meta.Warnings["code"] = "Removed invalid code URL: " + meta.VersionManifest.Code
			meta.VersionManifest.Code = ""
		}
	}
	if meta.VersionManifest.Funding != "" {
		err := validator.HttpURL(meta.VersionManifest.Funding)
		if err != nil {
			meta.Warnings["funding"] = "Removed invalid funding URL: " + meta.VersionManifest.Funding
			meta.VersionManifest.Funding = ""
		}
	}
}
