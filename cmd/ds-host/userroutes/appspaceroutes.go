package userroutes

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

// AppspaceResp is
type AppspaceResp struct {
	AppspaceID     int                  `json:"appspace_id"`
	AppID          int                  `json:"app_id"`
	AppVersion     domain.Version       `json:"app_version"`
	DomainName     string               `json:"domain_name"`
	NoTLS          bool                 `json:"no_tls"`
	PortString     string               `json:"port_string"`
	DropID         string               `json:"dropid"`
	Created        time.Time            `json:"created_dt"`
	Paused         bool                 `json:"paused"`
	UpgradeVersion domain.Version       `json:"upgrade_version,omitempty"`
	AppVersionData *domain.AppVersionUI `json:"ver_data,omitempty"`
}

// AppspaceRoutes handles routes for appspace uploading, creating, deleting.
type AppspaceRoutes struct {
	Config                domain.RuntimeConfig `checkinject:"required"`
	AppspaceUserRoutes    subRoutes            `checkinject:"required"`
	AppspaceExportRoutes  subRoutes            `checkinject:"required"`
	AppspaceRestoreRoutes subRoutes            `checkinject:"required"`
	AppModel              interface {
		GetFromID(domain.AppID) (domain.App, error)
		GetVersion(domain.AppID, domain.Version) (domain.AppVersion, error)
		GetVersionForUI(appID domain.AppID, version domain.Version) (domain.AppVersionUI, error)
	} `checkinject:"required"`
	AppspaceModel interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, error)
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
		GetForApp(appID domain.AppID) ([]*domain.Appspace, error)
	} `checkinject:"required"`
	CreateAppspace interface {
		Create(domain.DropID, domain.AppVersion, string, string) (domain.AppspaceID, domain.JobID, error)
	} `checkinject:"required"`
	PauseAppspace interface {
		Pause(appspaceID domain.AppspaceID, pause bool) error
	} `checkinject:"required"`
	DeleteAppspace interface {
		Delete(domain.Appspace) error
	} `checkinject:"required"`
	AppspaceLogger interface {
		Open(appspaceID domain.AppspaceID) domain.LoggerI
	} `checkinject:"required"`
	SandboxRunsModel interface {
		AppsaceSums(ownerID domain.UserID, appspaceID domain.AppspaceID, from time.Time, to time.Time) (domain.SandboxRunData, error)
	} `checkinject:"required"`
	DropIDModel interface {
		Get(handle string, dom string) (domain.DropID, error)
	} `checkinject:"required"`
	MigrationMinder interface {
		GetForAppspace(domain.Appspace) (domain.AppVersion, bool, error)
	} `checkinject:"required"`
	AppspaceMetaDB interface {
		Create(domain.AppspaceID, int) error
	} `checkinject:"required"`
}

func (a *AppspaceRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", a.getAppspaces) // return app vers for eah
	r.Post("/", a.postNewAppspace)

	r.Route("/{appspace}", func(r chi.Router) {
		r.Use(a.appspaceCtx)
		r.Get("/", a.getAppspace) // reutnr app vers UI
		r.Delete("/", a.deleteAppspace)
		r.Get("/log", a.getLog)
		r.Get("/usage", a.getUsage)
		r.Post("/pause", a.changeAppspacePause)
		r.Mount("/user", a.AppspaceUserRoutes.subRouter())
		r.Mount("/export", a.AppspaceExportRoutes.subRouter())
		r.Mount("/restore", a.AppspaceRestoreRoutes.subRouter())
	})

	return r
}

func (a *AppspaceRoutes) appspaceCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := domain.CtxAuthUserID(r.Context())

		appspaceIDStr := chi.URLParam(r, "appspace")

		appspaceIDInt, err := strconv.Atoi(appspaceIDStr)
		if err != nil {
			returnError(w, err)
			return
		}
		appspaceID := domain.AppspaceID(appspaceIDInt)

		appspace, err := a.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			if err == sql.ErrNoRows {
				returnError(w, errNotFound)
			} else {
				returnError(w, err)
			}
			return
		}
		if appspace.OwnerID != userID {
			returnError(w, errForbidden)
			return
		}

		r = r.WithContext(domain.CtxWithAppspaceData(r.Context(), *appspace))

		next.ServeHTTP(w, r)
	})
}

func (a *AppspaceRoutes) getAppspace(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	respData := a.makeAppspaceMeta(appspace)
	upgrade, ok, err := a.MigrationMinder.GetForAppspace(appspace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		respData.UpgradeVersion = upgrade.Version
	}
	ver, err := a.AppModel.GetVersionForUI(appspace.AppID, appspace.AppVersion)
	if err == nil {
		respData.AppVersionData = &ver
	} else if err != domain.ErrNoRowsInResultSet {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, respData)
}

func (a *AppspaceRoutes) getAppspaces(w http.ResponseWriter, r *http.Request) {
	_, ok := r.URL.Query()["app"]
	if ok {
		a.getAppspacesForApp(w, r)
	} else {
		a.getAppspacesForUser(w, r)
	}
}

