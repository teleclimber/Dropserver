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
	"github.com/teleclimber/DropServer/internal/validator"
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
		InstallPackage(userID domain.UserID, locationKey string, appIDs ...domain.AppID) (domain.AppGetKey, error)
		GetUser(key domain.AppGetKey) (domain.UserID, bool)
		GetLocationKey(key domain.AppGetKey) (string, bool)
		GetLastEvent(key domain.AppGetKey) (domain.AppGetEvent, bool)
		GetResults(domain.AppGetKey) (domain.AppGetMeta, bool)
		Commit(domain.AppGetKey) (domain.AppID, domain.Version, error)
		Delete(domain.AppGetKey)
	} `checkinject:"required"`
	DeleteApp interface {
		Delete(appID domain.AppID) error
		DeleteVersion(appID domain.AppID, version domain.Version) error
	} `checkinject:"required"`
	AppFilesModel interface {
		SavePackage(io.Reader) (string, error)
	} `checkinject:"required"`
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetForOwner(domain.UserID) ([]*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	} `checkinject:"required"`
	AppLogger interface {
		Get(locationKey string) domain.LoggerI
	} `checkinject:"required"`
}

func (a *ApplicationRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", a.getApplications)
	r.Post("/", a.postNewApplication) // could this be same route for new app and new version? Just include app id in metadata.

	r.Route("/in-process/{app-get-key}", func(r chi.Router) {
		r.Use(a.appGetKeyCtx)
		r.Get("/", a.getInProcess)
		r.Get("/log", a.getInProcessLog)
		r.Post("/", a.commitInProcess)
		r.Delete("/", a.cancelInProcess)
	})

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
	a.handlePackageUpload(r, w, userID)
}

func (a *ApplicationRoutes) postNewVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, _ := domain.CtxAuthUserID(ctx)
	app, _ := domain.CtxAppData(ctx)
	a.handlePackageUpload(r, w, userID, app.AppID)
}

type FilesUploadResp struct {
	Key domain.AppGetKey `json:"app_get_key"`
}

func (a *ApplicationRoutes) handlePackageUpload(r *http.Request, w http.ResponseWriter, userID domain.UserID, appIDs ...domain.AppID) {
	f, _, err := r.FormFile("package")
	if err != nil {
		http.Error(w, "unable to get package file from multipart: "+err.Error(), http.StatusBadRequest)
		return
	}
	// if we capture the header above, we can know the original package filename and propagate as desired.

	loc, err := a.AppFilesModel.SavePackage(f)
	if err != nil {
		http.Error(w, "unable to get package file from multipart: "+err.Error(), http.StatusBadRequest)
		return
	}

	appGetKey, err := a.AppGetter.InstallPackage(userID, loc, appIDs...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	writeJSON(w, FilesUploadResp{Key: appGetKey})
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

type InProcessResp struct {
	LastEvent domain.AppGetEvent `json:"last_event"`
	Meta      domain.AppGetMeta  `json:"meta"`
	// Maybe take manifest out of meta?
	// And add a frontend version of manifest data?
}

// getInProcess returns current status of uploaded or acquired app files
// for both new apps and new versions.
func (a *ApplicationRoutes) getInProcess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appGetKey, _ := domain.CtxAppGetKey(ctx)

	lastEvent, ok := a.AppGetter.GetLastEvent(appGetKey)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	meta := domain.AppGetMeta{}
	if lastEvent.Done {
		meta, ok = a.AppGetter.GetResults(appGetKey)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	writeJSON(w, InProcessResp{LastEvent: lastEvent, Meta: meta})
}

func (a *ApplicationRoutes) getInProcessLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appGetKey, _ := domain.CtxAppGetKey(ctx)
	locationKey, ok := a.AppGetter.GetLocationKey(appGetKey)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	logger := a.AppLogger.Get(locationKey)
	if logger == nil {
		writeJSON(w, domain.LogChunk{})
	}
	chunk, err := logger.GetLastBytes(4 * 1024)
	if err != nil {
		returnError(w, err)
		return
	}
	writeJSON(w, chunk)
}

type AppCommitResp struct {
	AppID   domain.AppID   `json:"app_id"`
	Version domain.Version `json:"version"`
}

// commitInProcess commits the in-process app files.
func (a *ApplicationRoutes) commitInProcess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appGetKey, _ := domain.CtxAppGetKey(ctx)

	appID, version, err := a.AppGetter.Commit(appGetKey)
	if err != nil {
		// error could be not found?
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, AppCommitResp{AppID: appID, Version: version})
}

func (a *ApplicationRoutes) cancelInProcess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appGetKey, _ := domain.CtxAppGetKey(ctx)

	a.AppGetter.Delete(appGetKey)
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

func (a *ApplicationRoutes) appGetKeyCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := domain.CtxAuthUserID(r.Context())

		key := chi.URLParam(r, "app-get-key")

		err := validator.AppGetKey(key)
		if err != nil {
			returnError(w, err)
			return
		}

		appGetKey := domain.AppGetKey(key)

		keyUserID, ok := a.AppGetter.GetUser(appGetKey)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if userID != keyUserID {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		r = r.WithContext(domain.CtxWithAppGetKey(r.Context(), appGetKey))

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
