package userroutes

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type SelectBackup struct {
	BackupFilename string `json:"backup_file"`
}

type RestoreError struct {
	// MissingFiles is top-level files missing from the zip file we are attempting to restore
	MissingFiles []string `json:"missing_files"`
	// ZipFiles is the set of files in the top level of the zip
	// provided for convenience to the user so they may find their mistake
	ZipFiles []string `json:"zip_files"`
}

//  RestoreData is the response to the selection / upload of an appspace archive
type RestoreData struct {
	Token  string        `json:"token"`
	Error  *RestoreError `json:"err"`
	Schema int           `json:"schema"`
	// more stuff...
}

type AppspaceRestoreRoutes struct {
	RestoreAppspace interface {
		Prepare(reader io.Reader) (string, error)
		PrepareBackup(appspaceID domain.AppspaceID, backupFile string) (string, error)
		CheckAppspaceDataValid(tok string) error
		GetMetaInfo(tok string) (domain.AppspaceMetaInfo, error)
		ReplaceData(tok string, appspaceID domain.AppspaceID) error
	} `checkinject:"required"`
}

// There are basically two routes:
// post /
// post /<token>

func (e *AppspaceRestoreRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Post("/", e.useBackup)
	r.Post("/upload", e.upload)
	r.Post("/{token}", e.commit)

	return r
}

func (e *AppspaceRestoreRoutes) useBackup(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	reqData := &SelectBackup{}
	err := readJSON(r, reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.AppspaceBackupFile(reqData.BackupFilename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tok, err := e.RestoreAppspace.PrepareBackup(appspace.AppspaceID, reqData.BackupFilename)
	if err != nil {
		handleError(w, err)
		return
	}

	resp := RestoreData{
		Token: tok}

	err = e.RestoreAppspace.CheckAppspaceDataValid(tok)
	if err != nil {
		if badZipErr, ok := err.(domain.BadRestoreZip); ok {
			resp.Error = &RestoreError{
				MissingFiles: badZipErr.MissingFiles(),
				ZipFiles:     badZipErr.ZipFiles()}
			writeJSON(w, resp)
		} else {
			handleError(w, err)
		}
		return
	}

	info, err := e.RestoreAppspace.GetMetaInfo(tok)
	if err != nil {
		handleError(w, err)
		return
	}

	resp.Schema = info.Schema

	writeJSON(w, resp)
}

func (e *AppspaceRestoreRoutes) upload(w http.ResponseWriter, r *http.Request) {
	f, _, err := r.FormFile("zip")
	if err != nil {
		http.Error(w, "unable to get zip file from multipart: "+err.Error(), http.StatusBadRequest)
		return
	}

	tok, err := e.RestoreAppspace.Prepare(f)
	if err != nil {
		handleError(w, err)
		return
	}

	resp := RestoreData{
		Token: tok}

	// Note that checking this here an bailing if incorrect is not right?
	// This needs to be versioned according to ds-api
	// And we need to try and get ds-api of zip, then process the full zip according to that API.
	err = e.RestoreAppspace.CheckAppspaceDataValid(tok)
	if err != nil {
		if badZipErr, ok := err.(domain.BadRestoreZip); ok {
			resp.Error = &RestoreError{
				MissingFiles: badZipErr.MissingFiles(),
				ZipFiles:     badZipErr.ZipFiles()}
			writeJSON(w, resp)
		} else {
			handleError(w, err)
		}
		return
	}

	info, err := e.RestoreAppspace.GetMetaInfo(tok)
	if err != nil {
		handleError(w, err)
		return
	}

	resp.Schema = info.Schema

	writeJSON(w, resp)
}

func (e *AppspaceRestoreRoutes) commit(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	token := chi.URLParam(r, "token")

	err := e.RestoreAppspace.ReplaceData(token, appspace.AppspaceID)
	if err != nil {
		returnError(w, err)
		return
	}
}

func handleError(w http.ResponseWriter, err error) {
	if strings.HasPrefix(err.Error(), "bad input:") {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		returnError(w, err)
	}
}
