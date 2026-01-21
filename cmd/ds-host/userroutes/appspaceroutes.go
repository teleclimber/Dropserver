package userroutes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

type AppspaceUserBase struct {
	ProxyID     domain.ProxyID `json:"proxy_id"`
	DisplayName string         `json:"display_name"`
	Avatar      string         `json:"avatar"`
}

// AppspaceResp
type AppspaceResp struct {
	AppspaceID       int                            `json:"appspace_id"`
	OwnerID          domain.UserID                  `json:"owner_id"` // maybe also add OwnerProxyID so taht they can be identified in list of users.
	AppID            int                            `json:"app_id"`
	AppVersion       domain.Version                 `json:"app_version"`
	DomainName       string                         `json:"domain_name"`
	NoTLS            bool                           `json:"no_tls"`
	PortString       string                         `json:"port_string"`
	Created          time.Time                      `json:"created_dt"`
	Paused           bool                           `json:"paused"`
	Status           domain.AppspaceStatusEvent     `json:"status"`
	TSNetStatus      domain.TSNetAppspaceStatus     `json:"tsnet_status"`
	UpgradeVersion   domain.Version                 `json:"upgrade_version,omitempty"`
	AppVersionData   *domain.AppVersionUI           `json:"ver_data,omitempty"`
	TSNetData        *domain.AppspaceTSNet          `json:"tsnet_data,omitempty"`
	Users            []AppspaceUserBase             `json:"users"`
	AppspaceAuthUser *domain.UserIDProxyIDConflicts `json:"auth_user_id_conflicts"`
}

// AppspaceRoutes handles routes for appspace uploading, creating, deleting.
type AppspaceRoutes struct {
	Config                domain.RuntimeConfig `checkinject:"required"`
	AppspaceLocation2Path interface {
		Avatar(string, string) string
	} `checkinject:"required"`
	AppspaceUserRoutes    subRoutes `checkinject:"required"`
	AppspaceExportRoutes  subRoutes `checkinject:"required"`
	AppspaceRestoreRoutes subRoutes `checkinject:"required"`
	AppModel              interface {
		GetFromID(domain.AppID) (domain.App, error)
		GetVersion(domain.AppID, domain.Version) (domain.AppVersion, error)
		GetVersionForUI(appID domain.AppID, version domain.Version) (domain.AppVersionUI, error)
	} `checkinject:"required"`
	AppFilesModel interface {
		GetLinkPath(string, string) string
	} `checkinject:"required"`
	AppspaceModel interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, error)
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
		GetForApp(appID domain.AppID) ([]*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceTSNetModel interface {
		Get(domain.AppspaceID) (domain.AppspaceTSNet, error)
		CreateOrUpdate(appspaceID domain.AppspaceID, backendURL string, hostname string, connect bool) error
		SetConnect(domain.AppspaceID, bool) error
		Delete(domain.AppspaceID) error
	} `checkinject:"required"`
	AppspaceStatus interface {
		Get(domain.AppspaceID) domain.AppspaceStatusEvent
	} `checkinject:"required"`
	AppspaceTSNet interface {
		Create(domain.AppspaceID, domain.TSNetCreateConfig) error
		Connect(domain.AppspaceID) error
		Disconnect(domain.AppspaceID)
		Delete(domain.AppspaceID) error
		GetStatus(domain.AppspaceID) domain.TSNetAppspaceStatus
		GetPeerUsers(domain.AppspaceID) []domain.TSNetPeerUser
	} `checkinject:"required"`
	AppspaceUserModel interface {
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	} `checkinject:"required"`
	ManageUsers interface {
		GetProxyIDForUserID(appspaceID domain.AppspaceID, userID domain.UserID) (domain.ProxyID, error)
		GetConflictsForUserID(appspaceID domain.AppspaceID, userID domain.UserID) (domain.UserIDProxyIDConflicts, error)
		AppspacesForUser(domain.UserID) (map[domain.AppspaceID]domain.UserIDProxyIDConflicts, error)
	} `checkinject:"required"`
	CreateAppspace interface {
		Create(domain.UserID, domain.AppVersion, string, string) (domain.AppspaceID, domain.JobID, error)
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
		AppspaceSums(ownerID domain.UserID, appspaceID domain.AppspaceID, from time.Time, to time.Time) (domain.SandboxRunData, error)
	} `checkinject:"required"`
	DropIDModel interface {
		Get(handle string, dom string) (domain.DropID, error)
	} `checkinject:"required"`
	MigrationMinder interface {
		GetForAppspace(domain.Appspace) (domain.Version, bool, error)
	} `checkinject:"required"`
}

