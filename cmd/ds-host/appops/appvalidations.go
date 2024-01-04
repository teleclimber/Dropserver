package appops

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"path"
	"slices"
	"sort"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/webp"

	"github.com/blang/semver/v4"
	"github.com/github/go-spdx/v2/spdxexp"
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

type appVersionSemver struct {
	domain.AppVersion
	semver semver.Version
}

func validateVersionSequence(version domain.Version, schema int, appVersions []appVersionSemver) (prev, next domain.Version, warnings []domain.ProcessWarning) {
	ver, _ := semver.Parse(string(version))

	for _, v := range appVersions {
		if v.semver.Equals(ver) {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   "version-sequence", // could this be "version"?
				Problem: domain.ProblemInvalid,
				Message: "Version is already installed",
			})
			return
		}
	}

	sort.Slice(appVersions, func(i, j int) bool {
		return appVersions[i].semver.Compare(appVersions[j].semver) == -1
	})

	var p domain.AppVersion
	for _, v := range appVersions {
		if v.semver.LT(ver) {
			p = v.AppVersion
		} else {
			break
		}
	}
	if p != (domain.AppVersion{}) {
		if p.Schema > schema {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   "version-sequence", // is this "schema" or "version" or "version-sequence"?
				Problem: domain.ProblemInvalid,
				Message: fmt.Sprintf("Data schema (%v) is less than previous version (%v)", schema, p.Schema),
			})
		}
		prev = p.Version
	}

	slices.Reverse(appVersions)

	var n domain.AppVersion
	for _, v := range appVersions {
		if v.semver.GT(ver) {
			n = v.AppVersion
		} else {
			break
		}
	}
	if n != (domain.AppVersion{}) {
		if n.Schema < schema {
			warnings = append(warnings, domain.ProcessWarning{
				Field:   "version-sequence",
				Problem: domain.ProblemInvalid,
				Message: fmt.Sprintf("Data schema (%v) is greater than next version (%v)", schema, n.Schema),
			})
		}
		next = n.Version
	}

	return
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

// GetValidChangelog returns the part of the changelog readable from r
// that matches version as a string. The line containing the version
// is not included in the returned string.
func GetValidChangelog(r io.Reader, version semver.Version) (string, error) {
	ret := ""
	found := false
	skipEmptyLine := false

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())

		parsedVer, verErr := semver.ParseTolerant(l)

		if !found && verErr == nil && version.EQ(parsedVer) {
			found = true
		} else if found && verErr == nil {
			break // found another version, so break
		} else if found {
			if l != "" || !skipEmptyLine {
				ret += l + "\n"
			}
			if l == "" {
				skipEmptyLine = true
			} else {
				skipEmptyLine = false
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err // An error while scanning is likely due to a bad changelog file.
	}

	return strings.TrimSpace(ret), nil
}

func validateIcon(byteSlice []byte) []domain.ProcessWarning {
	field := "icon"

	mimeType := http.DetectContentType(byteSlice)

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
	var err error
	if mimeType == "image/jpeg" || mimeType == "image/png" {
		config, _, err = image.DecodeConfig(bytes.NewReader(byteSlice))
		if err != nil {
			return []domain.ProcessWarning{{
				Field:   field,
				Problem: domain.ProblemError,
				Message: "Error reading app icon file. " + err.Error(),
			}}
		}
	} else if mimeType == "image/webp" {
		config, err = webp.DecodeConfig(bytes.NewReader(byteSlice))
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

	if mimeType == "image/png" {
		a, err := apng.DecodeAll(bytes.NewReader(byteSlice))
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

// errorFromWarnings returns an error if any warning
// makes the app uninstallable
func errorFromWarnings(warnings []domain.ProcessWarning, devOnly bool) error {
	//invalid migrations prevent app from running because we can't migrate to schema
	if hasProblem("migrations", domain.ProblemInvalid, warnings) {
		return errors.New("invalid migrations")
	}
	// similar for schema. Any issue here is a dealbreaker
	if hasWarnings("schema", warnings) {
		return errors.New("invalid schema")
	}
	// version sequence has to be clean
	if hasProblem("version-sequence", domain.ProblemInvalid, warnings) {
		return errors.New("invalid version and schema sequence")
	}

	// if devOnly (like ds-dev) then don't trigger errors for remaining problems
	if devOnly {
		return nil
	}

	if hasWarnings("version", warnings) {
		return errors.New("problem with version string")
	}

	return nil
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
