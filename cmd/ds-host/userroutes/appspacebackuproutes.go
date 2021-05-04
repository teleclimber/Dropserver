package userroutes

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/validator"
)

type BackupFile struct {
	Filename string `json:"filename"`
}

type AppspaceBackupRoutes struct {
	Config             *domain.RuntimeConfig
	AppspaceFilesModel interface {
		GetBackups(locationKey string) ([]string, error)
		DeleteBackup(locationKey string, filename string) error
	}
	BackupAppspace interface {
		CreateBackup(appspaceID domain.AppspaceID) (string, error)
	}
}

// not 100% sure what the api is here.
// - GET / :get list of archives
// - POST / :trigger creation of export archive
// - GET /<archive> :download an archive
// - DELETE /<archive> :delete that one
// - [Set automatic backup schedule and backups rotation? (1/day at 2am keep last 7)] //maybe another route altogether

func (e *AppspaceBackupRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if _, ok := ctxAuthUserID(ctx); !ok {
		e.getLogger("ServeHTTP").Error(errors.New("no authorized user incontext"))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	archive, _ := shiftpath.ShiftPath(ctxURLTail(ctx))

	if archive == "" {
		switch req.Method {
		case http.MethodGet:
			e.getArchives(res, req)
		case http.MethodPost:
			e.createArchive(res, req)
		default:
			http.Error(res, "bad method for export route", http.StatusBadRequest)
		}
	} else {
		switch req.Method {
		case http.MethodGet:
			e.downloadArchive(res, req, archive)
		case http.MethodDelete:
			e.deleteArchive(res, req, archive)
		default:
			http.Error(res, "bad method for export archive route", http.StatusBadRequest)
		}
	}
}

// These function signatures are getting ridiculous
// every time you add some context, you add an argument
func (e *AppspaceBackupRoutes) getArchives(res http.ResponseWriter, req *http.Request) {
	appspace, ok := ctxAppspaceData(req.Context())
	if !ok {
		e.getLogger("getArchives").Error(errors.New("no appspace data in context"))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	backups, err := e.AppspaceFilesModel.GetBackups(appspace.LocationKey)
	if err != nil {
		returnError(res, err)
		return
	}

	writeJSON(res, backups)
}

func (e *AppspaceBackupRoutes) createArchive(res http.ResponseWriter, req *http.Request) {
	appspace, ok := ctxAppspaceData(req.Context())
	if !ok {
		e.getLogger("getArchives").Error(errors.New("no appspace data in context"))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	backupFile, err := e.BackupAppspace.CreateBackup(appspace.AppspaceID)
	if err != nil {
		returnError(res, err)
		return
	}

	writeJSON(res, BackupFile{Filename: path.Base(backupFile)})
}

func (e *AppspaceBackupRoutes) downloadArchive(res http.ResponseWriter, req *http.Request, archive string) {
	// check it exists
	appspace, ok := ctxAppspaceData(req.Context())
	if !ok {
		e.getLogger("downloadArchive").Error(errors.New("no appspace data in context"))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	err := validator.AppspaceBackupFile(archive)
	if err != nil {
		returnError(res, err)
		return
	}

	fullPath := filepath.Join(e.Config.Exec.AppspacesPath, appspace.LocationKey, "backups", archive)

	splitDomain := strings.SplitN(appspace.DomainName, ".", 2)
	downloadFileName := splitDomain[0] + "-" + archive

	res.Header().Set("Content-Type", "application/zip")
	res.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", downloadFileName))

	http.ServeFile(res, req, fullPath)
}

func (e *AppspaceBackupRoutes) deleteArchive(res http.ResponseWriter, req *http.Request, archive string) {
	appspace, ok := ctxAppspaceData(req.Context())
	if !ok {
		e.getLogger("deleteArchive").Error(errors.New("no appspace data in context"))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	err := validator.AppspaceBackupFile(archive)
	if err != nil {
		returnError(res, err)
		return
	}

	err = e.AppspaceFilesModel.DeleteBackup(appspace.LocationKey, archive)
	if err != nil {
		returnError(res, err)
		return
	}
}

func (e *AppspaceBackupRoutes) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceBackupRoutes")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
