package appops

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/blang/semver/v4"
	"github.com/inhies/go-bytesize"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

type appGetData struct {
	key                domain.AppGetKey
	url                string
	locationKey        string
	userID             domain.UserID
	hasAppID           bool
	appID              domain.AppID
	sandbox            domain.SandboxI
	hasListing         bool                   // apparently unused
	listing            domain.AppListingFetch // listing that we fetch at app creation time
	autoRefreshListing bool                   // user's choice on whether listing should be refreshed automatically
}

type subscriber struct {
	hasKey bool
	key    domain.AppGetKey
	ch     chan domain.AppGetEvent
	// hasUserID, hasAppID
}

// AppGetter handles incoming application files
// It can stash them temporarily
// and examine its metadata for consistency
// And it can commit files to become an actual app version
type AppGetter struct {
	AppFilesModel interface {
		ExtractPackage(locationKey string) error
		GetManifestSize(string) (int64, error)
		ReadManifest(string) (domain.AppVersionManifest, error)
		WriteRoutes(string, []byte) error
		WriteFileLink(string, string, string) error
		Delete(string) error
	} `checkinject:"required"`
	AppLocation2Path interface {
		Files(string) string
	} `checkinject:"required"`
	AppModel interface {
		Create(domain.UserID) (domain.AppID, error)
		CreateFromURL(domain.UserID, string, bool, domain.AppListingFetch) (domain.AppID, error)
		GetAppUrlListing(domain.AppID) (domain.AppListing, domain.AppURLData, error)
		CreateVersion(domain.AppID, string, domain.AppVersionManifest) (domain.AppVersion, error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	} `checkinject:"required"`
	AppLogger interface {
		Log(locationKey string, source string, message string)
	} `checkinject:"required"`
	RemoteAppGetter interface {
		EnsureFreshListing(domain.AppID) (domain.AppListing, domain.AppURLData, error)
		FetchValidListing(url string) (domain.AppListingFetch, error)
		FetchPackageJob(url string) (string, error)
	} `checkinject:"required"`
	SandboxManager interface {
		ForApp(appVersion *domain.AppVersion) (domain.SandboxI, error)
	} `checkinject:"required"`
	AppRoutes interface {
		ValidateRoutes(routes []domain.AppRoute) error
	} `checkinject:"required"`
	AppGetterEvents interface {
		Send(domain.AppGetEvent)
	} `checkinject:"required"`

	keysMux sync.Mutex
	keys    map[domain.AppGetKey]appGetData
	results map[domain.AppGetKey]domain.AppGetMeta

	eventsMux sync.Mutex
	lastEvent map[domain.AppGetKey]domain.AppGetEvent // stash the last event so we can
}

// Init creates the map [and starts the timers]
func (g *AppGetter) Init() {
	g.keys = make(map[domain.AppGetKey]appGetData)
	g.results = make(map[domain.AppGetKey]domain.AppGetMeta)
	g.lastEvent = make(map[domain.AppGetKey]domain.AppGetEvent)

	// TODO have a way to get currenttly processing (or awaiting commit) new apps (by user)
	// and new versions (by app id)
	// this may prevent duplicate app gets.
}

func (g *AppGetter) Stop() {
	g.keysMux.Lock()
	keys := make([]domain.AppGetKey, 0)
	for key := range g.keys {
		keys = append(keys, key)
	}
	g.keysMux.Unlock()

	for _, key := range keys {
		g.Delete(key)
	}
}

// InstallFromURL installs a new app from a URL
func (g *AppGetter) InstallFromURL(userID domain.UserID, listingURL string, version domain.Version, autoRefreshListing bool) (domain.AppGetKey, error) {
	data := g.set(appGetData{
		url:                listingURL,
		userID:             userID,
		autoRefreshListing: autoRefreshListing,
	})

	go func() {
		g.sendEvent(data, domain.AppGetEvent{Step: "Fetching listing"})

		// re-validate listing URL? Make sure it parses or we'll get a fail somewhere down the line.
		err := validator.HttpURL(listingURL)
		if err != nil {
			g.appendErrorResult(data.key, fmt.Sprintf("Error validating listing URL %v: %v", listingURL, err.Error()))
			g.sendEvent(data, domain.AppGetEvent{Done: true})
			return
		}

		listingFetch, err := g.RemoteAppGetter.FetchValidListing(listingURL)
		if err != nil {
			g.appendErrorResult(data.key, err.Error())
			g.sendEvent(data, domain.AppGetEvent{Done: true})
			return
		}
		if listingFetch.NewURL != "" {
			g.appendErrorResult(data.key, "App listing has moved. New URL: "+listingFetch.NewURL)
			g.sendEvent(data, domain.AppGetEvent{Done: true})
			return
		}

		g.setListing(data.key, listingFetch)

		var versionListing domain.AppListingVersion
		var ok bool
		if version == domain.Version("") {
			version = listingFetch.LatestVersion
		}
		versionListing, ok = listingFetch.Listing.Versions[version]
		if !ok {
			g.appendErrorResult(data.key, "Unable to find the requested version in the app listing: "+string(version))
			g.sendEvent(data, domain.AppGetEvent{Done: true})
			return
		}

		// Later we'll use an actual job to fetch and install the app version to prevent duplicates
		// (Although since this is a new app install, a dedupe job is not necessary?)

		packageURL, err := URLFromListing(listingURL, listingFetch.Listing.Base, versionListing.Package)
		if err != nil {
			// Shouldn't happen. All URLs should be checked before we get to this step.
			g.getLogger("InstallFromURL").AddNote("Unexpected error fron URLFromListing").Error(err)
			g.appendErrorResult(data.key, "Unexpected error. This is a problem in Dropserver. Please check the logs and file a bug.")
			g.sendEvent(data, domain.AppGetEvent{Done: true})
			return
		}

		g.installFromURL(data.key, packageURL)
	}()

	return data.key, nil
}

func (g *AppGetter) InstallNewVersionFromURL(userID domain.UserID, appID domain.AppID, version domain.Version) (domain.AppGetKey, error) {
	listing, urlData, err := g.RemoteAppGetter.EnsureFreshListing(appID)
	if err != nil {
		return domain.AppGetKey(""), err
	}
	if urlData.LastResult == "error" {
		return domain.AppGetKey(""), errors.New("unable to proceed because the last attempt to fetch the app listing resulted in an error")
	}
	data := g.set(appGetData{
		url:      urlData.URL,
		userID:   userID,
		hasAppID: true,
		appID:    appID,
	})

	versionListing, ok := listing.Versions[version]
	if !ok {
		g.appendErrorResult(data.key, "Unable to find the requested version in the app listing: "+string(version))
		g.sendEvent(data, domain.AppGetEvent{Done: true})
		return data.key, err
	}

	packageURL, err := URLFromListing(urlData.URL, listing.Base, versionListing.Package)
	if err != nil {
		// Shouldn't happen. All URLs should be checked before we get to this step.
		g.getLogger("InstallFromURL").AddNote("Unexpected error fron URLFromListing").Error(err)
		g.appendErrorResult(data.key, "Unexpected error. This is a problem in Dropserver. Please check the logs and file a bug.")
		g.sendEvent(data, domain.AppGetEvent{Done: true})
		return data.key, err
	}

	go g.installFromURL(data.key, packageURL)

	return data.key, nil
}

func (g *AppGetter) installFromURL(key domain.AppGetKey, packageURL string) {
	data, ok := g.get(key)
	if !ok {
		return //error?
	}
	g.sendEvent(data, domain.AppGetEvent{Step: "Fetching app package"})

	locationKey, err := g.RemoteAppGetter.FetchPackageJob(packageURL)
	if locationKey != "" {
		data, ok = g.setLocationKey(data.key, locationKey)
		if !ok {
			return
		}
	}
	if err != nil {
		g.appendErrorResult(data.key, fmt.Sprintf("Error fetching package at %v: %v", packageURL, err.Error()))
		g.sendEvent(data, domain.AppGetEvent{Done: true})
		return
	}

	g.sendEvent(data, domain.AppGetEvent{Step: "Extracting app package"})

	err = g.AppFilesModel.ExtractPackage(locationKey)
	if err != nil {
		g.appendErrorResult(data.key, "Error extracting app package: "+err.Error())
		g.sendEvent(data, domain.AppGetEvent{Done: true})
		return
	}

	g.processApp(data)

	g.errorFromWarnings(data.key, false)

	results, ok := g.GetResults(data.key)
	if !ok {
		return
	}

	if len(results.Errors) == 0 && len(results.Warnings) != 0 {
		g.sendEvent(data, domain.AppGetEvent{Input: "commit"}) // user commit installation since there are warnings.
	} else {
		_, _, err = g.Commit(data.key)
		if err != nil {
			g.appendErrorResult(data.key, "Error committing app: "+err.Error())
		}
		g.sendEvent(data, domain.AppGetEvent{Done: true})
	}
}

// InstallPackage extracts the package at location key and
// begins process of extracting and verifying all data.
func (g *AppGetter) InstallPackage(userID domain.UserID, locationKey string, appIDs ...domain.AppID) (domain.AppGetKey, error) {
	data := appGetData{
		userID:      userID,
		locationKey: locationKey,
	}
	if len(appIDs) == 1 {
		data.hasAppID = true
		data.appID = appIDs[0]
	}
	data = g.set(data)

	g.sendEvent(data, domain.AppGetEvent{Step: "Unpackaging app"})

	go func() {
		err := g.AppFilesModel.ExtractPackage(locationKey)
		if err != nil {
			g.appendErrorResult(data.key, err.Error())
			g.sendEvent(data, domain.AppGetEvent{Done: true})
			return
		}
		g.processApp(data)
		g.errorFromWarnings(data.key, false)
		g.sendEvent(data, domain.AppGetEvent{Input: "commit"})
	}()

	return data.key, nil
}

// Reprocess performs the app processing steps again,
// replacing the results upon completion.
// Currently only intended for use by ds-dev.
func (g *AppGetter) Reprocess(userID domain.UserID, appID domain.AppID, locationKey string) (domain.AppGetKey, error) {

	data := appGetData{
		userID:      userID, // not clear whether we'll need userID or appID in here for ds-dev.
		hasAppID:    true,
		appID:       appID,
		locationKey: locationKey}
	data = g.set(data)

	g.AppLogger.Log(locationKey, "ds-host", "Reprocessing app version metadata")

	g.sendEvent(data, domain.AppGetEvent{Step: "Starting reprocess"})

	go func() {
		g.processApp(data)
		g.errorFromWarnings(data.key, true)
		g.sendEvent(data, domain.AppGetEvent{Done: true})
	}()

	return data.key, nil
}

func (g *AppGetter) errorFromWarnings(key domain.AppGetKey, devOnly bool) {
	meta, ok := g.GetResults(key)
	if !ok {
		return
	}
	err := errorFromWarnings(meta.Warnings, devOnly)
	if err != nil {
		g.appendErrorResult(key, err.Error())
	}
}

// GetUser returns the user associated with the key
// Used to authorize a request for data on that key
func (g *AppGetter) GetUser(key domain.AppGetKey) (domain.UserID, bool) {
	data, ok := g.get(key)
	return data.userID, ok
}

func (g *AppGetter) GetLocationKey(key domain.AppGetKey) (string, bool) {
	data, ok := g.get(key)
	return data.locationKey, ok
}

func (g *AppGetter) processApp(keyData appGetData) {
	err := g.readFilesManifest(keyData)
	abort := g.checkStep(keyData, err, "error reading file manifest")
	if abort {
		return
	}

	err = g.getEntrypoint(keyData)
	abort = g.checkStep(keyData, err, "error determining app entrypoint")
	if abort {
		return
	}

	err = g.getDataFromSandbox(keyData)
	abort = g.checkStep(keyData, err, "error getting data from sandbox")
	if abort {
		return
	}

	g.sendEvent(keyData, domain.AppGetEvent{Step: "Validating data"})

	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return
	}

	cleanManifest, warns := validateManifest(meta.VersionManifest)
	g.setWarningResult(keyData.key, warns...)
	g.setManifestResult(keyData.key, cleanManifest)

	if keyData.hasAppID {
		appVersions, err := g.getVersionSemvers(keyData.appID)
		abort = g.checkStep(keyData, err, "error getting app versions")
		if abort {
			return
		}
		prev, next, warns := validateVersionSequence(cleanManifest.Version, cleanManifest.Schema, appVersions)
		g.setWarningResult(keyData.key, warns...)
		g.setPrevNextResult(keyData.key, prev, next)
	}

	// we may get duplicate warnings for some of these because we are validating twice
	// once in genral manifest validation, and once again in the deeper full-paclage validation
	err = g.validateChangelog(keyData)
	abort = g.checkStep(keyData, err, "error validating changelog")
	if abort {
		return
	}

	err = g.validateLicense(keyData)
	abort = g.checkStep(keyData, err, "error validating app license")
	if abort {
		return
	}

	err = g.validateAppIcon(keyData)
	abort = g.checkStep(keyData, err, "error validating app icon")
	if abort {
		return
	}

	g.AppLogger.Log(keyData.locationKey, "ds-host", "App processing completed successfully")
}
func (g *AppGetter) checkStep(keyData appGetData, err error, errStr string) bool {
	if err != nil {
		if errStr != "" {
			err = fmt.Errorf("%s: %w", errStr, err)
		}
		g.appendErrorResult(keyData.key, "internal error while processing app")
		// Yes, it's internal error. Log or console output makes sense.
		// Still need to let user know why it all went wrong.
		g.getLogger("processApp").Error(err)
		g.sendEvent(keyData, domain.AppGetEvent{Done: true})
		return true
	}
	if g.resultHasError(keyData.key) {
		// errors in processing app meta data, probably app fault.
		g.sendEvent(keyData, domain.AppGetEvent{Done: true})
		return true
	}
	return false
}

