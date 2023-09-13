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
	"sort"
	"sync"
	"time"

	"github.com/blang/semver/v4"
	"github.com/github/go-spdx/v2/spdxexp"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type appGetData struct {
	key         domain.AppGetKey
	locationKey string
	userID      domain.UserID
	hasAppID    bool
	appID       domain.AppID
	sandbox     domain.SandboxI
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
		SavePackage(io.Reader) (string, error)
		ExtractPackage(locationKey string) error
		GetManifestSize(string) (int64, error)
		ReadManifest(string) (domain.AppVersionManifest, error)
		GetVersionChangelog(locationKey string, version domain.Version) (string, bool, error)
		WriteRoutes(string, []byte) error
		WriteFileLink(string, string, string) error
		Delete(string) error
	} `checkinject:"required"`
	AppLocation2Path interface {
		Files(string) string
	} `checkinject:"required"`
	AppModel interface {
		Create(domain.UserID) (domain.AppID, error)
		CreateVersion(domain.AppID, string, domain.AppVersionManifest) (domain.AppVersion, error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	} `checkinject:"required"`
	AppLogger interface {
		Log(locationKey string, source string, message string)
	} `checkinject:"required"`
	SandboxManager interface {
		ForApp(appVersion *domain.AppVersion) (domain.SandboxI, error)
	} `checkinject:"required"`
	V0AppRoutes interface {
		ValidateRoutes(routes []domain.V0AppRoute) error
	} `checkinject:"required"`

	keysMux sync.Mutex
	keys    map[domain.AppGetKey]appGetData
	results map[domain.AppGetKey]domain.AppGetMeta

	eventsMux   sync.Mutex
	lastEvent   map[domain.AppGetKey]domain.AppGetEvent // stash the last event so we can
	subscribers []subscriber
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
			g.setResults(data.key, domain.AppGetMeta{Key: data.key, Errors: []string{err.Error()}})
			g.sendEvent(data, domain.AppGetEvent{Done: true, Error: true})
			return
		}
		g.processApp(data)
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

	go g.processApp(data)

	return data.key, nil
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
	meta := domain.AppGetMeta{Key: keyData.key, Errors: make([]string, 0), Warnings: make(map[string]string)}

	err := g.readFilesManifest(keyData, &meta)
	abort := g.checkStep(keyData, meta, err, "error reading file manifest")
	if abort {
		return
	}

	err = g.getEntrypoint(keyData.locationKey, &meta)
	abort = g.checkStep(keyData, meta, err, "error determining app entrypoint")
	if abort {
		return
	}

	err = g.getDataFromSandbox(keyData, &meta)
	abort = g.checkStep(keyData, meta, err, "error getting data from sandbox")
	if abort {
		return
	}

	g.sendEvent(keyData, domain.AppGetEvent{Step: "Validating data"})

	err = validateMigrationSteps(&meta)
	abort = g.checkStep(keyData, meta, err, "error validating migrations")
	if abort {
		return
	}

	err = g.validateAppVersion(keyData, &meta)
	abort = g.checkStep(keyData, meta, err, "error validating app version")
	if abort {
		return
	}

	err = g.validateChangelog(keyData.locationKey, &meta)
	abort = g.checkStep(keyData, meta, err, "error validating changelog")
	if abort {
		return
	}

	err = g.validateLicense(keyData.locationKey, &meta)
	abort = g.checkStep(keyData, meta, err, "error validating app license")
	if abort {
		return
	}

	err = g.validateAppIcon(keyData, &meta)
	abort = g.checkStep(keyData, meta, err, "error validating app icon")
	if abort {
		return
	}
	err = validateAccentColor(&meta)
	abort = g.checkStep(keyData, meta, err, "error validating accent color")
	if abort {
		return
	}

	validateSoftData(&meta)

	g.AppLogger.Log(keyData.locationKey, "ds-host", "App processing completed successfully")

	g.setResults(keyData.key, meta)
	g.sendEvent(keyData, domain.AppGetEvent{Done: true})
}
func (g *AppGetter) checkStep(keyData appGetData, meta domain.AppGetMeta, err error, errStr string) bool {
	if err != nil {
		if errStr != "" {
			err = fmt.Errorf("%s: %w", errStr, err)
		}
		meta.Errors = append(meta.Errors, "internal error while processing app")
		// Yes, it's internal error. Log or console output makes sense.
		// Still need to let user know why it all went wrong.
		g.setResults(keyData.key, meta)
		g.getLogger("processApp").Error(err)
		g.sendEvent(keyData, domain.AppGetEvent{Done: true, Error: true})
		return true
	}
	if len(meta.Errors) != 0 {
		// errors in processing app meta data, probably app fault.
		g.setResults(keyData.key, meta)
		g.sendEvent(keyData, domain.AppGetEvent{Done: true, Error: true})
		return true
	}
	return false
}

