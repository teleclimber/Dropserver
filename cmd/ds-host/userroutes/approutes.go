package userroutes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// ApplicationRoutes handles routes for applications uploading, creating, deleting.
type ApplicationRoutes struct {
	AppFilesModel domain.AppFilesModel
	AppModel      interface {
		GetFromID(domain.AppID) (*domain.App, domain.Error)
		GetForOwner(domain.UserID) ([]*domain.App, domain.Error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, domain.Error)
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, domain.Error)
		Create(domain.UserID, string) (*domain.App, domain.Error)
		CreateVersion(domain.AppID, domain.Version, int, string) (*domain.AppVersion, domain.Error)
		DeleteVersion(domain.AppID, domain.Version) domain.Error
	}
	AppspaceModel interface {
		GetForApp(domain.AppID) ([]*domain.Appspace, domain.Error)
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

	app, dsErr := a.getAppFromPath(routeData)
	if dsErr != nil {
		dsErr.HTTPError(res)
	}
	method := req.Method

	if app == nil {
		switch method {
		case http.MethodGet:
			a.getAllApplications(res, req, routeData)
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
			version, dsErr := a.getVersionFromPath(routeData, app.AppID)
			if dsErr != nil {
				dsErr.HTTPError(res)
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

func (a *ApplicationRoutes) getAllApplications(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	apps, dsErr := a.AppModel.GetForOwner(routeData.Authentication.UserID)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	respData := GetAppsResp{
		Apps: make([]ApplicationMeta, 0)}

	fail := false
	for _, app := range apps {
		appVersions, dsErr := a.AppModel.GetVersionsForApp(app.AppID)
		if dsErr != nil { // willit error on zer versions found? -> it should not.
			fail = true
			break
		}

		verResp := make([]VersionMeta, 0)
		for _, appVersion := range appVersions {
			verResp = append(verResp, VersionMeta{
				AppName: appVersion.AppName,
				Version: appVersion.Version,
				Schema:  appVersion.Schema,
				Created: appVersion.Created})
		}

		respData.Apps = append(respData.Apps, ApplicationMeta{
			AppID:    int(app.AppID),
			AppName:  app.Name,
			Created:  app.Created,
			Versions: verResp})
	}

	if fail {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(respData)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(respJSON)

}

// postNewApplication is for Post with no app-id
// if there are files attached send appfilesmodel(?) for storage,
// ..then ask for files metadata.
// Create DB row for application and return app-id.
func (a *ApplicationRoutes) postNewApplication(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	fileData := a.extractFiles(req)
	if len(*fileData) > 0 {
		locationKey, dsErr := a.AppFilesModel.Save(fileData)
		if dsErr != nil {
			dsErr.HTTPError(res)
			return
		}

		filesMetadata, dsErr := a.AppFilesModel.ReadMeta(locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)

			// delete the files? ..it really depends on the error.
			return
		}

		app, dsErr := a.AppModel.Create(routeData.Authentication.UserID, filesMetadata.AppName)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)
			return
		}

		version, dsErr := a.AppModel.CreateVersion(app.AppID, filesMetadata.AppVersion, filesMetadata.SchemaVersion, locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)
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
		locationKey, dsErr := a.AppFilesModel.Save(fileData)
		if dsErr != nil {
			dsErr.HTTPError(res)
			return
		}

		filesMetadata, dsErr := a.AppFilesModel.ReadMeta(locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)

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

		version, dsErr := a.AppModel.CreateVersion(app.AppID, filesMetadata.AppVersion, filesMetadata.SchemaVersion, locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)
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

func (a *ApplicationRoutes) deleteVersion(res http.ResponseWriter, version *domain.AppVersion) {
	appspaces, dsErr := a.AppspaceModel.GetForApp(version.AppID)
	if dsErr != nil {
		dsErr.HTTPError(res)
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

	dsErr = a.AppModel.DeleteVersion(version.AppID, version.Version)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	dsErr = a.AppFilesModel.Delete(version.LocationKey)
	if dsErr != nil {
		dsErr.HTTPError(res)
	}
}

func (a *ApplicationRoutes) getAppFromPath(routeData *domain.AppspaceRouteData) (*domain.App, domain.Error) {
	appIDStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if appIDStr == "" {
		return nil, nil
	}

	appIDInt, err := strconv.Atoi(appIDStr)
	if err != nil {
		return nil, dserror.New(dserror.BadRequest)
	}
	appID := domain.AppID(appIDInt)

	app, dsErr := a.AppModel.GetFromID(appID)
	if dsErr != nil {
		return nil, dsErr
	}
	if app.OwnerID != routeData.Authentication.UserID {
		return nil, dserror.New(dserror.Unauthorized)
	}

	return app, nil
}

func (a *ApplicationRoutes) getVersionFromPath(routeData *domain.AppspaceRouteData, appID domain.AppID) (*domain.AppVersion, domain.Error) {
	versionStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if versionStr == "" {
		return nil, nil
	}

	// minimally check version string for size

	version, dsErr := a.AppModel.GetVersion(appID, domain.Version(versionStr))
	if dsErr != nil {
		return nil, dsErr
	}

	return version, nil
}

func (a *ApplicationRoutes) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("ApplicationRoutes")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