func (a *AppspaceRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", a.getAppspaces) // return app vers for eah
	r.Post("/", a.postNewAppspace)

	r.Route("/{appspace}", func(r chi.Router) {
		r.Use(a.appspaceCtx)

		r.Group(func(r chi.Router) {
			r.Use(a.userIsAppspaceUserOrOwner)
			r.Get("/", a.getAppspace)
			r.Get("/app-icon", a.getAppIcon)
			r.Get("/avatar/{filename}", a.getUserAvatar)
		})

		r.Group(func(r chi.Router) {
			r.Use(a.userIsAppspaceOwner)
			r.Delete("/", a.deleteAppspace)
			r.Get("/log", a.getLog)
			r.Get("/usage", a.getUsage)
			r.Post("/pause", a.changeAppspacePause)
			r.Get("/tsnet/peerusers", a.getTSNetPeerUsers)
			r.Post("/tsnet/connect", a.connectTSNet)
			r.Post("/tsnet", a.createTSNet)
			r.Delete("/tsnet", a.deleteTSNet)
			r.Mount("/user", a.AppspaceUserRoutes.subRouter())
			r.Mount("/export", a.AppspaceExportRoutes.subRouter())
			r.Mount("/restore", a.AppspaceRestoreRoutes.subRouter())
		})
	})

	return r
}

func (a *AppspaceRoutes) appspaceCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appspaceIDStr := chi.URLParam(r, "appspace")
		appspaceIDInt, err := strconv.Atoi(appspaceIDStr)
		if err != nil {
			returnError(w, err)
			return
		}
		appspaceID := domain.AppspaceID(appspaceIDInt)
		appspace, err := a.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			if err == domain.ErrNoRowsInResultSet {
				returnError(w, errNotFound)
			} else {
				returnError(w, err)
			}
			return
		}
		r = r.WithContext(domain.CtxWithAppspaceData(r.Context(), *appspace))
		next.ServeHTTP(w, r)
	})
}

func (a *AppspaceRoutes) userIsAppspaceUserOrOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := domain.CtxAuthUserID(r.Context())
		appspace, _ := domain.CtxAppspaceData(r.Context())
		if appspace.OwnerID == userID {
			next.ServeHTTP(w, r)
			return
		}
		_, err := a.ManageUsers.GetConflictsForUserID(appspace.AppspaceID, userID)
		if err == domain.ErrNoRowsInResultSet {
			returnError(w, errForbidden)
			return
		}
		if err != nil {
			returnInternalError(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *AppspaceRoutes) userIsAppspaceOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := domain.CtxAuthUserID(r.Context())
		appspace, _ := domain.CtxAppspaceData(r.Context())
		if appspace.OwnerID != userID {
			returnError(w, errForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *AppspaceRoutes) getAppIcon(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	appVersion, err := a.AppModel.GetVersion(appspace.AppID, appspace.AppVersion)
	if err != nil {
		returnInternalError(w)
		return
	}
	p := a.AppFilesModel.GetLinkPath(appVersion.LocationKey, "app-icon")
	if p == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, p)
}

func (a *AppspaceRoutes) getUserAvatar(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	avatarFilename := chi.URLParam(r, "filename")

	err := validator.AppspaceAvatarFilename(avatarFilename)
	if err != nil {
		returnError(w, err)
		return
	}

	http.ServeFile(w, r, a.AppspaceLocation2Path.Avatar(appspace.LocationKey, avatarFilename))
}

func (a *AppspaceRoutes) getAppspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())
	appspace, _ := domain.CtxAppspaceData(r.Context())
	respData := a.makeAppspaceMeta(appspace)

	if appspace.AppVersion == domain.Version("") { // at appspace creation time, AppVersion can be empty.
		writeJSON(w, respData)
		return
	}

	respData, err := a.makeAppspaceResp(appspace)
	if err != nil {
		returnInternalError(w)
		return
	}

	userConflicts, err := a.ManageUsers.GetConflictsForUserID(appspace.AppspaceID, userID)
	if err != nil && err != domain.ErrNoRowsInResultSet {
		returnInternalError(w)
		return
	}
	respData.AppspaceAuthUser = &userConflicts

	// getAppspace returns upgrade version getAppspaces does not.
	upgradeVersion, _, err := a.MigrationMinder.GetForAppspace(appspace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respData.UpgradeVersion = upgradeVersion

	writeJSON(w, respData)
}

