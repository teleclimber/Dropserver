package userroutes

import (
	"net/http"
	"math/rand"
	"time"
  

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// AppspaceRoutes handles routes for applications uploading, creating, deleting.
type AppspaceRoutes struct {
	AppspaceModel domain.AppspaceModel
	AppModel      domain.AppModel
	Logger domain.LogCLientI
}

// ServeHTTP handles http traffic to the appspace routes
// Namely create, delete, set version, etc...
func (a *AppspaceRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Cookie == nil || !routeData.Cookie.UserAccount {
		// maybe log it? Frankly this should be a panic.
		// It's programmer error pure and simple. Kill this thing.
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we dun fogged up
	}

	appspaceIDStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	method := req.Method

	if appspaceIDStr == "" {
		switch method {
		case http.MethodGet:
			a.getAllAppspaces(res, req, routeData)
		case http.MethodPost:
			a.postNewAppspace(res, req, routeData)
		default:
			http.Error(res, "bad method for /application", http.StatusBadRequest)
		}
	} else {
		// check appspace existence
		routeData.URLTail = tail //maybe?
		http.Error(res, "", http.StatusBadRequest)
	}
}

func (a *AppspaceRoutes) getAllAppspaces(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	appspaces, dsErr := a.AppspaceModel.GetForOwner(routeData.Cookie.UserID)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	respData := getAppspacesResp{
		Appspaces: make([]appspaceMeta, 0)}

	for _, appspace := range appspaces {
		respData.Appspaces = append(respData.Appspaces, appspaceMeta{
			AppID:     int(appspace.AppID),
			AppVersion:   string(appspace.AppVersion),
			Subdomain: appspace.Subdomain}) // yeah, subdomain versus name. Gonna need to do some work here.
	}

	writeJSON(res, respData)
}

// temporary ubdomain gneration stuff
const charset = "abcdefghijklmnopqrstuvwxyz"
var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))
  
func (a *AppspaceRoutes) postNewAppspace(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	reqData := &postAppspaceReq{}
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
	if app.OwnerID != routeData.Cookie.UserID {
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

	appspace, dsErr := a.AppspaceModel.Create(routeData.Cookie.UserID, app.AppID, version.Version, sub)
	if dsErr != nil {
		http.Error(res, "", http.StatusInternalServerError)
	}

	// return appspace Meta
	resp := postAppspaceResp{
		AppspaceMeta: appspaceMeta{
			AppID: int(appspace.AppID),
			AppVersion: string(appspace.AppVersion),
			Subdomain: appspace.Subdomain,
			Paused: appspace.Paused,
			Created: appspace.Created}}
	
	writeJSON(res, resp)
}

func (a *AppspaceRoutes) getNewSubdomain() (sub string) {
	for i := 0; i<10; i++ {
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
  
