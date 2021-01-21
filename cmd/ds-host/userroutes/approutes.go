package userroutes

import (
	"bytes"
	"encoding/json"
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

// don't we need PostAppReq?

// PostAppResp is response to creating an application
type PostAppResp struct {
	AppMeta ApplicationMeta `json:"app_meta"`
}

// don't we need post version req?

// PostVersionResp is
type PostVersionResp struct {
	VersionMeta VersionMeta `json:"version_meta"`
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
	AppFilesModel interface {
		Save(*map[string][]byte) (string, error)
		ReadMeta(string) (*domain.AppFilesMetadata, error)
		Delete(string) error
	}
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetForOwner(domain.UserID) ([]*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
		Create(domain.UserID, string) (*domain.App, error)
		CreateVersion(domain.AppID, domain.Version, int, domain.APIVersion, string) (*domain.AppVersion, error)
		DeleteVersion(domain.AppID, domain.Version) error
	}
	AppspaceModel interface {
		GetForApp(domain.AppID) ([]*domain.Appspace, error)
	}
}

// post to / to create a new application even if only partially,
// ..it gets an entry in DB along with an ID, which is returned with that first request.
// Subsequent updates, finalizing, etc... all reference the id /:id/ and use patch or update.

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
			appResp.Versions[j] = makeAppVersionResp(*appVersion)
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

			respData.AppVersions[i] = makeAppVersionResp(*appVersion)
		}

		writeJSON(res, respData)
		return
	}
	http.Error(res, "query params not supported", http.StatusNotImplemented)
}

// postNewApplication is for Post with no app-id
// if there are files attached send appfilesmodel(?) for storage,
// ..then ask for files metadata.
// Create DB row for application and return app-id.
func (a *ApplicationRoutes) postNewApplication(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	fileData := a.extractFiles(req)
	if len(*fileData) > 0 {
		locationKey, err := a.AppFilesModel.Save(fileData)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}

		filesMetadata, err := a.AppFilesModel.ReadMeta(locationKey)
		if err != nil {
			http.Error(res, err.Error(), 500)

			// delete the files? ..it really depends on the error.
			return
		}

		app, err := a.AppModel.Create(routeData.Authentication.UserID, filesMetadata.AppName)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}

		version, err := a.AppModel.CreateVersion(app.AppID, filesMetadata.AppVersion, filesMetadata.SchemaVersion, filesMetadata.APIVersion, locationKey)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}

		// Send back exact same thing we would send if doing a GET on applications.
		respData := PostAppResp{
			AppMeta: ApplicationMeta{
				AppID:   int(app.AppID),
				AppName: app.Name,
				Created: app.Created,
				Versions: []VersionMeta{{
					Version: version.Version,
					Schema:  version.Schema,
					Created: version.Created}}}}

		respJSON, err := json.Marshal(respData)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write(respJSON)

	} else {
		http.Error(res, "Got a post but no file data found", http.StatusBadRequest)
	}
}

func (a *ApplicationRoutes) postNewVersion(app *domain.App, res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	fileData := a.extractFiles(req)
	if len(*fileData) > 0 {
		locationKey, err := a.AppFilesModel.Save(fileData)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}

		filesMetadata, err := a.AppFilesModel.ReadMeta(locationKey)
		if err != nil {
			http.Error(res, err.Error(), 500)

			// delete the files? ..it really depends on the error.
			return
		}

		// TODO: here we should check that this version is coherent with previously uploaded versions
		// The frontend performs the checks, but we should repeat them at the backend and fail with bad request if violation is found?
		// actual violations:
		// - version exists
		// - schema and versions don't add up
		//.. that's it. Everything else is user's choice to break.
		// "version exists" is enforced at DB level with an index.
		// so just check versions and schemas.

		version, err := a.AppModel.CreateVersion(app.AppID, filesMetadata.AppVersion, filesMetadata.SchemaVersion, filesMetadata.APIVersion, locationKey)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}

		respData := PostVersionResp{ // actually might reuse createAppResp. ..to reflect uploaded data. Could callit uploadResp?
			VersionMeta: VersionMeta{
				Version: version.Version,
				Schema:  version.Schema,
				Created: version.Created}}

		respJSON, err := json.Marshal(respData)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError) //...
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write(respJSON)
	} else {
		http.Error(res, "Got a post but no file data found", http.StatusBadRequest)
	}
}

func (a *ApplicationRoutes) extractFiles(req *http.Request) *map[string][]byte {
	fileData := map[string][]byte{}

	// copied from http://sanatgersappa.blogspot.com/2013/03/handling-multiple-file-uploads-in-go.html
	// streaming version
	reader, err := req.MultipartReader()
	if err != nil {
		a.getLogger("extractFiles(), req.MultipartReader()").Error(err)
		return &fileData
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		if part.FileName() == "" {
			continue
		}

		buf := &bytes.Buffer{}
		buf.ReadFrom(part) //maybe limit bytes to read to avert file bomb.
		fileData[part.FileName()] = buf.Bytes()
	}

	return &fileData
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

func makeAppVersionResp(appVersion domain.AppVersion) VersionMeta {
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
