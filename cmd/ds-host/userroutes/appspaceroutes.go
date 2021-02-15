package userroutes

import (
	"database/sql"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

//AppspaceMeta is
type AppspaceMeta struct {
	AppspaceID int            `json:"appspace_id"`
	AppID      int            `json:"app_id"`
	AppVersion domain.Version `json:"app_version"`
	Subdomain  string         `json:"subdomain"`
	Created    time.Time      `json:"created_dt"`
	Paused     bool           `json:"paused"`
	Upgrade    *VersionMeta   `json:"upgrade,omitempty"`
}

// AppspaceRoutes handles routes for appspace uploading, creating, deleting.
type AppspaceRoutes struct {
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
	}
	AppspaceFilesModel interface {
		CreateLocation() (string, error)
	}
	AppspaceModel interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, error)
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
		Create(domain.UserID, domain.AppID, domain.Version, string, string) (*domain.Appspace, error)
		Pause(domain.AppspaceID, bool) error
		GetFromSubdomain(string) (*domain.Appspace, error)
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

		switch head {
		case "":
			a.getAppspace(res, req, routeData, appspace)
		case "pause":
			a.changeAppspacePause(res, req, routeData, appspace)
		default:
			http.Error(res, "", http.StatusNotImplemented)
		}
	}
}

func (a *AppspaceRoutes) getAppspace(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	respData := makeAppspaceMeta(*appspace)
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

// GetAppspacesResp is
type GetAppspacesResp struct {
	Appspaces []AppspaceMeta `json:"appspaces"`
}

func (a *AppspaceRoutes) getAllAppspaces(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	appspaces, err := a.AppspaceModel.GetForOwner(routeData.Authentication.UserID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	respData := GetAppspacesResp{
		Appspaces: make([]AppspaceMeta, 0)}

	for _, appspace := range appspaces {
		appspaceMeta := makeAppspaceMeta(*appspace)
		upgrade, ok, err := a.MigrationMinder.GetForAppspace(*appspace)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			upgradeMeta := makeVersionMeta(upgrade)
			appspaceMeta.Upgrade = &upgradeMeta
		}
		respData.Appspaces = append(respData.Appspaces, appspaceMeta)
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

// temporary ubdomain gneration stuff
const charset = "abcdefghijklmnopqrstuvwxyz"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// PostAppspaceReq is sent when creating a new appspace
type PostAppspaceReq struct {
	AppID   domain.AppID   `json:"app_id"`
	Version domain.Version `json:"app_version"`
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

	// TODO: validate version before using it with DB. At least for size.

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

	// OK, so currently we are supposed to generate a subdomain.
	// This is very temporary because I want to move to user-chosen subdomains.
	// But let's get things working first.
	sub := a.getNewSubdomain()

	locationKey, err := a.AppspaceFilesModel.CreateLocation()
	if err != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	appspace, err := a.AppspaceModel.Create(routeData.Authentication.UserID, app.AppID, version.Version, sub, locationKey)
	if err != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	err = a.AppspaceMetaDB.Create(appspace.AppspaceID, 0) // 0 is the ds api version
	if err != nil {
		http.Error(res, "Failed to create appspace meta db", http.StatusInternalServerError)
	}

	// migrate to whatever version was selected
	// TODO: Must block appspace from being used until migration is done
	// I think this is done by appspace status

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

func (a *AppspaceRoutes) getNewSubdomain() (sub string) {
	for i := 0; i < 10; i++ {
		sub = randomSubomainString()
		_, err := a.AppspaceModel.GetFromSubdomain(sub)
		if err == nil {
			break
		}
	}
	return
}

func makeAppspaceMeta(appspace domain.Appspace) AppspaceMeta {
	return AppspaceMeta{
		AppspaceID: int(appspace.AppspaceID),
		AppID:      int(appspace.AppID),
		AppVersion: appspace.AppVersion,
		Subdomain:  appspace.Subdomain,
		Paused:     appspace.Paused,
		Created:    appspace.Created}
}

func randomSubomainString() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
