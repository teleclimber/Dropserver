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
	"github.com/github/go-spdx/v2/spdxexp"
	"github.com/inhies/go-bytesize"
	"github.com/kettek/apng"
	"github.com/mazznoer/csscolorparser"
	"github.com/rivo/uniseg"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

// validateManifest validates data for the manifest
// it returns an array of warnings and a cleaned manifest
func validateManifest(manifest domain.AppVersionManifest) (domain.AppVersionManifest, []domain.ProcessWarning) {
	warnings := make([]domain.ProcessWarning, 0, 10)

	// validate the version string
	cleanVersion, warning := validateVersion(manifest.Version)
	warnings = addWarning(warnings, warning)
	manifest.Version = cleanVersion

	// app name
	cleanName, warning := validateAppName(manifest.Name)
	warnings = addWarning(warnings, warning)
	manifest.Name = cleanName

	// short description
	cleanDesc, warning := validateShortDescription(manifest.ShortDescription)
	warnings = addWarning(warnings, warning)
	manifest.ShortDescription = cleanDesc

	// validate the app entry point
	cleanEntry, ok := validatePackagePath(manifest.Entrypoint)
	if !ok {
		warnings = append(warnings, domain.ProcessWarning{
			Field:    "entrypoint",
			Problem:  domain.ProblemInvalid,
			Message:  "Entrypoint path is invalid",
			BadValue: manifest.Entrypoint})
	}
	manifest.Entrypoint = cleanEntry

	// validate schema
	warning = validateSchema(manifest.Schema)
	warnings = addWarning(warnings, warning)

	// validate migrations:
	warns := validateMigrationSteps(manifest.Migrations)
	if len(warns) != 0 {
		warnings = addWarning(warnings, warns...)
	} else {
		warns = validateMigrationSequence(manifest.Migrations)
		invalid := false
		for _, w := range warns {
			if w.Problem == domain.ProblemInvalid {
				invalid = true
			}
		}
		if !invalid {
			schema := getSchemaFromMigrations(manifest.Migrations)
			if schema != manifest.Schema {
				warns = append(warns, domain.ProcessWarning{
					Field:   "schema",
					Problem: domain.ProblemInvalid,
					Message: fmt.Sprintf("Schema is not equal to highest up migration (%v versus %v)", manifest.Schema, schema)})
			}
		}
	}
	warnings = addWarning(warnings, warns...)

	warning = validateLicenseFields(manifest.License, manifest.LicenseFile)
	warnings = addWarning(warnings, warning)

	// authors
	cleanAuthors, warns := validateAuthors(manifest.Authors)
	warnings = addWarning(warnings, warns...)
	manifest.Authors = cleanAuthors

	// accent color
	color, ok := validateAccentColor(manifest.AccentColor)
	if !ok {
		warnings = append(warnings, domain.ProcessWarning{
			Field:    "accent-color",
			Problem:  domain.ProblemInvalid,
			Message:  "Accent color is invalid.",
			BadValue: manifest.AccentColor})
	}
	manifest.AccentColor = color

	// web links...
	cleanURL, ok := validateWebsite(manifest.Website)
	if !ok {
		warnings = append(warnings, domain.ProcessWarning{
			Field:    "website",
			Problem:  domain.ProblemInvalid,
			Message:  "Website URL is invalid.",
			BadValue: manifest.Website})
	}
	manifest.Website = cleanURL

	cleanURL, ok = validateWebsite(manifest.Funding)
	if !ok {
		warnings = append(warnings, domain.ProcessWarning{
			Field:    "funding",
			Problem:  domain.ProblemInvalid,
			Message:  "Funding website URL is invalid.",
			BadValue: manifest.Funding})
	}
	manifest.Funding = cleanURL

	cleanURL, ok = validateWebsite(manifest.Code)
	if !ok {
		warnings = append(warnings, domain.ProcessWarning{
			Field:    "code",
			Problem:  domain.ProblemInvalid,
			Message:  "Code website URL is invalid.",
			BadValue: manifest.Code})
	}
	manifest.Code = cleanURL

	return manifest, warnings
}

func addWarning(warnings []domain.ProcessWarning, warns ...domain.ProcessWarning) []domain.ProcessWarning {
	for _, w := range warns {
		if w == (domain.ProcessWarning{}) {
			continue
		}
		warnings = append(warnings, w)
	}
	return warnings
}

type migrationPair struct {
	schema int
	up     bool
	down   bool
}

func validateMigrationSteps(migrations []domain.MigrationStep) []domain.ProcessWarning {
	field := "migrations"
	warnings := make([]domain.ProcessWarning, 0, 10)
	// first validate each individual step:
	for _, step := range migrations {
		if step.Direction != "up" && step.Direction != "down" {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   field,
				Problem: domain.ProblemInvalid,
				Message: fmt.Sprintf("Invalid migration step: %v %v: not up or down", step.Direction, step.Schema)})
		}
		if step.Schema < 1 {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   field,
				Problem: domain.ProblemInvalid,
				Message: fmt.Sprintf("Invalid migration step: %v %v: schema less than 1", step.Direction, step.Schema)})
		}
	}
	return warnings
}