func (g *AppGetter) readFilesManifest(keyData appGetData) error {
	g.sendEvent(keyData, domain.AppGetEvent{Step: "Reading manifest from app files"})

	size, err := g.AppFilesModel.GetManifestSize(keyData.locationKey)
	if err != nil {
		if err == domain.ErrAppManifestNotFound {
			g.appendErrorResult(keyData.key, "Application manifest file not found")
			return nil
		}
		return err
	}
	if size > domain.AppManifestMaxFileSize {
		g.appendErrorResult(keyData.key, "Application manifest file is too large")
		return nil
	}

	manifest, err := g.AppFilesModel.ReadManifest(keyData.locationKey)
	if err != nil {
		return err
	}

	g.setManifestResult(keyData.key, manifest)

	return nil
}

func (g *AppGetter) getEntrypoint(keyData appGetData) error {
	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return errors.New("could not get results metadata")
	}
	entry, ok := validatePackagePath(meta.VersionManifest.Entrypoint)
	if !ok {
		g.appendErrorResult(keyData.key, "Application entrypoint is invalid")
		return nil
	} else if entry != "" {
		meta.VersionManifest.Entrypoint = entry
		g.setManifestResult(keyData.key, meta.VersionManifest)
		return nil
	}

	for _, d := range []string{"app.ts", "app.js"} {
		_, err := os.Stat(filepath.Join(g.AppLocation2Path.Files(keyData.locationKey), d))
		if os.IsNotExist(err) {
			// file not there, no op.
		} else if err != nil {
			return err
		} else if entry != "" {
			g.appendErrorResult(keyData.key, "Application contains both app.js and app.ts and does not specify which one to use in the manifest")
			return nil
		} else {
			entry = d
		}
	}
	if entry == "" {
		g.appendErrorResult(keyData.key, "Application has neither app.js or app.ts and no entrypoint in the manifest")
		return nil
	}
	meta.VersionManifest.Entrypoint = entry
	g.setManifestResult(keyData.key, meta.VersionManifest)
	return nil
}

