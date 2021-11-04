package userroutes

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
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

// ApplicationRoutes handles routes for applications uploading, creating, deleting.
type ApplicationRoutes struct {
	AppGetter interface {
		FromRaw(userID domain.UserID, fileData *map[string][]byte, appIDs ...domain.AppID) (domain.AppGetKey, error)
		GetUser(key domain.AppGetKey) (domain.UserID, bool)
		GetMetaData(domain.AppGetKey) (domain.AppGetMeta, error)
		Commit(domain.AppGetKey) (domain.AppID, domain.Version, error)
		Delete(domain.AppGetKey)
	} `checkinject:"required"`
	DeleteApp interface {
		Delete(appID domain.AppID) error
		DeleteVersion(appID domain.AppID, version domain.Version) error
	} `checkinject:"required"`
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetForOwner(domain.UserID) ([]*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	} `checkinject:"required"`
}

func (a *ApplicationRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", a.getApplications)
	r.Post("/", a.postNewApplication)

	r.Route("/{application}", func(r chi.Router) {
		r.Use(a.applicationCtx)
		r.Get("/", a.getApplication)
		r.Delete("/", a.delete)
		r.Post("/version", a.postNewVersion)
		r.With(a.appVersionCtx).Get("/version/{app-version}", a.getVersion)
		r.With(a.appVersionCtx).Delete("/version/{app-version}", a.deleteVersion)
	})

	return r
}

func (a *ApplicationRoutes) getApplication(w http.ResponseWriter, r *http.Request) {
	app, _ := domain.CtxAppData(r.Context())

	appResp := makeAppResp(app)
	appVersions, err := a.AppModel.GetVersionsForApp(app.AppID)
	if err != nil {
		httpInternalServerError(w)
		return
	}
	appResp.Versions = make([]VersionMeta, len(appVersions))
	for j, appVersion := range appVersions {
		appResp.Versions[j] = makeVersionMeta(*appVersion)
	}

	writeJSON(w, appResp)
}
func (a *ApplicationRoutes) getApplications(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	_, ok := query["app-version"]
	if ok {
		a.getAppVersions(w, r)
	} else {
		a.getAllApplications(w, r)
	}
}

func (a *ApplicationRoutes) getAllApplications(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	apps, err := a.AppModel.GetForOwner(userID)
	if err != nil {
		http.Error(w, err.Error(), 500)
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
		httpInternalServerError(w)
		return
	}

	writeJSON(w, respData)
}

func (a *ApplicationRoutes) delete(w http.ResponseWriter, r *http.Request) {
	app, _ := domain.CtxAppData(r.Context())
	err := a.DeleteApp.Delete(app.AppID)
	if err != nil {
		returnError(w, err)
	}
}

