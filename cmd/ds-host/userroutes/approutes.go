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
	AppModel domain.AppModel
	Logger   domain.LogCLientI
}

// hmm, need to consider all the ways in which applications can be created
// And not corener ourselves with the API.
// -> I think /upload is a bad idea.
// instead, let's consider:
// post to / to create a new application even if only partially,
// ..it gets an entry in DB along with an ID, which is returned with that first request.
// Subsequent updates, finalizing, etc... all reference the id /:id/ and use patch or update.
// ^^ yes.

// OK, but Question: what about versions of applications?
// Do we treat each version as an application, and keep track of them in DB?
// What's the API for uploading / creating a new version of an application?
// -> it seems it should be POST /application/<app-id>/
// OK, but that muddies things a bit.
// ..because the creation of an application might be POST /applicatoin/
// ..but it would contain files that are attached to a version and metadata that is attached to application.

// Further thoughts:
// Any app files that land in ds-trusted for storage must have a json that includes a version, yes?
// And if so ds-trusted can read and validate that and use the version as a subdirectory for app files.
// -> it's only the application name that is ephemeral, and tehrefore should use app-id for that directory.
//    ..app-id being created at any POST request on application/
// -> if the files fail validation, what do you do with them?
//   - delete them immediately?
//   - preserve them for analytical purposes? It seems this should be an option.
// Counterpoint: consider ad-hoc application creation, like where a user can create an application live on the host?
// ..meh?
// ->

// ServeHTTP handles http traffic to the application routes
// Namely upload, create new application, delete, ...
func (a *ApplicationRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// I think all the routes require auth.
	// ..so run / check auth and bail if not authenticated

	appID, tail := shiftpath.ShiftPath(routeData.URLTail)
	method := req.Method

	if appID == "" {
		switch method {
		case http.MethodGet:
			// return list of applications for user
			// check query string params for more info on what to return.
			http.Error(res, "get /application", http.StatusNotImplemented)
		case http.MethodPost:
			// this is where it gets juicy.
			// You have to create a DB row, and figure out how to populate data
			// ..inlucding potentially shipping files over to ds-trusted.
			// ..which will look at them and validate and extract metadata, like name and version.
			// Then the ds-host will have to populate the version data for the application.
			// Then subsequent patches are either for application, or for version specifically (are there any?)
			// If this is a file upload, then you must also create a directory key and send that to ds-trusted.

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

// func reqContainsFiles(req *http.Request) {
// 	err := r.ParseMultipartForm(100000)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// }
// uhrf. Turning uploads into usable data is pretty heavy in Go.
// Takes up a big block of memory, writes to disk if block exceeded in size.

func (a *ApplicationRoutes) post(req *http.Request, routeData *domain.AppspaceRouteData) {
	// You can assume auth

	// copied from http://sanatgersappa.blogspot.com/2013/03/handling-multiple-file-uploads-in-go.html
	//----- Buffered version:

	// err := req.ParseMultipartForm(100000)
	// if err != nil {
	// 	//http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	fmt.Println("Error in parseMultipart ", err.Error())
	// 	return
	// }

	// //get a ref to the parsed multipart form
	// m := req.MultipartForm

	// fmt.Println("parsed mpf", m)

	// //get the *fileheaders
	// files := m.File["app_dir"]
	// for i := range files {
	// 	//for each fileheader, get a handle to the actual file
	// 	fileHeader := files[i]
	// 	fmt.Println("Fileheader Size:", fileHeader.Filename, fileHeader.Size)

	// 	file, err := files[i].Open()
	// 	defer file.Close()
	// 	if err != nil {
	// 		fmt.Println("Error in files[i].Open ", err.Error())
	// 		return
	// 	}

	// 	buf := &bytes.Buffer{}
	// 	buf.ReadFrom(file)
	// 	str := buf.String()

	// 	fmt.Println("as string:", str)

	// 	// //create destination file making sure the path is writeable.
	// 	// dst, err := os.Create("/home/developer/test-uploads/" + files[i].Filename)
	// 	// defer dst.Close()
	// 	// if err != nil {
	// 	// 	fmt.Println("Error in os Create ", err.Error())
	// 	// 	return
	// 	// }
	// 	// //copy the uploaded file to the destination file
	// 	// if _, err := io.Copy(dst, file); err != nil {
	// 	// 	fmt.Println("Error in files[i].Open ", err.Error())
	// 	// 	return
	// 	// }

	// }

	// ---------- streaming version
	reader, err := req.MultipartReader()
	if err != nil {
		fmt.Println("request may not be of type multipart form data")
		panic(err)
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

		fmt.Println("part:", part.FileName()) //does that mean we don't know the size?

		buf := &bytes.Buffer{}
		buf.ReadFrom(part) //maybe limit bytes to read to avert file bomb.
		str := buf.String()

		fmt.Println("as string:", str)
	}

}