// This will become a readAppMeta to get routes and migrations and all other data.
func (g *AppGetter) getDataFromSandbox(keyData appGetData) error {
	g.sendEvent(keyData, domain.AppGetEvent{Step: "Starting sandbox to get app data"})

	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return errors.New("could not get results metadata")
	}

	s, err := g.SandboxManager.ForApp(&domain.AppVersion{
		LocationKey: keyData.locationKey,
		Entrypoint:  meta.VersionManifest.Entrypoint,
	})
	if err != nil {
		// This could very well be an app error!
		return err
	}
	defer s.Graceful()

	keyData, ok = g.setSandbox(keyData.key, s)
	if !ok {
		err = errors.New("unable to set sandbox to app get data")
		g.getLogger("getDataFromSandbox, g.setSandbox").Error(err)
		return err
	}

	// Set a timeout so that this sandbox doesn't run forever in case of infinite loop or whatever.
	go func(sb domain.SandboxI) {
		time.Sleep(time.Minute) // one minute. Is that enough on heavily used system?
		if sb.Status() < domain.SandboxDead {
			g.getLogger("getDataFromSandbox").Log("sandbox not dead, killing. Location key: " + keyData.locationKey)
			sb.Kill()
		}
	}(s)

	g.sendEvent(keyData, domain.AppGetEvent{Step: "Getting migrations"})

	err = g.getMigrations(keyData, s)
	if err != nil {
		return err
	}

	g.sendEvent(keyData, domain.AppGetEvent{Step: "Getting routes"})

	routesData, err := g.getRoutes(s) // maybe pass meta so getRoutes can set app Errors?
	if err != nil {                   // assume any error returned is internal and fatal.
		return err
	}

	err = g.AppRoutes.ValidateRoutes(routesData) // pass meta, assume err is internal / fatal.
	if err != nil {
		return err
	}

	g.AppLogger.Log(keyData.locationKey, "ds-host", "Writing app routes to disk")
	g.sendEvent(keyData, domain.AppGetEvent{Step: "Writing app routes"})

	routerJson, err := json.Marshal(routesData)
	if err != nil {
		g.getLogger("getDataFromSandbox() json.Marshal").Error(err)
		return err
	}

	err = g.AppFilesModel.WriteRoutes(keyData.locationKey, routerJson)
	if err != nil {
		return err
	}
	return nil
}

