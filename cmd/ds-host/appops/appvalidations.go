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

func (g *AppGetter) validateMigrationSteps(keyData appGetData) error { // apparently never returns an error for now?
	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return nil
	}
	migrations := meta.VersionManifest.Migrations
	if len(migrations) == 0 {
		return nil
	}

	// first validate each individual step:
	for _, step := range migrations {
		if step.Direction != "up" && step.Direction != "down" {
			g.appendErrorResult(keyData.key, fmt.Sprintf("Invalid migration step: %v %v: not up or down", step.Direction, step.Schema))
			return nil
		}
		if step.Schema < 1 {
			g.appendErrorResult(keyData.key, fmt.Sprintf("Invalid migration step: %v %v: schema less than 1", step.Direction, step.Schema))
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
					g.appendErrorResult(keyData.key, fmt.Sprintf("Invalid migration step: %v %v declared more than once", step.Direction, step.Schema))
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
		g.appendErrorResult(keyData.key, fmt.Sprintf("Invalid migrations: the highest schema %v must have an up migration.", pairs[0].schema))
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
		if !pair.up || !pair.down {
			warn = true
		}
	}
	if warn {
		g.setWarningResult(keyData.key, "migrations", "Migrations should be sequential and come in up-down pairs.")
	}

	meta.VersionManifest.Schema = pairs[0].schema
	g.setManifestResult(keyData.key, meta.VersionManifest)

	return nil
}

// validate that app version has a name
// validate that the DS API is usabel in this version of DS
// Bit of a misnomer? It validates app name, api version, app version, user permissions.// it doenst nor shoud it.
// Oh wait validate "version" here means app code version.
func (g *AppGetter) validateVersion(keyData appGetData) error {
	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return nil
	}
	manifest := meta.VersionManifest
	if manifest.Name == "" {
		g.appendErrorResult(keyData.key, "App name can not be blank") // TODO I thought we said name wasn't required?
	}

	parsedVer, err := semver.ParseTolerant(string(manifest.Version))
	if err != nil {
		g.appendErrorResult(keyData.key, err.Error()) // TODO clarify it's a semver error
		return nil
	}

	manifest.Version = domain.Version(parsedVer.String())
	g.setManifestResult(keyData.key, manifest)

	return nil
}

func (g *AppGetter) validateAccentColor(keyData appGetData) {
	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return
	}
	color := meta.VersionManifest.AccentColor
	if color == "" {
		return
	}
	c, err := csscolorparser.Parse(color)
	if err != nil {
		g.setWarningResult(keyData.key, "icon", fmt.Sprintf("Unable to parse %s: invalid CSS color.", color))
		meta.VersionManifest.AccentColor = ""
	}
	meta.VersionManifest.AccentColor = c.HexString()
	g.setManifestResult(keyData.key, meta.VersionManifest)
}

