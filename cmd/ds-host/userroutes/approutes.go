package userroutes

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// GetAppsResp is
type GetAppsResp struct {
	Apps []ApplicationMeta `json:"apps"`
}

// ApplicationMeta is an application's metadata
type ApplicationMeta struct {
	AppID    int           `json:"app_id"`
	AppName  string        `json:"name"`
	Created  time.Time     `json:"created_dt"`
	Versions []VersionMeta `json:"versions"`
}

// VersionMeta is for listing versions of application code
type VersionMeta struct {
	AppID      domain.AppID      `json:"app_id"`
	AppName    string            `json:"app_name"`
	Version    domain.Version    `json:"version"`
	APIVersion domain.APIVersion `json:"api_version"`
	Schema     int               `json:"schema"`
	Created    time.Time         `json:"created_dt"`
}

// Versions should be embedded in application meta?
type Versions struct {
	AppVersions []VersionMeta `json:"app_versions"`
}

var errBadRequest = errors.New("bad request")
var errUnauthorized = errors.New("unauthorized")

// ApplicationRoutes handles routes for applications uploading, creating, deleting.
type ApplicationRoutes struct {
	AppGetter interface {
		FromRaw(userID domain.UserID, fileData *map[string][]byte, appIDs ...domain.AppID) (domain.AppGetKey, error)
		GetUser(key domain.AppGetKey) (domain.UserID, bool)
		GetMetaData(domain.AppGetKey) (domain.AppGetMeta, error)
		Commit(domain.AppGetKey) (domain.AppID, domain.Version, error)
		Delete(domain.AppGetKey)
	}
	AppFilesModel interface {
		Delete(string) error
	}
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetForOwner(domain.UserID) ([]*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
		DeleteVersion(domain.AppID, domain.Version) error
	}
	AppspaceModel interface {
		GetForApp(domain.AppID) ([]*domain.Appspace, error)
	}
}

// ServeHTTP handles http traffic to the application routes
// Namely upload, create new application, delete, ...
func (a *ApplicationRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Authentication == nil || !routeData.Authentication.UserAccount {
		// maybe log it?
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we dun fogged up
	}

	app, err := a.getAppFromPath(routeData)
	if err != nil {
		if errors.Is(err, errBadRequest) {
			http.Error(res, "bad request", http.StatusBadRequest)
		} else if errors.Is(err, errUnauthorized) {
			http.Error(res, "unauthorized", http.StatusUnauthorized)
		} else {
			http.Error(res, "", http.StatusInternalServerError)
		}
		return
	}
	method := req.Method

	if app == nil {
		switch method {
		case http.MethodGet:
			a.getApplications(res, req, routeData)
		case http.MethodPost:
			a.postNewApplication(res, req, routeData)
		default:
			http.Error(res, "bad method for /application", http.StatusBadRequest)
		}
	} else {
		head, tail := shiftpath.ShiftPath(routeData.URLTail)
		routeData.URLTail = tail

		// delete application??

		switch head {
		case "":
			a.getApplication(res, app)
		case "version": // application/<app-id>/version/*
			// get a version from path
			version, err := a.getVersionFromPath(routeData, app.AppID)
			if err != nil {
				http.Error(res, "", http.StatusInternalServerError)
				return
			}

			if version == nil {
				switch req.Method {
				case http.MethodPost:
					a.postNewVersion(app, res, req, routeData)
				default:
					http.Error(res, "bad method for version", http.StatusBadRequest)
				}
			} else {
				switch req.Method {
				case http.MethodGet:
					a.getVersion(res, version)
				case http.MethodDelete:
					a.deleteVersion(res, version)
				default:
					http.Error(res, "bad method for version", http.StatusBadRequest)
				}
			}
		default:
			res.WriteHeader(http.StatusNotFound)
		}
	}
}

func (a *ApplicationRoutes) getApplication(res http.ResponseWriter, app *domain.App) {
	appResp := makeAppResp(*app)
	appVersions, err := a.AppModel.GetVersionsForApp(app.AppID)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	appResp.Versions = make([]VersionMeta, len(appVersions))
	for j, appVersion := range appVersions {
		appResp.Versions[j] = makeVersionMeta(*appVersion)
	}

	writeJSON(res, appResp)
}
func (a *ApplicationRoutes) getApplications(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	query := req.URL.Query()
	_, ok := query["app-version"]
	if ok {
		a.getAppVersions(res, req, routeData)
	} else {
		a.getAllApplications(res, req, routeData)
	}
}