func (g *AppGetter) getMigrations(keyData appGetData, s domain.SandboxI) error {
	sent, err := s.SendMessage(domain.SandboxMigrateService, 12, nil)
	if err != nil {
		g.getLogger("getMigrations, s.SendMessage").Error(err)
		return err
	}

	reply, err := sent.WaitReply()
	if err != nil {
		// This one probaly means the sandbox crashed or some such
		g.getLogger("getMigrations, sent.WaitReply").Error(err)
		return err
	}
	err = reply.Error()
	if err != nil {
		// that's an error while running the code
		return nil
	}
	reply.SendOK()

	// Should also verify that the response is command 11?
	g.getLogger("getMigrations").Debug("got migrations payload")

	var migrations []domain.MigrationStep
	err = json.Unmarshal(reply.Payload(), &migrations)
	if err != nil {
		g.getLogger("getMigrations, json.Unmarshal").Error(err)
		g.appendErrorResult(keyData.key, fmt.Sprintf("failed to parse json migrations data: %v", err))
		return nil
	}

	schema := getSchemaFromMigrations(migrations)

	meta, _ := g.GetResults(keyData.key)
	meta.VersionManifest.Migrations = migrations
	meta.VersionManifest.Schema = schema
	g.setManifestResult(keyData.key, meta.VersionManifest)

	return nil
}