func validateMigrationSequence(migrations []domain.MigrationStep) []domain.ProcessWarning {
	if len(migrations) == 0 {
		return []domain.ProcessWarning{} // no warning if no migrations?
	}
	pairs := make([]migrationPair, 0)
	warnings := make([]domain.ProcessWarning, 0, 10)
	field := "migrations"
	for _, step := range migrations {
		found := false
		for i, p := range pairs {
			if p.schema == step.Schema {
				found = true
				if (step.Direction == "up" && p.up) || (step.Direction == "down" && p.down) {
					warnings = append(warnings, domain.ProcessWarning{
						Field:   field,
						Problem: domain.ProblemInvalid,
						Message: fmt.Sprintf("Invalid migration step: %v %v declared more than once", step.Direction, step.Schema)})
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
	if len(warnings) != 0 {
		return warnings
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].schema > pairs[j].schema // sort reversed, so we iterate from the highest schema
	})

	if !pairs[0].up {
		return append(warnings, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemInvalid,
			Message: fmt.Sprintf("Invalid migrations: the highest schema %v must have an up migration.", pairs[0].schema)})
	}

	// what should we be warning the user about:
	// - gap in up migrations (makes the lower ones unusable)
	// - gap in down migration (same)
	// - any mismatch pair, except for 0->1
	// - actually need to make any missing up migration from 1 to schema as invalid
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
		return append(warnings, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemPoorExperience,
			Message: "Migrations should be sequential and come in up and down pairs."})
	}
	return warnings
}

func validateSchema(s int) domain.ProcessWarning {
	if s < 0 {
		return domain.ProcessWarning{
			Field:   "schema",
			Message: "Schema is invalid",
			Problem: domain.ProblemInvalid}
	}
	return domain.ProcessWarning{}
}

// getSchemaFromMigrations returns schema for the migration steps
// It assumes the highest schema is the schema.
func getSchemaFromMigrations(migrations []domain.MigrationStep) int {
	ret := 0
	for _, m := range migrations {
		if m.Direction == "up" && m.Schema > ret {
			ret = m.Schema
		}
	}
	return ret
}

func validateVersion(version domain.Version) (domain.Version, domain.ProcessWarning) {
	v := strings.TrimSpace(string(version))
	if v == "" {
		return "", domain.ProcessWarning{
			Field:   "version",
			Message: "No version specified.",
			Problem: domain.ProblemEmpty}
	}
	parsedVer, err := semver.ParseTolerant(string(version))
	if err != nil {
		return "", domain.ProcessWarning{
			Field:    "version",
			Problem:  domain.ProblemInvalid,
			BadValue: string(version),
			Message:  err.Error()}
	}
	return domain.Version(parsedVer.String()), domain.ProcessWarning{}
}

func validateAccentColor(color string) (string, bool) {
	if color == "" {
		return "", true
	}
	c, err := csscolorparser.Parse(color)
	if err != nil {
		return "", false
	}
	return c.HexString(), true
}

func validateAppName(name string) (string, domain.ProcessWarning) {
	name = strings.TrimSpace(name)
	if name == "" {
		return name, domain.ProcessWarning{
			Field:   "name",
			Message: "App name not specified.",
			Problem: domain.ProblemEmpty}
	}
	c := uniseg.GraphemeClusterCount(name)
	if c > domain.AppNameMaxLength {
		return name, domain.ProcessWarning{
			Field:   "name",
			Message: "App name is long and may not be shown entirely on all devices.",
			Problem: domain.ProblemBig}
	}
	return name, domain.ProcessWarning{}
}

func validateShortDescription(desc string) (string, domain.ProcessWarning) {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return desc, domain.ProcessWarning{
			Field:   "short-description",
			Message: "Short description not specified.",
			Problem: domain.ProblemEmpty}
	}
	c := uniseg.GraphemeClusterCount(desc)
	if c > domain.AppShortDescriptionMaxLength {
		return desc, domain.ProcessWarning{
			Field:   "short-description",
			Message: "Short description is long and may not be shown entirely on all devices.",
			Problem: domain.ProblemBig}
	}
	return desc, domain.ProcessWarning{}
}

// validateLicenseFields performs basic license validation.
func validateLicenseFields(lic, licFile string) domain.ProcessWarning {
	if lic == "" && licFile == "" {
		return domain.ProcessWarning{
			Field:   "license",
			Message: "No license or license file specified.",
			Problem: domain.ProblemEmpty}
	}
	ok, _ := spdxexp.ValidateLicenses([]string{lic})
	if !ok {
		return domain.ProcessWarning{
			Field:   "license",
			Message: "License is not a valid SPDX identifier.",
			Problem: domain.ProblemInvalid}
	}
	return domain.ProcessWarning{}
}

