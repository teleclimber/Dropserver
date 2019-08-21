package userroutes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

	appID, tail := shiftpath.ShiftPath(routeData.URLTail)
	method := req.Method

	if appID == "" {
		switch method {
		case http.MethodGet:
			// return list of applications for user
			// check query string params for more info on what to return.
			//a.getAllApplications(res, req, routeData)
			http.Error(res, "get /application", http.StatusNotImplemented)
		case http.MethodPost:
			// Posting to applictaion implies creating an application. Response will include app-id
			a.handlePost(res, req, routeData)
		default:
			http.Error(res, "bad method for /application", http.StatusBadRequest)
		}
	} else {
		// get app from appid + user, error if not found.

		version, _ := shiftpath.ShiftPath(tail)

		if version == "" {
			switch method {
			case http.MethodGet:
				// return metadata for app, and maybe versionsetc... check query strings
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
				http.Error(res, "POST /application/<app-id>", http.StatusNotImplemented)
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

// func (a *ApplicationRoutes) getAllApplications(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
// 	// Return list (or map?) of application data
// 	// Need to determine a data format to return
// 	// and come up with a way of getting that data out from models.
// 	// UI expects application_meta + list of versions with useage counts?
// 	// -> can we limit ourselves a bit?
// 	// -> report numbers of versions and numbers of appspaces

// 	//a.AppModel....

// }

// handlePost is for Post with no app-id
// if there are files attached send appfilesmodel(?) for storage,
// ..then ask for files metadata.
// Create DB row for application and return app-id.
func (a *ApplicationRoutes) handlePost(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
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

		_, dsErr = a.AppModel.CreateVersion(app.AppID, appMetadata.AppVersion, locationKey)
		if dsErr != nil {
			fmt.Println(dsErr, dsErr.ExtraMessage())
			dsErr.HTTPError(res)
			return
		}

		// Send back exact same thing we would send if doing a GET on applications.
		respData := createAppResp{
			AppMeta: appMeta{
				AppID:        int(app.AppID),
				AppName:      app.Name,
				Created:      app.Created,
				NumVersion:   1,
				NumAppspaces: 0},
			VersionMeta: versionMeta{
				Version:      string(appMetadata.AppVersion),
				AppID:        int(app.AppID),
				NumAppspaces: 0}}

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