// Note this is a versioned API
func (g *AppGetter) getRoutes(s domain.SandboxI) ([]domain.AppRoute, error) {
	sent, err := s.SendMessage(domain.SandboxAppService, 11, nil)
	if err != nil {
		g.getLogger("getRoutes, s.SendMessage").Error(err)
		return nil, err
	}

	reply, err := sent.WaitReply()
	if err != nil {
		// This one probaly means the sandbox crashed or some such
		g.getLogger("getRoutes, sent.WaitReply").Error(err)
		return nil, err
	}

	// Should also verify that the response is command 11?

	var routes []domain.AppRoute

	err = json.Unmarshal(reply.Payload(), &routes)
	if err != nil {
		g.getLogger("getRoutes, json.Unmarshal").Error(err)
		reply.SendError("json unmarshal error")
		return nil, err
	}
	reply.SendOK()

	return routes, nil
}

func (g *AppGetter) getVersionSemvers(appID domain.AppID) ([]appVersionSemver, error) {
	appVersions, err := g.AppModel.GetVersionsForApp(appID)
	if err != nil {
		return nil, err
	}
	ret := make([]appVersionSemver, len(appVersions))
	for i, appVersion := range appVersions {
		sver, err := semver.Parse(string(appVersion.Version))
		if err != nil {
			g.getLogger("getVersionSemvers() semver.Parse").AppID(appID).AddNote(fmt.Sprintf("bad version: %v", appVersion.Version)).Error(err)
			return nil, err
		}
		ret[i] = appVersionSemver{*appVersion, sver}
	}
	return ret, nil
}