// getAppspaces returns all appspaces relevant to the user.
// Union of owned appspaces and appspaces they have access to,
// even if there are conflicts with auth identifiers
func (a *AppspaceRoutes) getAppspaces(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	respData := make([]AppspaceResp, 0)

	appspaceUserConflicts, err := a.ManageUsers.AppspacesForUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for appspaceID, uc := range appspaceUserConflicts {
		appspace, err := a.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			returnInternalError(w)
			return
		}
		appspaceResp, err := a.makeAppspaceResp(*appspace)
		if err != nil {
			returnInternalError(w)
			return
		}
		appspaceResp.AppspaceAuthUser = &uc
		respData = append(respData, appspaceResp)
	}

	// Include appspaces for which the owner is not a user:
	appspaces, err := a.AppspaceModel.GetForOwner(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, appspace := range appspaces {
		if _, ok := appspaceUserConflicts[appspace.AppspaceID]; ok {
			// skip if we already got this appspace above
			continue
		}
		appspaceResp, err := a.makeAppspaceResp(*appspace)
		if err != nil {
			returnInternalError(w)
			return
		}
		respData = append(respData, appspaceResp)
	}
	writeJSON(w, respData)
}

func (a *AppspaceRoutes) makeAppspaceResp(appspace domain.Appspace) (AppspaceResp, error) {
	appspaceResp := a.makeAppspaceMeta(appspace)
	appspaceResp.Status = a.AppspaceStatus.Get(appspace.AppspaceID)
	tsnetData, err := a.AppspaceTSNetModel.Get(appspace.AppspaceID)
	if err == nil {
		appspaceResp.TSNetData = &tsnetData
	}
	appspaceResp.TSNetStatus = a.AppspaceTSNet.GetStatus(appspace.AppspaceID)
	ver, err := a.AppModel.GetVersionForUI(appspace.AppID, appspace.AppVersion)
	if err == nil {
		appspaceResp.AppVersionData = &ver
	} else if err != domain.ErrNoRowsInResultSet {
		return appspaceResp, err
	}

	users, err := a.AppspaceUserModel.GetAll(appspace.AppspaceID)
	if err != nil {
		return appspaceResp, err
	}
	appspaceResp.Users = make([]AppspaceUserBase, len(users))
	for i, u := range users {
		appspaceResp.Users[i] = AppspaceUserBase{
			ProxyID:     u.ProxyID,
			DisplayName: u.DisplayName,
			Avatar:      u.Avatar}
	}
	return appspaceResp, nil
}

// PostAppspaceReq is sent when creating a new appspace
type PostAppspaceReq struct {
	AppID      domain.AppID   `json:"app_id"`
	Version    domain.Version `json:"app_version"`
	DomainName string         `json:"domain_name"`
	Subdomain  string         `json:"subdomain"`
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

	// Here we should check validity of requested domain?

	appspaceID, jobID, err := a.CreateAppspace.Create(userID, version, reqData.DomainName, reqData.Subdomain)
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

func (a *AppspaceRoutes) getTSNetPeerUsers(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	users := a.AppspaceTSNet.GetPeerUsers(appspace.AppspaceID)

	writeJSON(w, users)
}

func (a *AppspaceRoutes) createTSNet(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	reqData := domain.TSNetCreateConfig{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.TSNetCreateConfig(reqData)
	if err != nil {
		a.getLogger("createTSNet validator").Debug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.AppspaceTSNetModel.CreateOrUpdate(appspace.AppspaceID, reqData.ControlURL, reqData.Hostname, true)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = a.AppspaceTSNet.Create(appspace.AppspaceID, reqData)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type ConnectReq struct {
	Connect bool `json:"connect"`
}

func (a *AppspaceRoutes) connectTSNet(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	reqData := ConnectReq{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.AppspaceTSNetModel.SetConnect(appspace.AppspaceID, reqData.Connect)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	if reqData.Connect {
		err = a.AppspaceTSNet.Connect(appspace.AppspaceID)
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	} else {
		a.AppspaceTSNet.Disconnect(appspace.AppspaceID)
		w.WriteHeader(http.StatusOK)
	}
}

func (a *AppspaceRoutes) deleteTSNet(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	err := a.AppspaceTSNetModel.Delete(appspace.AppspaceID)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	a.AppspaceTSNet.Delete(appspace.AppspaceID)

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

	sums30d, err := a.SandboxRunsModel.AppspaceSums(appspace.OwnerID, appspace.AppspaceID, nowMinus30d, now)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, sums30d)
}

func (a *AppspaceRoutes) makeAppspaceMeta(appspace domain.Appspace) AppspaceResp {
	return AppspaceResp{
		AppspaceID: int(appspace.AppspaceID),
		OwnerID:    appspace.OwnerID,
		AppID:      int(appspace.AppID),
		AppVersion: appspace.AppVersion,
		DomainName: appspace.DomainName,
		NoTLS:      a.Config.ExternalAccess.Scheme == "http",
		PortString: a.Config.Exec.PortString,
		Paused:     appspace.Paused,
		Created:    appspace.Created}
}

func (a *AppspaceRoutes) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("AppspaceRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
