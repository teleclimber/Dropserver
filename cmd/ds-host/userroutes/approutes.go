package userroutes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// ApplicationRoutes handles routes for applications uploading, creating, deleting.
type ApplicationRoutes struct {
	AppFilesModel domain.AppFilesModel
	AppModel      domain.AppModel
	Logger        domain.LogCLientI
}

// post to / to create a new application even if only partially,
// ..it gets an entry in DB along with an ID, which is returned with that first request.
// Subsequent updates, finalizing, etc... all reference the id /:id/ and use patch or update.

// ServeHTTP handles http traffic to the application routes
// Namely upload, create new application, delete, ...
func (a *ApplicationRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Cookie == nil || !routeData.Cookie.UserAccount {
		// maybe log it?
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we dun fogged up
	}

	appIDStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	method := req.Method

	if appIDStr == "" {
		switch method {
		case http.MethodGet:
			a.getAllApplications(res, req, routeData)
		case http.MethodPost:
			// Posting to applictaion implies creating an application. Response will include app-id
			a.postNewApplication(res, req, routeData)
		default:
			http.Error(res, "bad method for /application", http.StatusBadRequest)
		}
	} else {
		// get app from appid + user, error if not found.
		appIDInt, err := strconv.Atoi(appIDStr)
		if err != nil {
			http.Error(res, "bad app id", http.StatusBadRequest)
			return
		}
		appID := domain.AppID(appIDInt)

		app, dsErr := a.AppModel.GetFromID(appID)
		if dsErr != nil {
			dsErr.HTTPError(res)
			return
		}

		version, _ := shiftpath.ShiftPath(tail)

		if version == "" {
			switch method {
			case http.MethodGet:
				// return metadata for app, and maybe versionsetc... check query strings
				// wait are we really going to use this?
				// Not for a long time I think.
				http.Error(res, "get /application/<app-id>", http.StatusNotImplemented)
			case http.MethodPatch:
				// update application data, like its name, etc...
				// You can not change anything about individual versions here.
				http.Error(res, "PATCH /application/<app-id>", http.StatusNotImplemented)
			case http.MethodPost:
				// create a new version. Might involve uploaded files
				// subsequent changes to data associated with version takes place with patch.
				// Here if there is an upload, you have to create key of some sort
				// ..and pass that on to ds-trusted, that will store it in folder <key>.
				// ds-trusted unwraps the files, validates, reads metadata (version)
				// ..then it returns that data so taht ds-host can put it in the DB.
				//http.Error(res, "POST /application/<app-id>", http.StatusNotImplemented)
				a.postNewVersion(app, res, req, routeData)
			default:
				http.Error(res, "bad method for /application/<app-id>", http.StatusBadRequest)
			}

		} else {
			// Operate on version of App.
			// first verify version is in DB
			switch method {
			case http.MethodGet:
				// return metadata about that version, may include stuff about code, um uses, ...
				http.Error(res, "get /application/<app-id>/<version>", http.StatusNotImplemented)

				// do we allow patch?

			default:
				http.Error(res, "bad method for /application/<app-id>/<version>", http.StatusBadRequest)
			}
		}
	}
}

func (a *ApplicationRoutes) getAllApplications(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	apps, dsErr := a.AppModel.GetForOwner(routeData.Cookie.UserID)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	respData := getAppsResp{
		Apps: make([]appMeta, 0)}

	fail := false
	for _, app := range apps {
		appVersions, dsErr := a.AppModel.GetVersionsForApp(app.AppID)
		if dsErr != nil { // willit error on zer versions found? -> it should not.
			fail = true
			break
		}

		verResp := make([]versionMeta, 0)
		for _, appVersion := range appVersions {
			verResp = append(verResp, versionMeta{
				Version: string(appVersion.Version),
				Created: appVersion.Created})
		}

		respData.Apps = append(respData.Apps, appMeta{
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

		appMetadata, dsErr := a.AppFilesModel.ReadMeta(locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)

			// delete the files? ..it really depends on the error.
			return
		}

		app, dsErr := a.AppModel.Create(routeData.Cookie.UserID, appMetadata.AppName)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)
			return
		}

		version, dsErr := a.AppModel.CreateVersion(app.AppID, appMetadata.AppVersion, locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)
			return
		}

		// Send back exact same thing we would send if doing a GET on applications.
		respData := createAppResp{
			AppMeta: appMeta{
				AppID:   int(app.AppID),
				AppName: app.Name,
				Created: app.Created,
				Versions: []versionMeta{{
					Version: string(version.Version),
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

		appMetadata, dsErr := a.AppFilesModel.ReadMeta(locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)

			// delete the files? ..it really depends on the error.
			return
		}

		// TODO: Check that the version does not exist already in DB for this app.
		// .. if it does thn it's a bad request, but should have a user-friendly message.

		// TODO: here we should check that this version is coherent with previously uploaded versions
		// .. like app name, author, version is greater than last greatest
		// though it's not clear we should not proceed.
		// This could be purely frontend...?: compare new version wi
		// -> although this implies sending "application data" that is actually "version data."
		// Maybe that' workable.
		// -> though remember that some appspaces auto-update,
		// ..so when is it OK to migrate data, etc...?
		// Other options is to make this deliberately 2-step?

		// -> another option is to read the dropapp manifest at frontend side prior to upload?

		version, dsErr := a.AppModel.CreateVersion(app.AppID, appMetadata.AppVersion, locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)
			return
		}

		respData := createVersionResp{ // actually might reuse createAppResp. ..to reflect uploaded data. Could callit uploadResp?
			VersionMeta: versionMeta{
				Version: string(version.Version),
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
		a.Logger.Log(domain.INFO, map[string]string{}, "Approutes:extractFiles: Request apparently not multipart form")
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