func (g *AppGetter) validateChangelog(keyData appGetData) error {
	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return errors.New("could not get results metadata")
	}

	field := "changelog"

	err := g.AppFilesModel.WriteFileLink(keyData.locationKey, "changelog", "")
	if err != nil {
		return err
	}

	// if no changelog specified, look for changelog.txt file by default.
	if meta.VersionManifest.Changelog == "" {
		_, err = os.Stat(filepath.Join(g.AppLocation2Path.Files(keyData.locationKey), "changelog.txt"))
		if errors.Is(err, fs.ErrNotExist) {
			g.setWarningResult(keyData.key, domain.ProcessWarning{
				Field:   field,
				Problem: domain.ProblemEmpty,
				Message: "changelog.txt not found and no changelog file specified"})
			return nil
		} else if err != nil {
			return err
		}
		meta.VersionManifest.Changelog = "changelog.txt"
		g.setManifestResult(keyData.key, meta.VersionManifest)
	}

	clPath, ok := validatePackagePath(meta.VersionManifest.Changelog)
	if !ok {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:    field,
			Problem:  domain.ProblemInvalid,
			Message:  "Changelog path is invalid.",
			BadValue: meta.VersionManifest.Changelog})
		return nil
	}
	meta.VersionManifest.Changelog = clPath // set the normalized path to generated manifest
	g.setManifestResult(keyData.key, meta.VersionManifest)

	f, err := os.Open(filepath.Join(g.AppLocation2Path.Files(keyData.locationKey), clPath))
	if errors.Is(err, fs.ErrNotExist) {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:    field,
			Problem:  domain.ProblemNotFound,
			Message:  "Changelog not found at path.",
			BadValue: meta.VersionManifest.Changelog})
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()

	sVer, err := semver.Parse(string(meta.VersionManifest.Version))
	if err != nil {
		// this shouldn't happen. version should be valid.
		g.getLogger("validateChangelog semver.Parse").Error(err)
		return err
	}
	cl, err := GetValidChangelog(f, sVer)
	if err != nil {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:    field,
			Problem:  domain.ProblemError,
			Message:  "Error parsing the changelog: " + err.Error(),
			BadValue: meta.VersionManifest.Changelog})
		return nil
	}
	if cl == "" {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemEmpty,
			Message: "There is no changelog for this version"})
	}

	err = g.AppFilesModel.WriteFileLink(keyData.locationKey, "changelog", meta.VersionManifest.Changelog)
	if err != nil {
		return err
	}

	return nil
}

func (g *AppGetter) validateLicense(keyData appGetData) error {
	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return errors.New("could not get results metadata")
	}

	fileField := "license-file"

	// first delete the license file link, if any
	err := g.AppFilesModel.WriteFileLink(keyData.locationKey, "license-file", "")
	if err != nil {
		return err
	}

	warning := validateLicenseFields(meta.VersionManifest.License, meta.VersionManifest.LicenseFile)
	g.setWarningResult(keyData.key, warning)

	if meta.VersionManifest.LicenseFile == "" {
		return nil
	}

	lPath, ok := validatePackagePath(meta.VersionManifest.LicenseFile)
	if !ok {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:    fileField,
			Problem:  domain.ProblemInvalid,
			Message:  "Invalid path for license file.",
			BadValue: meta.VersionManifest.LicenseFile})
		return nil
	}
	meta.VersionManifest.LicenseFile = lPath
	g.setManifestResult(keyData.key, meta.VersionManifest)

	lFile := filepath.Join(g.AppLocation2Path.Files(keyData.locationKey), lPath)
	_, err = os.Stat(lFile)
	if os.IsNotExist(err) {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   fileField,
			Message: "License file not found at path.",
			Problem: domain.ProblemNotFound})
		return nil
	} else if err != nil {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   fileField,
			Problem: domain.ProblemError,
			Message: "Error opening license file:  " + err.Error()})
		return nil
	}

	err = g.AppFilesModel.WriteFileLink(keyData.locationKey, "license-file", lPath)
	if err != nil {
		return err
	}
	return nil
}

func (g *AppGetter) validateAppIcon(keyData appGetData) error {
	meta, ok := g.GetResults(keyData.key)
	if !ok {
		return errors.New("could not get results metadata")
	}

	field := "icon"

	// start by removing any app icon link in case this is a reprocess and it was not correctly removed
	err := g.AppFilesModel.WriteFileLink(keyData.locationKey, "app-icon", "")
	if err != nil {
		return err
	}

	icon, ok := validatePackagePath(meta.VersionManifest.Icon)
	if !ok {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:    field,
			Problem:  domain.ProblemInvalid,
			Message:  "Icon path is invalid",
			BadValue: meta.VersionManifest.Icon})
		return nil
	}
	meta.VersionManifest.Icon = icon // set the normalized path to generated manifest
	g.setManifestResult(keyData.key, meta.VersionManifest)
	if meta.VersionManifest.Icon == "" {
		return nil
	}

	icon = filepath.Join(g.AppLocation2Path.Files(keyData.locationKey), icon)
	f, err := os.Open(icon)
	if os.IsNotExist(err) {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemNotFound,
			Message: "Icon file does not exist."})
		return nil
	}
	if err != nil {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemError,
			Message: "Error opening app icon:  " + err.Error()})
		return nil
	}
	defer f.Close()

	fInfo, err := f.Stat()
	if err != nil {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemError,
			Message: "Error getting icon file info:  " + err.Error()})
		return nil
	}
	if fInfo.IsDir() {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemInvalid,
			Message: "Icon path is a directory"})
		return nil
	}
	if fInfo.Size() > domain.AppIconMaxFileSize {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemBig,
			Message: fmt.Sprintf("App icon file is large: %s (under %s is recommended).",
				bytesize.New(float64(fInfo.Size())), bytesize.New(float64(domain.AppIconMaxFileSize))),
		})
		return nil
	}

	iconBytes, err := io.ReadAll(&io.LimitedReader{R: f, N: domain.AppIconMaxFileSize})
	if err != nil {
		g.setWarningResult(keyData.key, domain.ProcessWarning{
			Field:   field,
			Problem: domain.ProblemError,
			Message: "Error reading icon file:  " + err.Error()})
		return nil
	}
	warns := validateIcon(iconBytes)
	g.setWarningResult(keyData.key, warns...)
	ok = true
	for _, w := range warns {
		if w.Problem != domain.ProblemPoorExperience && w.Problem != domain.ProblemSmall {
			ok = false
			break
		}
	}
	if ok {
		err = g.AppFilesModel.WriteFileLink(keyData.locationKey, "app-icon", meta.VersionManifest.Icon)
		if err != nil {
			return err
		}
	}

	return nil
}