func validateAuthors(authors []domain.ManifestAuthor) ([]domain.ManifestAuthor, []domain.ProcessWarning) {
	warnings := make([]domain.ProcessWarning, 0, 10)

	for i, a := range authors {
		if a.Email != "" {
			err := validator.Email(a.Email)
			if err != nil {
				warnings = append(warnings, domain.ProcessWarning{
					Field:   "authors",
					Problem: domain.ProblemInvalid,
					Message: fmt.Sprintf("Author email is invalid: %s", a.Email)})
				authors[i].Email = ""
			}
		}
		if a.URL != "" {
			err := validator.HttpURL(a.URL)
			if err != nil {
				warnings = append(warnings, domain.ProcessWarning{
					Field:   "authors",
					Problem: domain.ProblemInvalid,
					Message: fmt.Sprintf("Author website is invalid: %s", a.URL)})
				authors[i].URL = ""
			}
		}
	}
	return authors, warnings
}

func validateWebsite(url string) (string, bool) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", true
	}
	err := validator.HttpURL(url)
	if err != nil {
		return "", false
	}
	return url, true
}

// validate icon should be able to work for both app get ops and before forwarding to frontend from remote.
func validateIcon(iconPath string) []domain.ProcessWarning {
	field := "icon"

	f, err := os.Open(iconPath)
	if os.IsNotExist(err) {
		return []domain.ProcessWarning{{
			Field:   field,
			Problem: domain.ProblemNotFound,
			Message: "Icon file does not exist.",
		}}
	}
	if err != nil {
		return []domain.ProcessWarning{{
			Field:   field,
			Problem: domain.ProblemError,
			Message: "Error opening app icon:  " + err.Error(),
		}}
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		return []domain.ProcessWarning{{
			Field:   field,
			Problem: domain.ProblemError,
			Message: "Error getting icon file info:  " + err.Error(),
		}}
	}
	if fInfo.IsDir() {
		return []domain.ProcessWarning{{
			Field:   field,
			Problem: domain.ProblemInvalid,
			Message: "Icon path is a directory",
		}}
	}

	mimeType, err := getFileMimeType(iconPath)
	if err != nil {
		return []domain.ProcessWarning{{
			Field:   field,
			Problem: domain.ProblemError,
			Message: "Error getting app icon mime type:  " + err.Error(),
		}}
	}

	mimeTypes := []string{"image/jpeg", "image/png", "image/svg+xml", "image/webp"}
	typeOk := false
	for _, t := range mimeTypes {
		if t == mimeType {
			typeOk = true
		}
	}
	if !typeOk {
		return []domain.ProcessWarning{{
			Field:   field,
			Problem: domain.ProblemInvalid,
			Message: "App icon type not supported:  " + mimeType + " Jpeg, png, svg and webp are supported.",
		}}
	}

	// get w and h and check: is square and then size.
	var config image.Config
	if mimeType == "image/jpeg" || mimeType == "image/png" {
		config, _, err = image.DecodeConfig(f)
		if err != nil {
			return []domain.ProcessWarning{{
				Field:   field,
				Problem: domain.ProblemError,
				Message: "Error reading app icon file. " + err.Error(),
			}}
		}
	} else if mimeType == "image/webp" {
		config, err = webp.DecodeConfig(f)
		if err != nil {
			return []domain.ProcessWarning{{
				Field:   field,
				Problem: domain.ProblemError,
				Message: "Error reading app icon file. " + err.Error(),
			}}
		}
	}

	warnings := make([]domain.ProcessWarning, 0, 10)
	if mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/webp" {
		if config.Height != config.Width {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   field,
				Problem: domain.ProblemPoorExperience,
				Message: fmt.Sprintf("App icon is not square: %v x %v.", config.Width, config.Height)})
		} else if config.Height < domain.AppIconMinPixelSize {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   field,
				Problem: domain.ProblemSmall,
				Message: fmt.Sprintf("App icon should be at least %v pixels. It is %v pixels.", domain.AppIconMinPixelSize, config.Width)})
		}
	}

	if fInfo.Size() > domain.AppIconMaxFileSize {
		warnings = append(warnings, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemSmall,
			Message: fmt.Sprintf("App icon file is large: %s (under %s is recommended).",
				bytesize.New(float64(fInfo.Size())), bytesize.New(float64(domain.AppIconMaxFileSize))),
		})
	}

	if mimeType == "image/png" {
		// need to open again so the decoder can work from the beginning
		fPng, err := os.Open(iconPath)
		if err != nil {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   field,
				Problem: domain.ProblemError,
				Message: "Error opening PNG app icon: " + err.Error()})
			return warnings
		}
		defer fPng.Close()
		a, err := apng.DecodeAll(fPng)
		if err != nil {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   field,
				Problem: domain.ProblemError,
				Message: "Error decoding PNG app icon: " + err.Error()})
			return warnings
		}
		if len(a.Frames) > 1 {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   field,
				Problem: domain.ProblemPoorExperience,
				Message: "App icon appears to be animated. Non-animated icons are preferred."})
		}
	}

	return warnings
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

func hasWarnings(field string, warnings []domain.ProcessWarning) bool {
	for _, w := range warnings {
		if w.Field == field {
			return true
		}
	}
	return false
}

func hasProblem(field string, problem domain.ProcessProblem, warnings []domain.ProcessWarning) bool {
	for _, w := range warnings {
		if w.Field == field && w.Problem == problem {
			return true
		}
	}
	return false
}
