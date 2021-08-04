package userroutes

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

type BackupFile struct {
	Filename string `json:"filename"`
}

type AppspaceBackupRoutes struct {
	Config             *domain.RuntimeConfig `checkinject:"required"`
	AppspaceFilesModel interface {
		GetBackups(locationKey string) ([]string, error)
		DeleteBackup(locationKey string, filename string) error
	} `checkinject:"required"`
	BackupAppspace interface {
		CreateBackup(appspaceID domain.AppspaceID) (string, error)
	} `checkinject:"required"`
}

// not 100% sure what the api is here.
// - GET / :get list of archives
// - POST / :trigger creation of export archive
// - GET /<archive> :download an archive
// - DELETE /<archive> :delete that one
// - [Set automatic backup schedule and backups rotation? (1/day at 2am keep last 7)] //maybe another route altogether

func (e *AppspaceBackupRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", e.getArchives)
	r.Post("/", e.createArchive)

	r.Route("/{archive}", func(r chi.Router) {
		r.Get("/", e.downloadArchive)
		r.Delete("/", e.deleteArchive)
	})

	return r
}

// These function signatures are getting ridiculous
// every time you add some context, you add an argument
func (e *AppspaceBackupRoutes) getArchives(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	backups, err := e.AppspaceFilesModel.GetBackups(appspace.LocationKey)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, backups)
}

func (e *AppspaceBackupRoutes) createArchive(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	backupFile, err := e.BackupAppspace.CreateBackup(appspace.AppspaceID)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, BackupFile{Filename: backupFile})
}

func (e *AppspaceBackupRoutes) downloadArchive(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	archive, err := getArchiveFromPath(r)
	if err != nil {
		returnError(w, err)
		return
	}

	fullPath := filepath.Join(e.Config.Exec.AppspacesPath, appspace.LocationKey, "backups", archive)

	splitDomain := strings.SplitN(appspace.DomainName, ".", 2)
	downloadFileName := splitDomain[0] + "-" + archive

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", downloadFileName))

	http.ServeFile(w, r, fullPath)
}

func (e *AppspaceBackupRoutes) deleteArchive(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	archive, err := getArchiveFromPath(r)
	if err != nil {
		returnError(w, err)
		return
	}

	err = e.AppspaceFilesModel.DeleteBackup(appspace.LocationKey, archive)
	if err != nil {
		returnError(w, err)
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

func getArchiveFromPath(r *http.Request) (string, error) {
	archive := chi.URLParam(r, "archive")

	err := validator.AppspaceBackupFile(archive)
	if err != nil {
		return "", err
	}

	return archive, nil
}