// Commit creates either a new app and version, or just a new version
func (g *AppGetter) Commit(key domain.AppGetKey) (domain.AppID, domain.Version, error) {
	keyData, ok := g.get(key) // g.setCommitting
	if !ok {
		err := errors.New("key does not exist")
		g.getLogger("Commit, g.get(key)").Error(err)
		return domain.AppID(0), domain.Version(""), err
	}
	// Here there is a chance Delete will be called while this function is running.
	// We could set a "committing" flag on that keyData to prevent this from happening.
	meta, ok := g.GetResults(key)
	if !ok {
		err := errors.New("results does not exist")
		g.getLogger("Commit, g.getResults").Error(err)
		return domain.AppID(0), domain.Version(""), err
	}

	ev, ok := g.GetLastEvent(key)
	if !ok {
		err := errors.New("last event does not exist")
		g.getLogger("Commit, g.GetLastEvent").Error(err)
		return domain.AppID(0), domain.Version(""), err
	}
	if ev.Done {
		err := errors.New("trying to commit when already Done")
		g.getLogger("Commit").Error(err)
		return domain.AppID(0), domain.Version(""), err
	}

	g.sendEvent(keyData, domain.AppGetEvent{Step: "Committing..."})

	appID := keyData.appID

	if !keyData.hasAppID {
		var aID domain.AppID
		var err error
		if keyData.url == "" {
			aID, err = g.AppModel.Create(keyData.userID) // Here we have to split depending on whether it's url or not.
		} else {
			aID, err = g.AppModel.CreateFromURL(keyData.userID, keyData.url, keyData.autoRefreshListing, keyData.listing)
		}
		if err != nil {
			g.appendErrorResult(key, err.Error())
			g.sendEvent(keyData, domain.AppGetEvent{Done: true})
			return domain.AppID(0), domain.Version(""), err
		}
		g.setAppIDResult(key, aID)
		appID = aID
	}

	version, err := g.AppModel.CreateVersion(appID, keyData.locationKey, meta.VersionManifest)
	if err != nil {
		g.appendErrorResult(key, err.Error())
		g.sendEvent(keyData, domain.AppGetEvent{Done: true})
		return appID, domain.Version(""), err
	}

	g.sendEvent(keyData, domain.AppGetEvent{Done: true})

	return appID, version.Version, nil
}

// Delete removes the files and the key
func (g *AppGetter) Delete(key domain.AppGetKey) {
	appGetData, ok := g.get(key)
	if !ok {
		return
	}

	ev, ok := g.GetLastEvent(key)
	if ok && ev.Done {
		return // do not delete files if done. Have to go through app delete.
	}

	if appGetData.sandbox != nil && appGetData.sandbox.Status() < domain.SandboxDead {
		appGetData.sandbox.Kill()
	}

	if appGetData.locationKey != "" {
		err := g.AppFilesModel.Delete(appGetData.locationKey)
		if err != nil {
			// should be logged by afm. just return.
			return
		}
	}

	g.DeleteKeyData(key)
}

// DeleteKeyData removes the data related to the key
// but leaves files in place.
func (g *AppGetter) DeleteKeyData(key domain.AppGetKey) {
	appGetData, ok := g.del(key)
	if !ok {
		return
	}

	// Send one last event in case there are any subscribers
	g.sendEvent(appGetData, domain.AppGetEvent{Key: key, Done: true, Step: "Deleting processing data"})
	// delete the last_event
	g.eventsMux.Lock()
	delete(g.lastEvent, key)
	g.eventsMux.Unlock()

	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	delete(g.results, key)
}

