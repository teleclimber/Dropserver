package userroutes

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
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
	Config             domain.RuntimeConfig
	AppspaceUserRoutes interface {
		ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace)
	}
	AppspaceExportRoutes http.Handler
	AppModel             interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
	}
	AppspaceFilesModel interface {
		CreateLocation() (string, error)
	}
	AppspaceModel interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, error)
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
		Create(domain.Appspace) (*domain.Appspace, error)
		Pause(domain.AppspaceID, bool) error
		GetFromDomain(string) (*domain.Appspace, error)
		Delete(domain.AppspaceID) error
	}
	AppspaceUserModel interface {
		Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
		UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, permissions []string) error
	}
	DomainController interface {
		CheckAppspaceDomain(userID domain.UserID, dom string, subdomain string) (domain.DomainCheckResult, error)
	}
	DropIDModel interface {
		Get(handle string, dom string) (domain.DropID, error)
	}
	MigrationMinder interface {
		GetForAppspace(domain.Appspace) (domain.AppVersion, bool, error)
	}
	AppspaceMetaDB    domain.AppspaceMetaDB
	MigrationJobModel interface {
		Create(domain.UserID, domain.AppspaceID, domain.Version, bool) (*domain.MigrationJob, error)
	}
	MigrationJobController interface {
		WakeUp()
	}
}

// ServeHTTP handles http traffic to the appspace routes
// Namely create, delete, set version, etc...
func (a *AppspaceRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	ctx := req.Context()

	if routeData.Authentication == nil || !routeData.Authentication.UserAccount {
		// maybe log it? Frankly this should be a panic.
		// It's programmer error pure and simple. Kill this thing.
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we dun fogged up
		return
	}

	appspace, err := a.getAppspaceFromPath(routeData)
	if err != nil {
		returnError(res, err)
		return
	}

	if appspace == nil {
		switch req.Method {
		case http.MethodGet:
			a.getAllAppspaces(res, req, routeData)
		case http.MethodPost:
			a.postNewAppspace(res, req, routeData)
		default:
			http.Error(res, "bad method for /appspace", http.StatusBadRequest)
		}
	} else {
		head, tail := shiftpath.ShiftPath(routeData.URLTail)
		routeData.URLTail = tail
		ctx = ctxWithURLTail(ctx, tail)
		ctx = ctxWithAppspaceData(ctx, *appspace)
		req = req.WithContext(ctx)

		switch head {
		case "":
			switch req.Method {
			case http.MethodGet:
				a.getAppspace(res, req, routeData, appspace)
			case http.MethodDelete:
				a.deleteAppspace(res, req, routeData, appspace)
			default:
				http.Error(res, "bad method for /appspace/appspace-id", http.StatusBadRequest)
			}
		case "pause":
			a.changeAppspacePause(res, req, routeData, appspace)
		case "user":
			a.AppspaceUserRoutes.ServeHTTP(res, req, routeData, appspace)
		case "export":
			a.AppspaceExportRoutes.ServeHTTP(res, req)
		default:
			http.Error(res, "", http.StatusNotImplemented)
		}
	}
}

func (a *AppspaceRoutes) getAppspace(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	respData := a.makeAppspaceMeta(*appspace)
	upgrade, ok, err := a.MigrationMinder.GetForAppspace(*appspace)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		upgradeMeta := makeVersionMeta(upgrade)
		respData.Upgrade = &upgradeMeta
	}

	writeJSON(res, respData)
}

func (a *AppspaceRoutes) getAllAppspaces(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	appspaces, err := a.AppspaceModel.GetForOwner(routeData.Authentication.UserID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	respData := make([]AppspaceMeta, 0)

	for _, appspace := range appspaces {
		appspaceMeta := a.makeAppspaceMeta(*appspace)
		upgrade, ok, err := a.MigrationMinder.GetForAppspace(*appspace)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			upgradeMeta := makeVersionMeta(upgrade)
			appspaceMeta.Upgrade = &upgradeMeta
		}
		respData = append(respData, appspaceMeta)
	}

	writeJSON(res, respData)
}

func (a *AppspaceRoutes) getAppspaceFromPath(routeData *domain.AppspaceRouteData) (*domain.Appspace, error) {
	appspaceIDStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if appspaceIDStr == "" {
		return nil, nil
	}

	appspaceIDInt, err := strconv.Atoi(appspaceIDStr)
	if err != nil {
		return nil, errBadRequest
	}
	appspaceID := domain.AppspaceID(appspaceIDInt)

	appspace, err := a.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errNotFound
		}
		return nil, err
	}
	if appspace.OwnerID != routeData.Authentication.UserID {
		return nil, errForbidden
	}

	return appspace, nil
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