func (a *AppspaceRoutes) getAppspacesForApp(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	query := r.URL.Query()
	appIDStr := query["app"]
	appIDInt, err := strconv.Atoi(appIDStr[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	appID := domain.AppID(appIDInt)

	// need to check that app id is owned by owner
	app, err := a.AppModel.GetFromID(appID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if app.OwnerID != userID {
		http.Error(w, "app not owned by user", http.StatusForbidden)
		return
	}
	appspaces, err := a.AppspaceModel.GetForApp(appID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respData := make([]AppspaceResp, len(appspaces))
	for i, appspace := range appspaces {
		respData[i] = a.makeAppspaceMeta(*appspace)
	}
	writeJSON(w, respData)
}

func (a *AppspaceRoutes) getAppspacesForUser(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())
	appspaces, err := a.AppspaceModel.GetForOwner(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respData := make([]AppspaceResp, 0)
	for _, appspace := range appspaces {
		appspaceResp := a.makeAppspaceMeta(*appspace) // TODO
		upgrade, ok, err := a.MigrationMinder.GetForAppspace(*appspace)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			appspaceResp.UpgradeVersion = upgrade.Version
		}
		ver, err := a.AppModel.GetVersionForUI(appspace.AppID, appspace.AppVersion)
		if err == nil {
			appspaceResp.AppVersionData = &ver
		} else if err != domain.ErrNoRowsInResultSet {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respData = append(respData, appspaceResp)
	}
	writeJSON(w, respData)
}

// PostAppspaceReq is sent when creating a new appspace
type PostAppspaceReq struct {
	AppID      domain.AppID   `json:"app_id"`
	Version    domain.Version `json:"app_version"`
	DomainName string         `json:"domain_name"`
	Subdomain  string         `json:"subdomain"`
	DropID     string         `json:"dropid"`
}

// PostAppspaceResp is the return data after creating a new appspace
type PostAppspaceResp struct {
	AppspaceID domain.AppspaceID `json:"appspace_id"`
	JobID      domain.JobID      `json:"job_id"`
}

func (a *AppspaceRoutes) postNewAppspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	reqData := &PostAppspaceReq{}
	err := readJSON(r, reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	app, err := a.AppModel.GetFromID(reqData.AppID)
	if err != nil {
		if err != domain.ErrNoRowsInResultSet {
			http.Error(w, "App not found", http.StatusGone)
		} else {
			http.Error(w, err.Error(), 500)
		}
		return
	}
	if app.OwnerID != userID {
		http.Error(w, "Application not owned by logged in user", http.StatusForbidden)
		return
	}

	version, err := a.AppModel.GetVersion(app.AppID, reqData.Version)
	if err != nil {
		if err == domain.ErrNoRowsInResultSet {
			http.Error(w, "Version not found", http.StatusGone)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	// also need to validate dropid
	err = validator.DropIDFull(reqData.DropID)
	if err != nil {
		returnError(w, err)
		return
	}
	dropIDStr := validator.NormalizeDropIDFull(reqData.DropID)
	dropIDHandle, dropIDDomain := validator.SplitDropID(dropIDStr)
	dropID, err := a.DropIDModel.Get(dropIDHandle, dropIDDomain)
	if err != nil {
		if err == domain.ErrNoRowsInResultSet {
			http.Error(w, "DropID not found", http.StatusGone)
		} else {
			returnError(w, err)
		}
		return
	}
	if dropID.UserID != userID {
		returnError(w, errors.New("DropID user does not match authenticated user"))
		return
	}

	// Here we should check validity of requested domain?

	appspaceID, jobID, err := a.CreateAppspace.Create(dropID, version, reqData.DomainName, reqData.Subdomain)
	if err != nil {
		returnError(w, err)
	}

	resp := PostAppspaceResp{
		AppspaceID: appspaceID,
		JobID:      jobID,
	}

	writeJSON(w, resp)
}

type PostAppspacePauseReq struct {
	Pause bool `json:"pause"`
}

func (a *AppspaceRoutes) changeAppspacePause(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	reqData := PostAppspacePauseReq{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.PauseAppspace.Pause(appspace.AppspaceID, reqData.Pause)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *AppspaceRoutes) deleteAppspace(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	err := a.DeleteAppspace.Delete(appspace)
	if err != nil {
		returnError(w, err)
		return
	}
}

func (a *AppspaceRoutes) getLog(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	logger := a.AppspaceLogger.Open(appspace.AppspaceID)
	if logger == nil {
		writeJSON(w, domain.LogChunk{})
		return
	}
	chunk, err := logger.GetLastBytes(4 * 1024)
	if err != nil {
		returnError(w, err)
		return
	}
	writeJSON(w, chunk)
}

func (a *AppspaceRoutes) getUsage(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	now := time.Now()
	nowMinus30d := now.Add(-30 * 24 * time.Hour)

	sums30d, err := a.SandboxRunsModel.AppsaceSums(appspace.OwnerID, appspace.AppspaceID, nowMinus30d, now)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, sums30d)
}

func (a *AppspaceRoutes) makeAppspaceMeta(appspace domain.Appspace) AppspaceResp {
	return AppspaceResp{
		AppspaceID: int(appspace.AppspaceID),
		AppID:      int(appspace.AppID),
		AppVersion: appspace.AppVersion,
		DomainName: appspace.DomainName,
		NoTLS:      a.Config.ExternalAccess.Scheme == "http",
		PortString: a.Config.Exec.PortString,
		DropID:     appspace.DropID,
		Paused:     appspace.Paused,
		Created:    appspace.Created}
}