// keys functions:
func (g *AppGetter) set(d appGetData) appGetData { // createKey
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	var key domain.AppGetKey
	for {
		key = randomKey()
		if _, ok := g.keys[key]; !ok {
			break
		}
	}

	d.key = key
	g.keys[key] = d

	// then init the results map for this key
	g.results[key] = domain.AppGetMeta{
		Key:      key,
		AppID:    d.appID,
		Errors:   make([]string, 0),
		Warnings: make([]domain.ProcessWarning, 0),
	}

	return d
}
func (g *AppGetter) setLocationKey(key domain.AppGetKey, locationKey string) (appGetData, bool) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	data, ok := g.keys[key]
	if !ok {
		return appGetData{}, false
	}
	data.locationKey = locationKey
	g.keys[key] = data
	return data, true
}
func (g *AppGetter) setSandbox(key domain.AppGetKey, sb domain.SandboxI) (appGetData, bool) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	data, ok := g.keys[key]
	if !ok {
		return appGetData{}, false
	}
	data.sandbox = sb
	g.keys[key] = data
	return data, true
}
func (g *AppGetter) setListing(key domain.AppGetKey, listing domain.AppListingFetch) (appGetData, bool) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	data, ok := g.keys[key]
	if !ok {
		return appGetData{}, false
	}
	data.hasListing = true
	data.listing = listing
	g.keys[key] = data
	return data, true
}

func (g *AppGetter) get(key domain.AppGetKey) (appGetData, bool) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	data, ok := g.keys[key]
	return data, ok
}

// del the key and return the key data
func (g *AppGetter) del(key domain.AppGetKey) (appGetData, bool) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	data, ok := g.keys[key]
	delete(g.keys, key)
	return data, ok
}

// results functions:
func (g *AppGetter) appendErrorResult(key domain.AppGetKey, errStr string) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	result, ok := g.results[key]
	if ok {
		result.Errors = append(result.Errors, errStr)
		g.results[key] = result
	}
}
func (g *AppGetter) resultHasError(key domain.AppGetKey) bool {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	result, ok := g.results[key]
	return ok && len(result.Errors) > 0
}
func (g *AppGetter) setWarningResult(key domain.AppGetKey, warnings ...domain.ProcessWarning) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	result, ok := g.results[key]
	if ok {
		result.Warnings = addWarning(result.Warnings, warnings...)
		g.results[key] = result
	}
}
func (g *AppGetter) setManifestResult(key domain.AppGetKey, manifest domain.AppVersionManifest) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	result, ok := g.results[key]
	if ok {
		result.VersionManifest = manifest
		g.results[key] = result
	}
}
func (g *AppGetter) setPrevNextResult(key domain.AppGetKey, prev, next domain.Version) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	result, ok := g.results[key]
	if ok {
		result.PrevVersion = prev
		result.NextVersion = next
		g.results[key] = result
	}
}
func (g *AppGetter) setAppIDResult(key domain.AppGetKey, appID domain.AppID) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	result, ok := g.results[key]
	if ok {
		result.AppID = appID
		g.results[key] = result
	}
}
func (g *AppGetter) GetResults(key domain.AppGetKey) (domain.AppGetMeta, bool) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	m, ok := g.results[key]
	return m, ok
}

// event related functions:

// GetLastEvent returns the last event for the key
func (g *AppGetter) GetLastEvent(key domain.AppGetKey) (domain.AppGetEvent, bool) {
	g.eventsMux.Lock()
	defer g.eventsMux.Unlock()
	e, ok := g.lastEvent[key]
	return e, ok
}

func (g *AppGetter) sendEvent(getData appGetData, ev domain.AppGetEvent) {
	ev.Key = getData.key
	g.eventsMux.Lock()
	defer g.eventsMux.Unlock()

	// TODO Here we could check the last event to ensure we're not sending conflicting signals.
	// But what happens if we do send conflicing signals?
	// Maybe just log it? We can investigate further if we see this in logs.

	ev.OwnerID = getData.userID

	g.lastEvent[ev.Key] = ev

	g.AppGetterEvents.Send(ev)
}

func (g *AppGetter) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("AppGetter")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

// //////////
// random string
const chars36 = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand2 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomKey() domain.AppGetKey {
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars36[seededRand2.Intn(len(chars36))]
	}
	return domain.AppGetKey(string(b))
}
