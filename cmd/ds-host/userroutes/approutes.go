package userroutes

import (
	"bytes"
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
	// I think all the routes require auth.
	// ..so run / check auth and bail if not authenticated
	// RouteData should have been augmented with cookie,
	// .but is there a way to check that very quickly?
	if routeData.Cookie == nil || !routeData.Cookie.UserAccount {
		panic("Should not have made it to app routes without a proper cookie")
	}
	// OK, I suupose. But I would still like something cleaner.
	// Can't we just call authenticator again? -> yes but it has to be injected
	// Other option is to add another object to routeData, like *Auth that describes auth state.
	// -> but then I'd rather switch to our new approach of an injectable request data interface

	appID, tail := shiftpath.ShiftPath(routeData.URLTail)
	method := req.Method

	if appID == "" {
		switch method {
		case http.MethodGet:
			// return list of applications for user
			// check query string params for more info on what to return.
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

// handlePost is for Post with no app-id
// if there are files attached send to ds-trusted for storage,
// ..then ask ds-trusted for files metadata.
// Create DB row for application and return app-id.
func (a *ApplicationRoutes) handlePost(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	fileData := a.extractFiles(req)
	if len(*fileData) > 0 {
		locationKey, err := a.AppFilesModel.Save(fileData)
		if err != nil {
			err.HTTPError(res)
			return
		}

		appMeta, err := a.AppFilesModel.ReadMeta(locationKey)
		if err != nil {
			fmt.Println(err, err.ExtraMessage())
			err.HTTPError(res)

			// delete the files? ..it really depends on the error.
			return
		}

		app, dsErr := a.AppModel.Create(routeData.Cookie.UserID, appMeta.AppName)
		if dsErr != nil {
			fmt.Println(err, err.ExtraMessage())
			dsErr.HTTPError(res)
			return
		}

		appVersion, dsErr := a.AppModel.CreateVersion(app.AppID, appMeta.AppVersion, locationKey)
		if dsErr != nil {
			fmt.Println(err, err.ExtraMessage())
			dsErr.HTTPError(res)
			return
		}

		fmt.Println("got response for metadata", appMeta, app, appVersion)

		res.WriteHeader(http.StatusOK)

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