func (g *AppGetter) validateSoftData(keyData appGetData) {
	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return
	}

	c := uniseg.GraphemeClusterCount(meta.VersionManifest.Name)
	if c > domain.AppNameMaxLength {
		g.setWarningResult(keyData.key, "name", fmt.Sprintf("App name is over %v characters (%v). It may be difficult to display.", domain.AppNameMaxLength, c))
	}
	c = uniseg.GraphemeClusterCount(meta.VersionManifest.ShortDescription)
	if c > domain.AppShortDescriptionMaxLength {
		g.setWarningResult(keyData.key, "short-description", fmt.Sprintf("Short description is over %v characters (%v). It may be difficult to display.", domain.AppShortDescriptionMaxLength, c))
	}

	for i, a := range meta.VersionManifest.Authors {
		if a.Email != "" {
			err := validator.Email(a.Email)
			if err != nil {
				g.setWarningResult(keyData.key, "authors", "Invalid author email: "+a.Email)
				meta.VersionManifest.Authors[i].Email = ""
			}
		}
		if a.URL != "" {
			err := validator.HttpURL(a.URL)
			if err != nil {
				g.setWarningResult(keyData.key, "authors", "Invalid author URL: "+a.URL)
				meta.VersionManifest.Authors[i].URL = ""
			}
		}
	}

	if meta.VersionManifest.Website != "" {
		err := validator.HttpURL(meta.VersionManifest.Website)
		if err != nil {
			g.setWarningResult(keyData.key, "website", "Removed invalid website URL: "+meta.VersionManifest.Website)
			meta.VersionManifest.Website = ""
		}
	}
	if meta.VersionManifest.Code != "" {
		err := validator.HttpURL(meta.VersionManifest.Code)
		if err != nil {
			g.setWarningResult(keyData.key, "code", "Removed invalid code URL: "+meta.VersionManifest.Code)
			meta.VersionManifest.Code = ""
		}
	}
	if meta.VersionManifest.Funding != "" {
		err := validator.HttpURL(meta.VersionManifest.Funding)
		if err != nil {
			g.setWarningResult(keyData.key, "funding", "Removed invalid funding URL: "+meta.VersionManifest.Funding)
			meta.VersionManifest.Funding = ""
		}
	}

	g.setManifestResult(keyData.key, meta.VersionManifest)
}

func validateIcon(iconPath string) (string, bool) {
	f, err := os.Open(iconPath)
	if os.IsNotExist(err) {
		return "App icon file not found at specified path", false
	}
	if err != nil {
		return "Error processing app icon:  " + err.Error(), false
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		return "Error getting icon file info:  " + err.Error(), false
	}
	if fInfo.IsDir() {
		return "Error: icon path is a directory", false
	}

	mimeType, err := getFileMimeType(iconPath)
	if err != nil {
		return "Error getting app icon mime type:  " + err.Error(), false
	}

	mimeTypes := []string{"image/jpeg", "image/png", "image/svg+xml", "image/webp"}
	typeOk := false
	for _, t := range mimeTypes {
		if t == mimeType {
			typeOk = true
		}
	}
	if !typeOk {
		return "App icon type not supported:  " + mimeType + " Jpeg, png, svg and webp are supported.", false
	}

	// get w and h and check: is square and then size.
	var config image.Config
	if mimeType == "image/jpeg" || mimeType == "image/png" {
		config, _, err = image.DecodeConfig(f)
		if err != nil {
			return "Error reading app icon file. " + err.Error(), false
		}
	} else if mimeType == "image/webp" {
		config, err = webp.DecodeConfig(f)
		if err != nil {
			return "Error reading app icon file. " + err.Error(), false
		}
	}
	warn := ""
	if mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/webp" {
		if config.Height != config.Width {
			warn = fmt.Sprintf("App icon is not square: %v x %v.", config.Width, config.Height)
		} else if config.Height < domain.AppIconMinPixelSize {
			warn = fmt.Sprintf("App icon should be at least %v pixels. It is %v pixels.", domain.AppIconMinPixelSize, config.Width)
		}
	}

	if fInfo.Size() > domain.AppIconMaxFileSize {
		warn = appendWarning(warn, fmt.Sprintf("App icon file is large: %s (under %s is recommended).",
			bytesize.New(float64(fInfo.Size())), bytesize.New(float64(domain.AppIconMaxFileSize))))
	}

	if mimeType == "image/png" {
		// need to open again so the decoder can work from the beginning
		fPng, err := os.Open(iconPath)
		if err != nil {
			return "Error opening PNG app icon: " + err.Error(), false
		}
		defer fPng.Close()
		a, err := apng.DecodeAll(fPng)
		if err != nil {
			return "Error opening decoding PNG app icon: " + err.Error(), false
		}
		if len(a.Frames) > 1 {
			return "App icon appears to be animated. Non-animated icons are preferred.", false
		}
	}

	return warn, true
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

func appendWarning(og, ap string) string {
	if og == "" {
		return ap
	}
	return og + " " + ap
}