func (a *ApplicationRoutes) getAllApplications(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	apps, err := a.AppModel.GetForOwner(routeData.Authentication.UserID)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}

	respData := GetAppsResp{
		Apps: make([]ApplicationMeta, len(apps))}

	fail := false
	for i, app := range apps {
		appResp := makeAppResp(*app)
		appVersions, err := a.AppModel.GetVersionsForApp(app.AppID)
		if err != nil {
			fail = true
			break
		}
		appResp.Versions = make([]VersionMeta, len(appVersions))
		for j, appVersion := range appVersions {
			appResp.Versions[j] = makeVersionMeta(*appVersion)
		}
		respData.Apps[i] = appResp
	}

	if fail {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(res, respData)
}

func (a *ApplicationRoutes) getAppVersions(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// check query string
	query := req.URL.Query()
	appVerionIDs, ok := query["app-version"]
	if ok {
		respData := Versions{
			AppVersions: make([]VersionMeta, len(appVerionIDs))}

		for i, id := range appVerionIDs {
			appID, version, err := parseAppVersionID(id)
			if err != nil {
				http.Error(res, "bad app version id", http.StatusBadRequest)
				return
			}
			// first get the app to ensure owner is legit
			app, err := a.AppModel.GetFromID(appID)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			if app.OwnerID != routeData.Authentication.UserID {
				http.Error(res, "app version not owned by user", http.StatusForbidden)
				return
			}
			appVersion, err := a.AppModel.GetVersion(appID, version)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}

			respData.AppVersions[i] = makeVersionMeta(*appVersion)
		}

		writeJSON(res, respData)
		return
	}
	http.Error(res, "query params not supported", http.StatusNotImplemented)
}

// NewAppResp returns the new app and nversion metadata
type NewAppResp struct {
	App     domain.App        `json:"app"`
	Version domain.AppVersion `json:"app_version"`
}

// postNewApplication is for Post with no app-id
// if there are files attached send appfilesmodel(?) for storage,
// ..then ask for files metadata, and return along with key.
// if there are no files but there is a key, then create a new app with files found at key.
func (a *ApplicationRoutes) postNewApplication(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	query := req.URL.Query()
	keys, ok := query["key"]
	if ok && len(keys) == 1 {
		appID, _, ok := a.commitKey(res, routeData, keys[0])
		if !ok { // something went wrong. response sent.
			return
		}
		app, err := a.AppModel.GetFromID(appID)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		a.getApplication(res, app)
	} else {
		a.handleFilesUpload(req, res, routeData)
	}
}

func (a *ApplicationRoutes) postNewVersion(app *domain.App, res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	query := req.URL.Query()
	keys, ok := query["key"]
	if ok && len(keys) == 1 {
		appID, version, ok := a.commitKey(res, routeData, keys[0])
		if !ok { // something went wrong. response sent.
			return
		}
		appVersion, err := a.AppModel.GetVersion(appID, version)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		a.getVersion(res, appVersion)
	} else {
		a.handleFilesUpload(req, res, routeData, app.AppID)
	}
}

func (a *ApplicationRoutes) handleFilesUpload(req *http.Request, res http.ResponseWriter, routeData *domain.AppspaceRouteData, appIDs ...domain.AppID) {
	fileData, err := a.extractFiles(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
	}
	if len(*fileData) == 0 {
		http.Error(res, "no files in request", http.StatusBadRequest)
	}
	appGetKey, err := a.AppGetter.FromRaw(routeData.Authentication.UserID, fileData, appIDs...)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}

	fileMeta, err := a.AppGetter.GetMetaData(appGetKey)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}

	writeJSON(res, fileMeta)
}

func (a *ApplicationRoutes) extractFiles(req *http.Request) (*map[string][]byte, error) {
	fileData := map[string][]byte{}

	// copied from http://sanatgersappa.blogspot.com/2013/03/handling-multiple-file-uploads-in-go.html
	// streaming version
	reader, err := req.MultipartReader()
	if err != nil {
		a.getLogger("extractFiles(), req.MultipartReader()").Error(err)
		return &fileData, nil
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if part.FileName() == "" {
			continue
		}

		buf := &bytes.Buffer{}
		buf.ReadFrom(part) //maybe limit bytes to read to avert file bomb.
		fileData[part.FileName()] = buf.Bytes()
	}

	return &fileData, nil
}