func (a *AppspaceRoutes) postNewAppspace(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	reqData := &PostAppspaceReq{}
	err := readJSON(req, reqData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	app, err := a.AppModel.GetFromID(reqData.AppID)
	if err != nil {
		if err != sql.ErrNoRows {
			http.Error(res, "App not found", http.StatusGone)
		} else {
			http.Error(res, err.Error(), 500)
		}
		return
	}
	if app.OwnerID != routeData.Authentication.UserID {
		http.Error(res, "Application not owned by logged in user", http.StatusForbidden)
		return
	}

	version, err := a.AppModel.GetVersion(app.AppID, reqData.Version)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(res, "Version not found", http.StatusGone)
		} else {
			http.Error(res, "", http.StatusInternalServerError)
		}
		return
	}

	// Here it would be nice if CheckAppspaceDomain also reserved that name temporarily
	check, err := a.DomainController.CheckAppspaceDomain(routeData.Authentication.UserID, reqData.DomainName, reqData.Subdomain)
	if err != nil {
		returnError(res, err)
		return
	}
	if !check.Valid || !check.Available {
		http.Error(res, "domain or subdomain no longer valid or available", http.StatusGone)
		return
	}

	fullDomain := reqData.DomainName
	if reqData.Subdomain != "" {
		fullDomain = reqData.Subdomain + "." + reqData.DomainName
	}

	// also need to validate dropid
	err = validator.DropIDFull(reqData.DropID)
	if err != nil {
		returnError(res, err)
		return
	}
	dropIDStr := validator.NormalizeDropIDFull(reqData.DropID)
	dropIDHandle, dropIDDomain := validator.SplitDropID(dropIDStr)
	dropID, err := a.DropIDModel.Get(dropIDHandle, dropIDDomain)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(res, "DropID not found", http.StatusGone)
		} else {
			returnError(res, err)
		}
		return
	}
	if dropID.UserID != routeData.Authentication.UserID {
		returnError(res, errors.New("DropID user does not match authenticated user"))
		return
	}

	locationKey, err := a.AppspaceFilesModel.CreateLocation()
	if err != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	inAppspace := domain.Appspace{
		OwnerID:     routeData.Authentication.UserID,
		AppID:       app.AppID,
		AppVersion:  version.Version,
		DomainName:  fullDomain,
		DropID:      reqData.DropID,
		LocationKey: locationKey,
	}

	appspace, err := a.AppspaceModel.Create(inAppspace)
	if err != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	err = a.AppspaceMetaDB.Create(appspace.AppspaceID, 0) // 0 is the ds api version
	if err != nil {
		http.Error(res, "Failed to create appspace meta db", http.StatusInternalServerError)
	}

	// Create owner user
	_, err = a.AppspaceUserModel.Create(appspace.AppspaceID, "dropid", dropIDStr)
	if err != nil {
		returnError(res, err)
		return
	}

	// set permissions for owner to max permissions.
	// 	err = a.AppspaceUserModel.UpdateMeta(appspace.AppspaceID, proxyID, displayName, []string{})
	// 	if err != nil {
	// 	}

	// migrate to whatever version was selected
	_, err = a.MigrationJobModel.Create(routeData.Authentication.UserID, appspace.AppspaceID, version.Version, true)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	a.MigrationJobController.WakeUp()

	resp := PostAppspaceResp{
		AppspaceID: appspace.AppspaceID}

	writeJSON(res, resp)
}

func (a *AppspaceRoutes) changeAppspacePause(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	if req.Method != http.MethodPost {
		http.Error(res, "expected POST", http.StatusBadRequest)
		return
	}

	reqData := PostAppspacePauseReq{}
	err := readJSON(req, &reqData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.AppspaceModel.Pause(appspace.AppspaceID, reqData.Pause)
	if err != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (a *AppspaceRoutes) deleteAppspace(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	// Pause or do a process pause..., then wait for stopped (like migration)
	// backup or download data or something?
	// actually delete row
}

func (a *AppspaceRoutes) makeAppspaceMeta(appspace domain.Appspace) AppspaceMeta {
	return AppspaceMeta{
		AppspaceID: int(appspace.AppspaceID),
		AppID:      int(appspace.AppID),
		AppVersion: appspace.AppVersion,
		DomainName: appspace.DomainName,
		NoSSL:      a.Config.Server.NoSsl,
		PortString: a.Config.Exec.PortString,
		DropID:     appspace.DropID,
		Paused:     appspace.Paused,
		Created:    appspace.Created}
}
