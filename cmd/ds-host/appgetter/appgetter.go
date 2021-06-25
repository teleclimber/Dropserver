package appgetter

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/blang/semver/v4"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type AppGetData struct {
	locationKey string
	userID      domain.UserID
	hasAppID    bool
	appID       domain.AppID
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
	}
	AppModel interface {
		Create(domain.UserID, string) (*domain.App, error)
		CreateVersion(domain.AppID, domain.Version, int, domain.APIVersion, string) (*domain.AppVersion, error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	}
	SandboxMaker interface {
		ForApp(appVersion *domain.AppVersion) (domain.SandboxI, error)
	}
	V0AppRoutes interface {
		ValidateStoredRoutes(routes []domain.V0AppRoute) error
	}

	keys map[domain.AppGetKey]AppGetData
}

// Init creates the map [and starts the timers]
func (g *AppGetter) Init() {
	g.keys = make(map[domain.AppGetKey]AppGetData)

	// initiate a timer to periodically clear keys and assocaited files after 10 minutes or so.
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

	data := AppGetData{
		userID:      userID,
		locationKey: locationKey,
	}

	if len(appIDs) == 1 {
		data.hasAppID = true
		data.appID = appIDs[0]
	}

	key := g.set(data)

	return key, nil
}

// GetUser returns the user associated with the key
func (g *AppGetter) GetUser(key domain.AppGetKey) (domain.UserID, bool) {
	// we'll need a lock?
	data, ok := g.keys[key]
	return data.userID, ok
}

// GetMetaData returns metadata, and errors related to , of the files
// read everything make sure it makes sense on its own
// if there is an app id compare with existing versions.
// version should be unique, schemas should increment
// TODO After an initial read to determine DS API version, branch to version-specific
func (g *AppGetter) GetMetaData(key domain.AppGetKey) (domain.AppGetMeta, error) { // will need to figure out what to return.
	keyData, ok := g.keys[key]
	if !ok {
		return domain.AppGetMeta{}, errors.New("Key does not exist")
	}

	ret := domain.AppGetMeta{Key: key, Errors: make([]string, 0)}

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

	// also start a sandbox and read the router data.
	routerData, err := g.GetRouterData(keyData.locationKey)
	if err != nil {
		return ret, err
	}

	err = g.V0AppRoutes.ValidateStoredRoutes(routerData)
	if err != nil {
		return ret, err
	}

	routerJson, err := json.Marshal(routerData)
	if err != nil {
		g.getLogger("GetMetaData() json.Marshal").Error(err)
		return ret, err
	}

	err = g.AppFilesModel.WriteRoutes(keyData.locationKey, routerJson)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

// TODO Heads up this is a versioned API!
func (g *AppGetter) GetRouterData(loc string) ([]domain.V0AppRoute, error) {
	s, err := g.SandboxMaker.ForApp(&domain.AppVersion{LocationKey: loc})
	if err != nil {
		return nil, err
	}

	defer s.Graceful()

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

	fmt.Println("json payload", string(reply.Payload()))

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
	keyData, ok := g.keys[key]
	if !ok {
		return domain.AppID(0), domain.Version(""), errors.New("Key does not exist")
	}

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

	return appID, version.Version, nil
}

// Delete removed the files and the key
func (g *AppGetter) Delete(domain.AppGetKey) {

}

func (g *AppGetter) set(d AppGetData) (key domain.AppGetKey) {
	for {
		key = randomKey()
		if _, ok := g.keys[key]; !ok {
			break
		}
	}

	// set date time on d here

	g.keys[key] = d

	return
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