func (g *AppGetter) readFilesManifest(keyData appGetData, meta *domain.AppGetMeta) error {
	g.sendEvent(keyData, domain.AppGetEvent{Step: "Reading manifest from app files"})

	size, err := g.AppFilesModel.GetManifestSize(keyData.locationKey)
	if err != nil {
		if err == domain.ErrAppManifestNotFound {
			meta.Errors = append(meta.Errors, "Application manifest file not found")
			return nil
		}
		return err
	}
	if size > domain.AppManifestMaxFileSize {
		meta.Errors = append(meta.Errors, "Application manifest file is too large")
		return nil
	}

	manifest, err := g.AppFilesModel.ReadManifest(keyData.locationKey)
	if err != nil {
		return err
	}
	meta.VersionManifest = manifest
	return nil
}

func (g *AppGetter) getEntrypoint(locationKey string, meta *domain.AppGetMeta) error {
	entry, ok := validatePackagePath(meta.VersionManifest.Entrypoint)
	if !ok {
		meta.Errors = append(meta.Errors, "Application entrypoint is invalid")
		return nil
	} else if entry != "" {
		meta.VersionManifest.Entrypoint = entry
		return nil
	}

	for _, d := range []string{"app.ts", "app.js"} {
		_, err := os.Stat(filepath.Join(g.AppLocation2Path.Files(locationKey), d))
		if os.IsNotExist(err) {
			// file not there, no op.
		} else if err != nil {
			return err
		} else if entry != "" {
			meta.Errors = append(meta.Errors, "Application contains both app.js and app.ts and does not specify which one to use in the manifest")
			return nil
		} else {
			entry = d
		}
	}
	if entry == "" {
		meta.Errors = append(meta.Errors, "Application has neither app.js or app.ts and no entrypoint in the manifest")
		return nil
	}
	meta.VersionManifest.Entrypoint = entry
	return nil
}

// This will become a readAppMeta to get routes and migrations and all other data.
func (g *AppGetter) getDataFromSandbox(keyData appGetData, meta *domain.AppGetMeta) error {
	g.sendEvent(keyData, domain.AppGetEvent{Step: "Starting sandbox to get app data"})

	s, err := g.SandboxManager.ForApp(&domain.AppVersion{
		LocationKey: keyData.locationKey,
		Entrypoint:  meta.VersionManifest.Entrypoint,
	})
	if err != nil {
		// This could very well be an app error!
		return err
	}
	defer s.Graceful()

	ok := g.setSandbox(keyData.key, s)
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

	err = g.getMigrations(keyData, meta, s)
	if err != nil {
		return err
	}

	g.sendEvent(keyData, domain.AppGetEvent{Step: "Getting routes"})

	routesData, err := g.getRoutes(keyData, s) // maybe pass meta so getRoutes can set app Errors?
	if err != nil {                            // assume any error returned is internal and fatal.
		return err
	}

	err = g.V0AppRoutes.ValidateRoutes(routesData) // pass meta, assume err is internal / fatal.
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

func (g *AppGetter) getMigrations(data appGetData, meta *domain.AppGetMeta, s domain.SandboxI) error {
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
		meta.Errors = append(meta.Errors, err.Error())
		return nil
	}
	reply.SendOK()

	// Should also verify that the response is command 11?
	g.getLogger("getMigrations").Debug("got migrations payload")

	var migrations []domain.MigrationStep
	err = json.Unmarshal(reply.Payload(), &migrations)
	if err != nil {
		g.getLogger("getMigrations, json.Unmarshal").Error(err)
		meta.Errors = append(meta.Errors, fmt.Sprintf("failed to parse json migrations data: %v", err))
		return nil
	}
	meta.VersionManifest.Migrations = migrations

	return nil
}

