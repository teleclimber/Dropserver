package appops

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/blang/semver/v4"
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
		Save(*map[string][]byte) (string, error)
		ReadMeta(string) (*domain.AppFilesMetadata, error)
		WriteRoutes(string, []byte) error
		Delete(string) error
	} `checkinject:"required"`
	AppModel interface {
		Create(domain.UserID, string) (*domain.App, error)
		CreateVersion(domain.AppID, domain.Version, int, domain.APIVersion, string) (*domain.AppVersion, error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	} `checkinject:"required"`
	AppLogger interface {
		Log(locationKey string, source string, message string)
	} `checkinject:"required"`
	SandboxMaker interface {
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

// FromRaw takes raw file data as app files
func (g *AppGetter) FromRaw(userID domain.UserID, fileData *map[string][]byte, appIDs ...domain.AppID) (domain.AppGetKey, error) {
	if len(*fileData) == 0 {
		return domain.AppGetKey(""), errors.New("no files")
	}
	locationKey, err := g.AppFilesModel.Save(fileData)
	if err != nil {
		return domain.AppGetKey(""), err
	}

	g.AppLogger.Log(locationKey, "ds-host", "Reading new app version metadata")

	data := appGetData{
		userID:      userID,
		locationKey: locationKey,
	}
	if len(appIDs) == 1 {
		data.hasAppID = true
		data.appID = appIDs[0]
	}
	data = g.set(data)

	g.sendEvent(data, domain.AppGetEvent{Step: "Starting process"})

	go g.processApp(data)

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
	meta, err := g.validateMetaData(keyData)
	if err != nil {
		// some error occurred, possibly external to
		meta.Errors = append(meta.Errors, "internal error while validating metadata, see log")
		g.setResults(keyData.key, meta)
		g.getLogger("processApp").Error(err)
		g.sendEvent(keyData, domain.AppGetEvent{Done: true, Error: true})
		return
	}
	if len(meta.Errors) != 0 {
		// errors in processing app meta data, probably app fault.
		g.setResults(keyData.key, meta)
		g.sendEvent(keyData, domain.AppGetEvent{Done: true, Error: true})
		return
	}

	g.AppLogger.Log(keyData.locationKey, "ds-host", "App metadata validated, reading routes")

	err = g.getAppRoutes(keyData)
	if err != nil {
		meta.Errors = append(meta.Errors, fmt.Sprintf("error while getting routes: %v", err))
		g.setResults(keyData.key, meta)
		g.sendEvent(keyData, domain.AppGetEvent{Done: true, Error: true})
		return
	}

	g.AppLogger.Log(keyData.locationKey, "ds-host", "App processing completed successfully")

	g.setResults(keyData.key, meta)
	g.sendEvent(keyData, domain.AppGetEvent{Done: true})
}

// validateMetaData tries to read the metadata for the app
// It returns the output and errors that it encounters.
// read everything make sure it makes sense on its own
// if there is an app id compare with existing versions.
// version should be unique, schemas should increment
// TODO After an initial read to determine DS API version, branch to version-specific
func (g *AppGetter) validateMetaData(keyData appGetData) (domain.AppGetMeta, error) {
	g.sendEvent(keyData, domain.AppGetEvent{Step: "Validating metadata"})

	ret := domain.AppGetMeta{Key: keyData.key, Errors: make([]string, 0)}

	filesMetadata, err := g.AppFilesModel.ReadMeta(keyData.locationKey)
	if err != nil {
		if err == domain.ErrAppConfigNotFound {
			ret.Errors = append(ret.Errors, "Application config file not found")
			return ret, nil
		}
		return ret, err
	}

	ret.VersionMetadata = *filesMetadata

	errs, err := g.validateVersion(filesMetadata)
	if err != nil {
		return ret, err
	}
	ret.Errors = append(ret.Errors, errs...)

	if keyData.hasAppID {
		err := g.validateVersionSequence(keyData.appID, filesMetadata, &ret)
		if err != nil {
			return ret, err
		}
	}

	return ret, nil
}

func (g *AppGetter) getAppRoutes(keyData appGetData) error {
	g.sendEvent(keyData, domain.AppGetEvent{Step: "Getting app routes"})

	routerData, err := g.getRouterData(keyData)
	if err != nil {
		return err
	}

	err = g.V0AppRoutes.ValidateRoutes(routerData)
	if err != nil {
		return err
	}

	g.AppLogger.Log(keyData.locationKey, "ds-host", "Writing app routes to disk")
	g.sendEvent(keyData, domain.AppGetEvent{Step: "Writing app routes"})

	routerJson, err := json.Marshal(routerData)
	if err != nil {
		g.getLogger("GetMetaData() json.Marshal").Error(err)
		return err
	}

	err = g.AppFilesModel.WriteRoutes(keyData.locationKey, routerJson)
	if err != nil {
		return err
	}
	return nil
}

// getRouterData fires up a sandbox and retrieves the app version's router data.
// It is exported so it can be used directly by ds-dev
// I wonder if we could make it so app getter can be used by ds-dev as-is?
// Because it essentially has to restart the "processApp" steps entirely on each app change.
// TODO Heads up this is a versioned API!
func (g *AppGetter) getRouterData(data appGetData) ([]domain.V0AppRoute, error) {
	s, err := g.SandboxMaker.ForApp(&domain.AppVersion{LocationKey: data.locationKey})
	if err != nil {
		return nil, err
	}
	defer s.Graceful()

	ok := g.setSandbox(data.key, s)
	if !ok {
		err = errors.New("unable to set sandbox to app get data")
		g.getLogger("getRouterData, g.setSandbox").Error(err)
		return nil, err
	}

	// Set a timeout so that this sandbox doesn't run forever in case of infinite loop or whatever.
	go func(sb domain.SandboxI) {
		time.Sleep(time.Minute) // one minute. Is that enough on heavily used system?
		if sb.Status() != domain.SandboxDead {
			g.getLogger("getRouterData").Log("sandbox not dead, killing. Location key: " + data.locationKey)
			sb.Kill()
		}
	}(s)

	sent, err := s.SendMessage(domain.SandboxAppService, 11, nil)
	if err != nil {
		g.getLogger("getRouterData, s.SendMessage").Error(err)
		return nil, err
	}

	reply, err := sent.WaitReply()
	if err != nil {
		// This one probaly means the sandbox crashed or some such
		g.getLogger("getRouterData, sent.WaitReply").Error(err)
		return nil, err
	}

	// Should also verify that the response is command 11?

	var routes []domain.V0AppRoute

	err = json.Unmarshal(reply.Payload(), &routes)
	if err != nil {
		g.getLogger("getRouterData, json.Unmarshal").Error(err)
		return nil, err
	}

	return routes, nil
}

// validate that app version has a name
// validate that the DS API is usabel in this version of DS
func (g *AppGetter) validateVersion(filesMetadata *domain.AppFilesMetadata) ([]string, error) {
	errs := make([]string, 0)

	if filesMetadata.AppName == "" {
		errs = append(errs, "App name can not be blank")
	}

	if filesMetadata.APIVersion != 0 {
		errs = append(errs, "Unsupported API version")
	}

	_, err := semver.New(string(filesMetadata.AppVersion))
	if err != nil {
		errs = append(errs, err.Error())
	}

	for i, p := range filesMetadata.UserPermissions {
		if p.Key == "" {
			errs = append(errs, "Permission key can not be blank")
		}
		if i+1 < len(filesMetadata.UserPermissions) {
			for _, p2 := range filesMetadata.UserPermissions[i+1:] {
				if p.Key == p2.Key {
					errs = append(errs, "Duplicate permission key: "+p.Key)
				}
			}
		}
	}

	return errs, nil
}

func (g *AppGetter) validateVersionSequence(appID domain.AppID, filesMetadata *domain.AppFilesMetadata, meta *domain.AppGetMeta) error {
	ver, _ := semver.New(string(filesMetadata.AppVersion)) // already validated in validateVersion

	semVersions, errs, err := g.getVersions(appID, *ver)
	if err != nil {
		return err
	}
	if len(errs) != 0 {
		// already an error, we probably can't validate further, bail.
		meta.Errors = append(meta.Errors, errs...)
		return nil
	}

	verIndex, _ := getVerIndex(semVersions, *ver)
	if verIndex != 0 {
		prev := semVersions[verIndex-1]
		if prev.appVersion.Schema > filesMetadata.SchemaVersion {
			meta.Errors = append(meta.Errors, "Previous version has a higher schema")
		}
		meta.PrevVersion = prev.appVersion.Version
	}
	if verIndex != len(semVersions)-1 {
		next := semVersions[verIndex+1]
		if next.appVersion.Schema < filesMetadata.SchemaVersion {
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

func (g *AppGetter) getVersions(appID domain.AppID, newVer semver.Version) ([]semverAppVersion, []string, error) {
	errs := make([]string, 0)

	appVersions, err := g.AppModel.GetVersionsForApp(appID)
	if err != nil {
		return nil, errs, err
	}

	semVersions := make([]semverAppVersion, len(appVersions)+1)
	semVersions[0] = semverAppVersion{semver: newVer}
	for i, appVersion := range appVersions {
		sver, err := semver.New(string(appVersion.Version))
		if err != nil {
			// couldn't parse semver of existing version.
			return nil, errs, err
		}
		cmp := sver.Compare(newVer)
		if cmp == 0 {
			errs = append(errs, "This version aleady exists in this app")
			return nil, errs, nil
		}
		semVersions[i+1] = semverAppVersion{semver: *sver, appVersion: appVersion}
	}

	sort.Slice(semVersions, func(i, j int) bool {
		return semVersions[i].semver.Compare(semVersions[j].semver) == -1
	})
	return semVersions, errs, nil
}

func getVerIndex(semVers []semverAppVersion, ver semver.Version) (int, bool) {
	for i, v := range semVers {
		if v.semver.Compare(ver) == 0 {
			return i, true
		}
	}
	return 0, false
}

// Commit creates either a new app and version, or just a new version
func (g *AppGetter) Commit(key domain.AppGetKey) (domain.AppID, domain.Version, error) {
	keyData, ok := g.get(key) // g.setCommitting
	if !ok {
		return domain.AppID(0), domain.Version(""), errors.New("key does not exist")
	}
	// Here there is a slight chance Delete will be called while this function is running.
	// We could set a "committing" flag on that keyData to prevent this from happening.

	appID := keyData.appID

	filesMetadata, err := g.AppFilesModel.ReadMeta(keyData.locationKey)
	if err != nil {
		return appID, domain.Version(""), err
	}

	if !keyData.hasAppID {
		app, err := g.AppModel.Create(keyData.userID, filesMetadata.AppName)
		if err != nil {
			return domain.AppID(0), domain.Version(""), err
		}
		appID = app.AppID
	}

	version, err := g.AppModel.CreateVersion(appID, filesMetadata.AppVersion, filesMetadata.SchemaVersion, filesMetadata.APIVersion, keyData.locationKey)
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

	if appGetData.sandbox != nil && appGetData.sandbox.Status() != domain.SandboxDead {
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

	// Send one last event in case there are any subscribers
	g.sendEvent(appGetData, domain.AppGetEvent{Key: key, Done: true, Step: "Deleting processing data"})
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

////////////
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
