package userapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mfcochauxlaberge/jsonapi"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/jsonapirouter"
)

// AppVersion is for listing versions of application code
type AppVersion struct {
	ID         string    `json:"id" api:"app_versions"`
	Name       string    `json:"name"  api:"attr"`
	Version    string    `json:"version"  api:"attr"`
	APIVersion int       `json:"api" api:"attr"`
	Schema     int       `json:"schema" api:"attr"`
	Created    time.Time `json:"created_dt" api:"attr"`
	App        string    `json:"app" api:"rel,apps,versions"`
	Appspaces  []string  `json:"appspaces" api:"rel,appspaces,app_version"`
}

func makeAppVersion(a domain.AppVersion) AppVersion {
	return AppVersion{
		ID:         appVersionID(a.AppID, a.Version),
		Name:       a.AppName,
		Version:    string(a.Version),
		APIVersion: int(a.APIVersion),
		Schema:     a.Schema,
		Created:    a.Created,
	}
}
func wrapAppVersion(a domain.AppVersion) jsonapi.Resource {
	aa := makeAppVersion(a)
	return jsonapi.Wrap(&aa)
}

func appVersionID(appID domain.AppID, version domain.Version) string {
	return fmt.Sprintf("%v-%s", appID, version)
}
func parseAppVersionID(appVersionID string) (appID domain.AppID, version domain.Version, err error) {
	//split on first '-'
	pieces := strings.SplitN(appVersionID, "-", 2)
	if len(pieces) != 2 {
		err = errors.New("failed to parse app version id: " + appVersionID)
		return
	}
	var appIDInt int
	appIDInt, err = strconv.Atoi(pieces[0])
	if err != nil {
		err = errors.New("failed to parse app version id: " + appVersionID)
		return
	}
	appID = domain.AppID(appIDInt)

	if len(pieces[1]) == 0 {
		err = errors.New("failed to parse app version id: " + appVersionID)
		return
	}
	// could validate version further.
	version = domain.Version(pieces[1])

	return
}

func getAppVersionHandler(api *UserJSONAPI) jsonapirouter.JSONAPIRouteHandler {
	return func(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
		auth := api.authenticateUser(res, req)
		if !auth.Authenticated {
			return jsonapirouter.Unauthorized
		}

		appID, version, err := parseAppVersionID(rReq.URL.ResID)
		if err != nil {
			return jsonapirouter.Error
		}
		app, err := api.AppModel.GetFromID(appID)
		if err != nil {
			return jsonapirouter.NotFound // or is this OK with an empty data?
		}
		if app.OwnerID != auth.UserID {
			return jsonapirouter.Unauthorized
		}
		appVersion, err := api.AppModel.GetVersion(appID, version)
		if err != nil {
			return jsonapirouter.OK
		}
		apiAppVersion := makeAppVersion(*appVersion)

		appspaces, dsErr := api.AppspaceModel.GetForAppVersion(appID, version)
		if dsErr != nil {
			return jsonapirouter.Error
		}

		apiAppVersion.Appspaces = make([]string, len(appspaces))
		for i, appspace := range appspaces {
			apiAppVersion.Appspaces[i] = fmt.Sprint(appspace.AppspaceID)
			rReq.Includes.HoldResource(wrapAppspace(*appspace))
		}

		rReq.Doc.Data = jsonapi.Wrap(apiAppVersion)

		rReq.Doc.RelData = map[string][]string{
			"app_versions": {"appspaces"},
		}

		return jsonapirouter.OK
	}
}

// does this one even make any sense?
// It seems what we are looking for is getApps
// with includes of appVersions, and possible appspaces too
func getAppVersionsHandler(api *UserJSONAPI) jsonapirouter.JSONAPIRouteHandler {
	return func(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
		auth := api.authenticateUser(res, req)
		if !auth.Authenticated {
			return jsonapirouter.Unauthorized
		}

		filterLabel := rReq.URL.Params.FilterLabel
		if filterLabel != "owner" { // wait shouldn't filter be filter=owner:1
			//http.Error(res, "missing filter by owner", 403)
			return jsonapirouter.Unauthorized
		}
		// do different filters later,
		// .. or allow unfiltered results for admin

		//appVersion

		appspaces, dsErr := api.AppspaceModel.GetForOwner(auth.UserID)
		if dsErr != nil {
			return jsonapirouter.Error
		}

		appspacesCol := api.router.NewCollection("appspaces")
		for _, appspace := range appspaces {

			appspacesCol.Add(wrapAppspace(*appspace))
		}
		rReq.Doc.Data = appspacesCol

		// This adds the type and id of related resource in the data:
		rReq.Doc.RelData = map[string][]string{ // why is this not taken care of by the URL
			"appspaces": {"app_version"},
		}

		return jsonapirouter.OK
	}
}

//// data loaders
func getAppVersionsLoader(api *UserJSONAPI) jsonapirouter.JSONAPIDataLoader {
	return func(ids []string, rReq *jsonapirouter.RouterReq) ([]jsonapi.Resource, error) {
		// would like to get auth with that....
		ret := make([]jsonapi.Resource, len(ids))
		for i, id := range ids {
			appID, version, err := parseAppVersionID(id)
			if err != nil {
				return nil, err
			}

			appVersion, err := api.AppModel.GetVersion(appID, version)
			if err != nil {
				return nil, err
			}

			ret[i] = wrapAppVersion(*appVersion)
		}

		return ret, nil
	}
}
