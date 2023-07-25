package appops

import (
	"fmt"
	"image"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/webp"

	"github.com/blang/semver/v4"
	"github.com/inhies/go-bytesize"
	"github.com/kettek/apng"
	"github.com/mazznoer/csscolorparser"
	"github.com/rivo/uniseg"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type migrationPair struct {
	schema int
	up     bool
	down   bool
}

func validateMigrationSteps(meta *domain.AppGetMeta) error { // apparently never returns an error for now?
	migrations := meta.VersionManifest.Migrations
	if len(migrations) == 0 {
		return nil
	}

	// first validate each individual step:
	for _, step := range migrations {
		if step.Direction != "up" && step.Direction != "down" {
			meta.Errors = append(meta.Errors, fmt.Sprintf("Invalid migration step: %v %v: not up or down", step.Direction, step.Schema))
			return nil
		}
		if step.Schema < 1 {
			meta.Errors = append(meta.Errors, fmt.Sprintf("Invalid migration step: %v %v: schema less than 1", step.Direction, step.Schema))
			return nil
		}
	}

	pairs := make([]migrationPair, 0)
	for _, step := range migrations {
		found := false
		for i, p := range pairs {
			if p.schema == step.Schema {
				found = true
				if (step.Direction == "up" && p.up) || (step.Direction == "down" && p.down) {
					meta.Errors = append(meta.Errors, fmt.Sprintf("Invalid migration step: %v %v declared more than once", step.Direction, step.Schema))
				} else if step.Direction == "up" {
					pairs[i].up = true
				} else {
					pairs[i].down = true
				}
			}
		}
		if !found {
			pairs = append(pairs, migrationPair{schema: step.Schema, up: step.Direction == "up", down: step.Direction == "down"})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].schema > pairs[j].schema // sort reversed, so we iterate from the highest schema
	})

	if !pairs[0].up {
		meta.Errors = append(meta.Errors, fmt.Sprintf("Invalid migrations: the highest schema %v must have an up migration.", pairs[0].schema))
	}

	// what should we be warning the user about:
	// - gap in up migrations (makes the lower ones unusable)
	// - gap in down migration (same)
	// - any mismatch pair, except for 0->1
	prevUp := -1
	nextDown := -1
	warn := false
	for _, pair := range pairs {
		if pair.up {
			if prevUp != -1 && prevUp-1 != pair.schema {
				warn = true
			}
			prevUp = pair.schema
		}
		if pair.down {
			if nextDown != -1 && nextDown-1 != pair.schema {
				warn = true
			}
			nextDown = pair.schema
		}
		if !pair.up || (!pair.down && pair.schema != 1) {
			warn = true
		}
	}
	if warn {
		meta.Warnings["migrations"] = "Migrations should be sequential and come in up-down pairs."
	}

	meta.VersionManifest.Schema = pairs[0].schema

	return nil
}

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

func validateIcon(meta *domain.AppGetMeta, iconPath string) bool {
	f, err := os.Open(iconPath)
	if os.IsNotExist(err) {
		meta.Warnings["icon"] = "App icon not found at package path " + meta.VersionManifest.Icon
		return false
	}
	if err != nil {
		meta.Warnings["icon"] = "Error processing app icon:  " + err.Error()
		return false
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		meta.Warnings["icon"] = "Error getting icon file info:  " + err.Error()
		return false
	}
	if fInfo.IsDir() {
		meta.Warnings["icon"] = "Error: icon path is a directory"
		return false
	}

	mimeType, err := getFileMimeType(iconPath)
	if err != nil {
		meta.Warnings["icon"] = "Error getting app icon mime type:  " + err.Error()
		return false
	}

	mimeTypes := []string{"image/jpeg", "image/png", "image/svg+xml", "image/webp"}
	typeOk := false
	for _, t := range mimeTypes {
		if t == mimeType {
			typeOk = true
		}
	}
	if !typeOk {
		meta.Warnings["icon"] = "App icon type not supported:  " + mimeType + " Jpeg, png, svg and webp are supported."
		return false
	}

	// get w and h and check: is square and then size.
	var config image.Config
	if mimeType == "image/jpeg" || mimeType == "image/png" {
		config, _, err = image.DecodeConfig(f)
		if err != nil {
			meta.Warnings["icon"] = "Error reading app icon file. " + err.Error()
			return false
		}
	} else if mimeType == "image/webp" {
		config, err = webp.DecodeConfig(f)
		if err != nil {
			meta.Warnings["icon"] = "Error reading app icon file. " + err.Error()
			return false
		}
	}
	if mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/webp" {
		if config.Height != config.Width {
			meta.Warnings["icon"] = fmt.Sprintf("App icon is not square: %v x %v.", config.Width, config.Height)
		} else if config.Height < domain.AppIconMinPixelSize {
			meta.Warnings["icon"] = fmt.Sprintf("App icon should be at least %v pixels. It is %v pixels.", domain.AppIconMinPixelSize, config.Width)
		}
	}

	if fInfo.Size() > domain.AppIconMaxFileSize {
		appendWarning(meta, "icon", fmt.Sprintf("App icon file is large: %s (under %s is recommended).",
			bytesize.New(float64(fInfo.Size())), bytesize.New(float64(domain.AppIconMaxFileSize))))
	}

	if mimeType == "image/png" {
		// need to open again so the decoder can work from the beginning
		fPng, err := os.Open(iconPath)
		if err != nil {
			meta.Warnings["icon"] = "Error opening PNG app icon: " + err.Error()
			return false
		}
		defer fPng.Close()
		a, err := apng.DecodeAll(fPng)
		if err != nil {
			meta.Warnings["icon"] = "Error opening decoding PNG app icon: " + err.Error()
		}
		if len(a.Frames) > 1 {
			appendWarning(meta, "icon", "App icon appears to be animated. Non-animated icons are preferred.")
		}
	}

	return true
}

func getFileMimeType(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	byteSlice := make([]byte, 512)
	_, err = f.Read(byteSlice)
	if err != nil {
		return "", fmt.Errorf("error reading bytes from file: %w", err)
	}
	contentType := http.DetectContentType(byteSlice)

	return contentType, nil
}

func validatePackagePath(p string) (string, bool) {
	if p == "" {
		return "", true
	}
	p = path.Clean(p)
	if p == "" || p == "." || p == "/" || strings.Contains(p, "..") || strings.Contains(p, "\\") {
		return "", false
	}
	if path.IsAbs(p) {
		// accept abolute but turn it to relative
		p = p[1:]
	}
	return p, true
}

func appendWarning(meta *domain.AppGetMeta, key string, warning string) {
	w := meta.Warnings[key]
	if w != "" {
		w = w + " "
	}
	meta.Warnings[key] = w + warning
}
