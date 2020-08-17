package userroutes

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// AppspaceRoutes handles routes for appspace uploading, creating, deleting.
type AppspaceRoutes struct {
	AppModel interface {
		GetFromID(domain.AppID) (*domain.App, domain.Error)
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, domain.Error)
	}
	AppspaceFilesModel domain.AppspaceFilesModel
	AppspaceModel      interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, domain.Error)
		GetFromID(domain.AppspaceID) (*domain.Appspace, domain.Error)
		Create(domain.UserID, domain.AppID, domain.Version, string, string) (*domain.Appspace, domain.Error)
		Pause(domain.AppspaceID, bool) domain.Error
		GetFromSubdomain(string) (*domain.Appspace, domain.Error)
	}
	AppspaceMetaDB         domain.AppspaceMetaDB
	MigrationJobModel      domain.MigrationJobModel
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

	appspace, dsErr := a.getAppspaceFromPath(routeData)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	if appspace == nil {
		switch req.Method {
		case http.MethodGet:
			a.getAllAppspaces(res, req, routeData)
		case http.MethodPost:
			a.postNewAppspace(res, req, routeData)
		default:
			http.Error(res, "bad method for /application", http.StatusBadRequest)
		}
	} else {
		head, tail := shiftpath.ShiftPath(routeData.URLTail)
		routeData.URLTail = tail

		switch head {
		case "pause":
			a.changeAppspacePause(res, req, routeData, appspace)
		case "version":
			a.changeAppspaceVersion(res, req, routeData, appspace)
		default:
			http.Error(res, "", http.StatusNotImplemented)
		}
	}
}

func (a *AppspaceRoutes) getAllAppspaces(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	appspaces, dsErr := a.AppspaceModel.GetForOwner(routeData.Authentication.UserID)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	respData := GetAppspacesResp{
		Appspaces: make([]AppspaceMeta, 0)}

	for _, appspace := range appspaces {
		respData.Appspaces = append(respData.Appspaces, AppspaceMeta{
			AppspaceID: int(appspace.AppspaceID),
			AppID:      int(appspace.AppID),
			AppVersion: appspace.AppVersion,
			Subdomain:  appspace.Subdomain, // yeah, subdomain versus name. Gonna need to do some work here.
			Paused:     appspace.Paused,
			Created:    appspace.Created})
	}

	writeJSON(res, respData)
}

func (a *AppspaceRoutes) getAppspaceFromPath(routeData *domain.AppspaceRouteData) (*domain.Appspace, domain.Error) {
	appspaceIDStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if appspaceIDStr == "" {
		return nil, nil
	}

	appspaceIDInt, err := strconv.Atoi(appspaceIDStr)
	if err != nil {
		return nil, dserror.New(dserror.BadRequest)
	}
	appspaceID := domain.AppspaceID(appspaceIDInt)

	appspace, dsErr := a.AppspaceModel.GetFromID(appspaceID)
	if dsErr != nil {
		return nil, dsErr
	}
	if appspace.OwnerID != routeData.Authentication.UserID {
		return nil, dserror.New(dserror.Unauthorized)
	}

	return appspace, nil
}

// temporary ubdomain gneration stuff
const charset = "abcdefghijklmnopqrstuvwxyz"

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func (a *AppspaceRoutes) postNewAppspace(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	reqData := &PostAppspaceReq{}
	dsErr := readJSON(req, reqData)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	// TODO: validate version before using it with DB. At least for size.

	app, dsErr := a.AppModel.GetFromID(reqData.AppID)
	if dsErr != nil {
		if dsErr.Code() == dserror.NoRowsInResultSet {
			// means we didn't find the application.
		}
		dsErr.HTTPError(res)
		return
	}
	if app.OwnerID != routeData.Authentication.UserID {
		http.Error(res, "Application not owned by logged in user", http.StatusUnauthorized)
		// this could just be internal error? because this shouldn't happen unless we made a mistake?
		return
	}

	version, dsErr := a.AppModel.GetVersion(app.AppID, reqData.Version)
	if dsErr != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	// OK, so currently we are supposed to generate a subdomain.
	// This is very temporary because I want to move to user-chosen subdomains.
	// But let's get things working first.
	sub := a.getNewSubdomain()

	locationKey, dsErr := a.AppspaceFilesModel.CreateLocation()
	if dsErr != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	appspace, dsErr := a.AppspaceModel.Create(routeData.Authentication.UserID, app.AppID, version.Version, sub, locationKey)
	if dsErr != nil {
		http.Error(res, "", http.StatusInternalServerError)
		return
	}

	err := a.AppspaceMetaDB.Create(appspace.AppspaceID, 0) // 0 is the ds api version
	if err != nil {
		http.Error(res, "Failed to create appspace meta db", http.StatusInternalServerError)
	}

	// migrate to whatever version was selected
	// TODO: Must block appspace from being used until migration is done

	job, dsErr := a.MigrationJobModel.Create(routeData.Authentication.UserID, appspace.AppspaceID, version.Version, true)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	a.MigrationJobController.WakeUp()

	// return appspace Meta
	resp := PostAppspaceResp{
		JobID: job.JobID,
		AppspaceMeta: AppspaceMeta{
			AppspaceID: int(appspace.AppspaceID),
			AppID:      int(appspace.AppID),
			AppVersion: appspace.AppVersion,
			Subdomain:  appspace.Subdomain,
			Paused:     appspace.Paused,
			Created:    appspace.Created}}

	writeJSON(res, resp)
}

func (a *AppspaceRoutes) changeAppspacePause(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	if req.Method != http.MethodPost {
		http.Error(res, "expected POST", http.StatusBadRequest)
		return
	}

	reqData := PostAppspacePauseReq{}
	dsErr := readJSON(req, &reqData)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	dsErr = a.AppspaceModel.Pause(appspace.AppspaceID, reqData.Pause)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func (a *AppspaceRoutes) changeAppspaceVersion(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, appspace *domain.Appspace) {
	if req.Method != http.MethodPost {
		http.Error(res, "expected POST", http.StatusBadRequest)
		return
	}

	reqData := PostAppspaceVersionReq{}
	dsErr := readJSON(req, &reqData)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	// minimally validate version string? At least to see if it's not a huge string that would bog down the DB

	_, dsErr = a.AppModel.GetVersion(appspace.AppID, reqData.Version)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	_, dsErr = a.MigrationJobModel.Create(routeData.Authentication.UserID, appspace.AppspaceID, reqData.Version, true)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	a.MigrationJobController.WakeUp()

	res.WriteHeader(http.StatusOK)
}

func (a *AppspaceRoutes) getNewSubdomain() (sub string) {
	for i := 0; i < 10; i++ {
		sub = randomSubomainString()
		_, dsErr := a.AppspaceModel.GetFromSubdomain(sub)
		if dsErr == nil {
			break
		}
	}
	return
}

func randomSubomainString() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