// Note this is a versioned API
func (g *AppGetter) getRoutes(data appGetData, s domain.SandboxI) ([]domain.V0AppRoute, error) {
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

	var routes []domain.V0AppRoute

	err = json.Unmarshal(reply.Payload(), &routes)
	if err != nil {
		g.getLogger("getRoutes, json.Unmarshal").Error(err)
		reply.SendError("json unmarshal error")
		return nil, err
	}
	reply.SendOK()

	return routes, nil
}

func (g *AppGetter) validateAppVersion(keyData appGetData, meta *domain.AppGetMeta) error {
	err := validateVersion(meta)
	if err != nil {
		return err
	}
	if keyData.hasAppID {
		err = g.validateVersionSequence(keyData.appID, meta)
	}
	return err
}

// validateVersionSequence ensures the candidate app version fits
// with existing versions already on system.
func (g *AppGetter) validateVersionSequence(appID domain.AppID, meta *domain.AppGetMeta) error {
	ver, _ := semver.New(string(meta.VersionManifest.Version)) // already validated in validateVersion
	schema := meta.VersionManifest.Schema

	semVersions, appErr, err := g.getVersions(appID, *ver)
	if err != nil {
		return err
	}
	if appErr != "" {
		meta.Errors = append(meta.Errors, appErr)
		return nil
	}

	verIndex, _ := getVerIndex(semVersions, *ver)
	if verIndex != 0 {
		prev := semVersions[verIndex-1]
		if prev.appVersion.Schema > schema {
			meta.Errors = append(meta.Errors, "Previous version has a higher schema")
		}
		meta.PrevVersion = prev.appVersion.Version
	}
	if verIndex != len(semVersions)-1 {
		next := semVersions[verIndex+1]
		if next.appVersion.Schema < schema {
			meta.Errors = append(meta.Errors, "Next version has a lower schema")
		}
		meta.NextVersion = next.appVersion.Version
	}

	return nil
}

type semverAppVersion struct {
	semver     semver.Version
	appVersion *domain.AppVersion
}

func (g *AppGetter) getVersions(appID domain.AppID, newVer semver.Version) ([]semverAppVersion, string, error) {

	appVersions, err := g.AppModel.GetVersionsForApp(appID)
	if err != nil {
		return nil, "", err
	}

	semVersions := make([]semverAppVersion, len(appVersions)+1)
	semVersions[0] = semverAppVersion{semver: newVer}
	for i, appVersion := range appVersions {
		sver, err := semver.New(string(appVersion.Version))
		if err != nil {
			// couldn't parse semver of existing version.
			return nil, "", err
		}
		cmp := sver.Compare(newVer)
		if cmp == 0 {
			return nil, fmt.Sprintf("Version %s of the app aleady exists on the system", newVer), nil
		}
		semVersions[i+1] = semverAppVersion{semver: *sver, appVersion: appVersion}
	}

	sort.Slice(semVersions, func(i, j int) bool {
		return semVersions[i].semver.Compare(semVersions[j].semver) == -1
	})
	return semVersions, "", nil
}

func getVerIndex(semVers []semverAppVersion, ver semver.Version) (int, bool) {
	for i, v := range semVers {
		if v.semver.Compare(ver) == 0 {
			return i, true
		}
	}
	return 0, false
}