func (a *ApplicationRoutes) getAppVersions(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	// check query string
	query := r.URL.Query()
	appVerionIDs, ok := query["app-version"]
	if ok {
		respData := Versions{
			AppVersions: make([]VersionMeta, len(appVerionIDs))}

		for i, id := range appVerionIDs {
			appID, version, err := parseAppVersionID(id)
			if err != nil {
				http.Error(w, "bad app version id", http.StatusBadRequest)
				return
			}
			// first get the app to ensure owner is legit
			app, err := a.AppModel.GetFromID(appID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if app.OwnerID != userID {
				http.Error(w, "app version not owned by user", http.StatusForbidden)
				return
			}
			appVersion, err := a.AppModel.GetVersion(appID, version)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			respData.AppVersions[i] = makeVersionMeta(*appVersion)
		}

		writeJSON(w, respData)
		return
	}
	http.Error(w, "query params not supported", http.StatusNotImplemented)
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
func (a *ApplicationRoutes) postNewApplication(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	query := r.URL.Query()
	keys, ok := query["key"]
	if ok && len(keys) == 1 {
		appID, _, ok := a.commitKey(w, userID, keys[0])
		if !ok { // something went wrong. response sent.
			return
		}
		app, err := a.AppModel.GetFromID(appID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		r = r.WithContext(domain.CtxWithAppData(r.Context(), *app))
		a.getApplication(w, r)
	} else {
		a.handleFilesUpload(r, w, userID)
	}
}

func (a *ApplicationRoutes) postNewVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, _ := domain.CtxAuthUserID(ctx)
	app, _ := domain.CtxAppData(ctx)

	query := r.URL.Query()
	keys, ok := query["key"]
	if ok && len(keys) == 1 {
		appID, version, ok := a.commitKey(w, userID, keys[0])
		if !ok { // something went wrong. response sent.
			return
		}
		appVersion, err := a.AppModel.GetVersion(appID, version)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		r = r.WithContext(domain.CtxWithAppVersionData(ctx, *appVersion))
		a.getVersion(w, r)
	} else {
		a.handleFilesUpload(r, w, userID, app.AppID)
	}
}

func (a *ApplicationRoutes) handleFilesUpload(r *http.Request, w http.ResponseWriter, userID domain.UserID, appIDs ...domain.AppID) {
	fileData, err := a.extractFiles(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if len(*fileData) == 0 {
		http.Error(w, "no files in request", http.StatusBadRequest)
	}
	appGetKey, err := a.AppGetter.FromRaw(userID, fileData, appIDs...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fileMeta, err := a.AppGetter.GetMetaData(appGetKey)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	writeJSON(w, fileMeta)
}

func (a *ApplicationRoutes) extractFiles(r *http.Request) (*map[string][]byte, error) {
	fileData := map[string][]byte{}

	// copied from http://sanatgersappa.blogspot.com/2013/03/handling-multiple-file-uploads-in-go.html
	// streaming version
	reader, err := r.MultipartReader()
	if err != nil {
		a.getLogger("extractFiles(), r.MultipartReader()").Error(err)
		return &fileData, nil
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		_, params, err := mime.ParseMediaType(part.Header["Content-Disposition"][0])
		if err != nil {
			return nil, err
		}
		filename := params["filename"]
		if filename == "" {
			continue
		}

		buf := &bytes.Buffer{}
		buf.ReadFrom(part) //maybe limit bytes to read to avert file bomb.
		fileData[filename] = buf.Bytes()
	}

	return &fileData, nil
}

func (a *ApplicationRoutes) commitKey(w http.ResponseWriter, userID domain.UserID, key string) (domain.AppID, domain.Version, bool) {
	// TODO basic validation on key
	keyUser, ok := a.AppGetter.GetUser(domain.AppGetKey(key))
	if !ok {
		w.WriteHeader(http.StatusGone)
		return domain.AppID(0), domain.Version(""), false
	}
	if keyUser != userID {
		w.WriteHeader(http.StatusForbidden)
		return domain.AppID(0), domain.Version(""), false
	}
	appID, appVersion, err := a.AppGetter.Commit(domain.AppGetKey(key))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return domain.AppID(0), domain.Version(""), false
	}
	return appID, appVersion, true
}

func (a *ApplicationRoutes) getVersion(w http.ResponseWriter, r *http.Request) {
	appVersion, _ := domain.CtxAppVersionData(r.Context())
	respData := VersionMeta{
		AppID:      appVersion.AppID,
		AppName:    appVersion.AppName,
		Version:    appVersion.Version,
		Schema:     appVersion.Schema,
		APIVersion: appVersion.APIVersion,
		Created:    appVersion.Created}

	writeJSON(w, respData)
}

func (a *ApplicationRoutes) deleteVersion(w http.ResponseWriter, r *http.Request) {
	version, _ := domain.CtxAppVersionData(r.Context())

	err := a.DeleteApp.DeleteVersion(version.AppID, version.Version)
	if err != nil {
		if err == domain.ErrAppVersionInUse {
			http.Error(w, "appspaces use this version of app", http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

func (a *ApplicationRoutes) applicationCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := domain.CtxAuthUserID(r.Context())

		appIDStr := chi.URLParam(r, "application")

		appIDInt, err := strconv.Atoi(appIDStr)
		if err != nil {
			returnError(w, err)
			return
		}
		appID := domain.AppID(appIDInt)

		app, err := a.AppModel.GetFromID(appID)
		if err != nil {
			if err == sql.ErrNoRows {
				returnError(w, errNotFound)
			} else {
				returnError(w, err)
			}
			return
		}
		if app.OwnerID != userID {
			returnError(w, errForbidden)
			return
		}

		r = r.WithContext(domain.CtxWithAppData(r.Context(), *app))

		next.ServeHTTP(w, r)
	})
}

func (a *ApplicationRoutes) appVersionCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		app, ok := domain.CtxAppData(ctx) //maybe check there is an app in context.
		if !ok {
			a.getLogger("appVersionCtx").Error(errors.New("app data missing from Context"))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		versionStr := chi.URLParam(r, "app-version")

		// TODO validate / normalize version string

		version, err := a.AppModel.GetVersion(app.AppID, domain.Version(versionStr))
		if err != nil {
			if err == sql.ErrNoRows {
				returnError(w, errNotFound)
			} else {
				returnError(w, err)
			}
			return
		}

		r = r.WithContext(domain.CtxWithAppVersionData(ctx, *version))

		next.ServeHTTP(w, r)
	})
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
