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

//AppspaceMeta is
type AppspaceMeta struct {
	AppspaceID int            `json:"appspace_id"`
	AppID      int            `json:"app_id"`
	AppVersion domain.Version `json:"app_version"`
	DomainName string         `json:"domain_name"`
	NoSSL      bool           `json:"no_ssl"`
	PortString string         `json:"port_string"`
	DropID     string         `json:"dropid"`
	Created    time.Time      `json:"created_dt"`
	Paused     bool           `json:"paused"`
	Upgrade    *VersionMeta   `json:"upgrade,omitempty"`
}

// AppspaceRoutes handles routes for appspace uploading, creating, deleting.
type AppspaceRoutes struct {
	Config               domain.RuntimeConfig `checkinject:"required"`
	AppspaceUserRoutes   subRoutes            `checkinject:"required"`
	AppspaceExportRoutes subRoutes            `checkinject:"required"`
	AppModel             interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
	} `checkinject:"required"`
	AppspaceFilesModel interface {
		CreateLocation() (string, error)
	} `checkinject:"required"`
	AppspaceModel interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, error)
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
		GetForApp(appID domain.AppID) ([]*domain.Appspace, error)
		Create(domain.Appspace) (*domain.Appspace, error)
		Pause(domain.AppspaceID, bool) error
		GetFromDomain(string) (*domain.Appspace, error)
	} `checkinject:"required"`
	DeleteAppspace interface {
		Delete(domain.Appspace) error
	} `checkinject:"required"`
	AppspaceUserModel interface {
		Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
	} `checkinject:"required"`
	DomainController interface {
		CheckAppspaceDomain(userID domain.UserID, dom string, subdomain string) (domain.DomainCheckResult, error)
	} `checkinject:"required"`
	DropIDModel interface {
		Get(handle string, dom string) (domain.DropID, error)
	} `checkinject:"required"`
	MigrationMinder interface {
		GetForAppspace(domain.Appspace) (domain.AppVersion, bool, error)
	} `checkinject:"required"`
	AppspaceMetaDB    domain.AppspaceMetaDB `checkinject:"required"`
	MigrationJobModel interface {
		Create(domain.UserID, domain.AppspaceID, domain.Version, bool) (*domain.MigrationJob, error)
	} `checkinject:"required"`
	MigrationJobController interface {
		WakeUp()
	} `checkinject:"required"`
}

func (a *AppspaceRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", a.getAppspaces)
	r.Post("/", a.postNewAppspace)

	r.Route("/{appspace}", func(r chi.Router) {
		r.Use(a.appspaceCtx)
		r.Get("/", a.getAppspace)
		r.Delete("/", a.deleteAppspace)
		r.Post("/pause", a.changeAppspacePause)
		r.Mount("/user", a.AppspaceUserRoutes.subRouter())
		r.Mount("/export", a.AppspaceExportRoutes.subRouter())
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
		upgradeMeta := makeVersionMeta(upgrade)
		respData.Upgrade = &upgradeMeta
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
	respData := make([]AppspaceMeta, len(appspaces))
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
	respData := make([]AppspaceMeta, 0)
	for _, appspace := range appspaces {
		appspaceMeta := a.makeAppspaceMeta(*appspace)
		upgrade, ok, err := a.MigrationMinder.GetForAppspace(*appspace)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			upgradeMeta := makeVersionMeta(upgrade)
			appspaceMeta.Upgrade = &upgradeMeta
		}
		respData = append(respData, appspaceMeta)
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

//PostAppspaceResp is the return data after creating a new appspace
type PostAppspaceResp struct {
	AppspaceID domain.AppspaceID `json:"appspace_id"`
}

func (a *AppspaceRoutes) postNewAppspace(w http.ResponseWriter, r *http.Request) {
	// This whole process should be in an appspace ops function, not in the route handler.
	userID, _ := domain.CtxAuthUserID(r.Context())

	reqData := &PostAppspaceReq{}
	err := readJSON(r, reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	app, err := a.AppModel.GetFromID(reqData.AppID)
	if err != nil {
		if err != sql.ErrNoRows {
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
		if err == sql.ErrNoRows {
			http.Error(w, "Version not found", http.StatusGone)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	// Here it would be nice if CheckAppspaceDomain also reserved that name temporarily
	check, err := a.DomainController.CheckAppspaceDomain(userID, reqData.DomainName, reqData.Subdomain)
	if err != nil {
		returnError(w, err)
		return
	}
	if !check.Valid || !check.Available {
		http.Error(w, "domain or subdomain no longer valid or available", http.StatusGone)
		return
	}

	fullDomain := reqData.DomainName
	if reqData.Subdomain != "" {
		fullDomain = reqData.Subdomain + "." + reqData.DomainName
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
		if err == sql.ErrNoRows {
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

	locationKey, err := a.AppspaceFilesModel.CreateLocation()
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	inAppspace := domain.Appspace{
		OwnerID:     userID,
		AppID:       app.AppID,
		AppVersion:  version.Version,
		DomainName:  fullDomain,
		DropID:      reqData.DropID,
		LocationKey: locationKey,
	}

	appspace, err := a.AppspaceModel.Create(inAppspace)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = a.AppspaceMetaDB.Create(appspace.AppspaceID, 0) // 0 is the ds api version
	if err != nil {
		http.Error(w, "Failed to create appspace meta db", http.StatusInternalServerError)
	}

	// Create owner user
	_, err = a.AppspaceUserModel.Create(appspace.AppspaceID, "dropid", dropIDStr)
	if err != nil {
		returnError(w, err)
		return
	}
	// TODO use whatver process that sets values of display name and avatar to set those for owner user

	// migrate to whatever version was selected
	_, err = a.MigrationJobModel.Create(userID, appspace.AppspaceID, version.Version, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a.MigrationJobController.WakeUp()

	resp := PostAppspaceResp{
		AppspaceID: appspace.AppspaceID}

	writeJSON(w, resp)
}

func (a *AppspaceRoutes) changeAppspacePause(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	reqData := PostAppspacePauseReq{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.AppspaceModel.Pause(appspace.AppspaceID, reqData.Pause)
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

func (a *AppspaceRoutes) makeAppspaceMeta(appspace domain.Appspace) AppspaceMeta {
	return AppspaceMeta{
		AppspaceID: int(appspace.AppspaceID),
		AppID:      int(appspace.AppID),
		AppVersion: appspace.AppVersion,
		DomainName: appspace.DomainName,
		NoSSL:      a.Config.Server.NoSsl,
		PortString: a.Config.PortString,
		DropID:     appspace.DropID,
		Paused:     appspace.Paused,
		Created:    appspace.Created}
}