func (g *AppGetter) validateChangelog(locationKey string, meta *domain.AppGetMeta) error {
	err := g.AppFilesModel.WriteFileLink(locationKey, "changelog", "")
	if err != nil {
		return err
	}

	// if no changelog specified, look for changelog.txt file by default.
	if meta.VersionManifest.Changelog == "" {
		_, err = os.Stat(filepath.Join(g.AppLocation2Path.Files(locationKey), "changelog.txt"))
		if errors.Is(err, fs.ErrNotExist) {
			meta.Warnings["changelog"] = "changelog.txt not found and no changelog file specified"
			return nil
		} else if err != nil {
			return err
		}
		meta.VersionManifest.Changelog = "changelog.txt"
	}

	clPath, ok := validatePackagePath(meta.VersionManifest.Changelog)
	if !ok {
		meta.Warnings["changelog"] = "Changelog path is invalid"
		return nil
	}
	meta.VersionManifest.Changelog = clPath // set the normalized path to generated manifest

	_, err = os.Stat(filepath.Join(g.AppLocation2Path.Files(locationKey), clPath))
	if errors.Is(err, fs.ErrNotExist) {
		meta.Warnings["changelog"] = "Changelog path is incorrect"
		return nil
	} else if err != nil {
		return err
	}

	err = g.AppFilesModel.WriteFileLink(locationKey, "changelog", meta.VersionManifest.Changelog)
	if err != nil {
		return err
	}

	cl, ok, err := g.AppFilesModel.GetVersionChangelog(locationKey, meta.VersionManifest.Version)
	if !ok {
		meta.Warnings["changelog"] = "There was a problem reading the changelog."
	} else if err != nil {
		return err
	}
	if cl == "" {
		meta.Warnings["changelog"] = "Changelog is blank for this version"
	}

	return nil
}

func (g *AppGetter) validateLicense(locationKey string, meta *domain.AppGetMeta) error {
	// first delete the license file link, if any
	err := g.AppFilesModel.WriteFileLink(locationKey, "license-file", "")
	if err != nil {
		return err
	}

	if meta.VersionManifest.License == "" && meta.VersionManifest.LicenseFile == "" {
		meta.Warnings["license"] = "No license information provided"
		return nil
	}
	ok, _ := spdxexp.ValidateLicenses([]string{meta.VersionManifest.License})
	if !ok {
		meta.Warnings["license"] = "License is not a recognized SPDX identifier"
	}

	if meta.VersionManifest.LicenseFile == "" {
		return nil
	}

	lPath, ok := validatePackagePath(meta.VersionManifest.LicenseFile)
	if !ok {
		meta.Errors = append(meta.Errors, "License file path is invalid")
		return nil
	}
	meta.VersionManifest.LicenseFile = lPath

	lFile := filepath.Join(g.AppLocation2Path.Files(locationKey), lPath)
	_, err = os.Stat(lFile)
	if os.IsNotExist(err) {
		meta.Warnings["license"] = "License file not found at package path " + meta.VersionManifest.LicenseFile
		return nil
	} else if err != nil {
		meta.Warnings["license"] = "Error opening license file:  " + err.Error()
		return nil
	}

	err = g.AppFilesModel.WriteFileLink(locationKey, "license-file", lPath)
	if err != nil {
		return err
	}
	return nil
}