func (a *ApplicationRoutes) commitKey(res http.ResponseWriter, routeData *domain.AppspaceRouteData, key string) (domain.AppID, domain.Version, bool) {
	// TODO basic validation on key
	keyUser, ok := a.AppGetter.GetUser(domain.AppGetKey(key))
	if !ok {
		res.WriteHeader(http.StatusGone)
		return domain.AppID(0), domain.Version(""), false
	}
	if keyUser != routeData.Authentication.UserID {
		res.WriteHeader(http.StatusForbidden)
		return domain.AppID(0), domain.Version(""), false
	}
	appID, appVersion, err := a.AppGetter.Commit(domain.AppGetKey(key))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return domain.AppID(0), domain.Version(""), false
	}
	return appID, appVersion, true
}

func (a *ApplicationRoutes) getVersion(res http.ResponseWriter, appVersion *domain.AppVersion) {
	respData := VersionMeta{
		AppID:      appVersion.AppID,
		AppName:    appVersion.AppName,
		Version:    appVersion.Version,
		Schema:     appVersion.Schema,
		APIVersion: appVersion.APIVersion,
		Created:    appVersion.Created}

	writeJSON(res, respData)
}

func (a *ApplicationRoutes) deleteVersion(res http.ResponseWriter, version *domain.AppVersion) {
	appspaces, err := a.AppspaceModel.GetForApp(version.AppID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	found := false
	for _, as := range appspaces {
		if as.AppVersion == version.Version {
			found = true
			break
		}
	}
	if found {
		http.Error(res, "appspaces use this version of app", http.StatusConflict)
		return
	}

	err = a.AppModel.DeleteVersion(version.AppID, version.Version)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}

	err = a.AppFilesModel.Delete(version.LocationKey)
	if err != nil {
		http.Error(res, err.Error(), 500)
	}
}

func (a *ApplicationRoutes) getAppFromPath(routeData *domain.AppspaceRouteData) (*domain.App, error) {
	appIDStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if appIDStr == "" {
		return nil, nil
	}

	appIDInt, err := strconv.Atoi(appIDStr)
	if err != nil {
		return nil, errBadRequest
	}
	appID := domain.AppID(appIDInt)

	app, err := a.AppModel.GetFromID(appID)
	if err != nil {
		return nil, err
	}
	if app.OwnerID != routeData.Authentication.UserID {
		return nil, errUnauthorized
	}

	return app, nil
}

func (a *ApplicationRoutes) getVersionFromPath(routeData *domain.AppspaceRouteData, appID domain.AppID) (*domain.AppVersion, error) {
	versionStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if versionStr == "" {
		return nil, nil
	}

	// minimally check version string for size

	version, err := a.AppModel.GetVersion(appID, domain.Version(versionStr))
	if err != nil {
		return nil, err
	}

	return version, nil
}

func parseAppVersionID(id string) (appID domain.AppID, version domain.Version, err error) {
	pieces := strings.SplitN(id, "-", 2)
	if len(pieces) != 2 {
		err = errors.New("invalid id string for app version")
		return
	}
	IDint, err := strconv.Atoi(pieces[0])
	if err != nil {
		return
	}
	appID = domain.AppID(IDint)

	if len(pieces[1]) == 0 {
		err = errors.New("invalid version string for app version")
	}
	if len(pieces[1]) > 20 { // 20 should be enough for even complex versions?
		err = errors.New("invalid version string for app version")
	}
	version = domain.Version(pieces[1])

	return
}

func makeAppResp(app domain.App) ApplicationMeta {
	return ApplicationMeta{
		AppID:   int(app.AppID),
		AppName: app.Name,
		Created: app.Created}
}

func makeVersionMeta(appVersion domain.AppVersion) VersionMeta {
	return VersionMeta{
		AppID:      appVersion.AppID,
		AppName:    appVersion.AppName,
		Version:    appVersion.Version,
		Schema:     appVersion.Schema,
		APIVersion: appVersion.APIVersion,
		Created:    appVersion.Created}
}

func (a *ApplicationRoutes) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("ApplicationRoutes")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