func (g *AppGetter) validateAppIcon(keyData appGetData, meta *domain.AppGetMeta) error {
	// start by removing any app icon link in case this is a reprocess and it was not correctly removed
	err := g.AppFilesModel.WriteFileLink(keyData.locationKey, "app-icon", "")
	if err != nil {
		return err
	}

	icon, ok := validatePackagePath(meta.VersionManifest.Icon)
	if !ok {
		meta.Errors = append(meta.Errors, "App icon path is invalid")
		return nil
	}
	meta.VersionManifest.Icon = icon // set the normalized path to generated manifest
	if meta.VersionManifest.Icon == "" {
		return nil
	}

	icon = filepath.Join(g.AppLocation2Path.Files(keyData.locationKey), icon)
	if validateIcon(meta, icon) {
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
	// Here there is a slight chance Delete will be called while this function is running.
	// We could set a "committing" flag on that keyData to prevent this from happening.
	meta, ok := g.GetResults(key)
	if !ok {
		err := errors.New("results does not exist")
		g.getLogger("Commt, g.getResults").Error(err)
		return domain.AppID(0), domain.Version(""), err
	}

	appID := keyData.appID

	if !keyData.hasAppID {
		aID, err := g.AppModel.Create(keyData.userID)
		if err != nil {
			return domain.AppID(0), domain.Version(""), err
		}
		appID = aID
	}

	version, err := g.AppModel.CreateVersion(appID, keyData.locationKey, meta.VersionManifest)
	if err != nil {
		return appID, domain.Version(""), err
	}

	g.DeleteKeyData(key)

	return appID, version.Version, nil
}

// Delete removes the files and the key
func (g *AppGetter) Delete(key domain.AppGetKey) {
	appGetData, ok := g.get(key)
	if !ok {
		return
	}

	if appGetData.sandbox != nil && appGetData.sandbox.Status() < domain.SandboxDead {
		appGetData.sandbox.Kill()
	}

	err := g.AppFilesModel.Delete(appGetData.locationKey)
	if err != nil {
		// should be logged by afm. just return.
		return
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

	results, _ := g.GetResults(key)

	// Send one last event in case there are any subscribers
	g.sendEvent(appGetData, domain.AppGetEvent{Key: key, Done: true, Error: len(results.Errors) > 0, Step: "Deleting processing data"})
	// unsubscribe the channels listenning for updates to the key
	g.unsubscribeKey(appGetData.key)
	// delete the last_event
	g.eventsMux.Lock()
	delete(g.lastEvent, key)
	g.eventsMux.Unlock()

	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	delete(g.results, key)
}

// keys functions:
func (g *AppGetter) set(d appGetData) appGetData {
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

	return d
}
func (g *AppGetter) setSandbox(key domain.AppGetKey, sb domain.SandboxI) bool {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	data, ok := g.keys[key]
	if !ok {
		return false
	}
	data.sandbox = sb
	g.keys[key] = data
	return true
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
func (g *AppGetter) setResults(key domain.AppGetKey, meta domain.AppGetMeta) {
	g.keysMux.Lock()
	defer g.keysMux.Unlock()
	g.results[key] = meta
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

// SubscribeKey returns the last event and a channel if the process is ongoing
func (g *AppGetter) SubscribeKey(key domain.AppGetKey) (domain.AppGetEvent, <-chan domain.AppGetEvent) {
	g.eventsMux.Lock()
	defer g.eventsMux.Unlock()
	lastEvent, ok := g.lastEvent[key]
	if !ok || lastEvent.Done {
		return lastEvent, nil
	}
	ch := make(chan domain.AppGetEvent)
	g.subscribers = append(g.subscribers, subscriber{hasKey: true, key: key, ch: ch})
	return lastEvent, ch
}
func (g *AppGetter) sendEvent(getData appGetData, ev domain.AppGetEvent) {
	ev.Key = getData.key
	g.eventsMux.Lock()
	defer g.eventsMux.Unlock()

	g.lastEvent[ev.Key] = ev

	for _, s := range g.subscribers {
		if s.hasKey && ev.Key == s.key {
			s.ch <- ev
		}
		// else if hasUserID; else if hasAppID ..
	}
}
func (g *AppGetter) unsubscribeKey(key domain.AppGetKey) { // TODO please at least test this.
	g.eventsMux.Lock()
	defer g.eventsMux.Unlock()
	k := 0
	for _, s := range g.subscribers {
		if s.hasKey && s.key == key {
			close(s.ch)
		} else {
			g.subscribers[k] = s
			k++
		}
	}
	g.subscribers = g.subscribers[:k]
}
func (g *AppGetter) Unsubscribe(ch <-chan domain.AppGetEvent) {
	g.eventsMux.Lock()
	defer g.eventsMux.Unlock()
	for i, s := range g.subscribers {
		if s.ch == ch {
			g.subscribers[i] = g.subscribers[len(g.subscribers)-1]
			g.subscribers = g.subscribers[:len(g.subscribers)-1]
			close(s.ch)
			return
		}
	}
	g.getLogger("Unsubscribe").Log("Failed to find subscriber channel.")
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
