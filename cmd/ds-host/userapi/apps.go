package userapi

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mfcochauxlaberge/jsonapi"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/jsonapirouter"
)

// App is an application's metadata
type App struct {
	ID       string    `json:"id" api:"apps"`
	Name     string    `json:"name" api:"attr"`
	Created  time.Time `json:"created_dt" api:"attr"`
	Versions []string  `json:"versions" api:"rel,app_versions,app"`
	//Owner    string    `json:"owner" api:"rel,users"`
}

func makeApp(a domain.App) App {
	return App{
		ID:      fmt.Sprint(a.AppID),
		Name:    a.Name,
		Created: a.Created,
		// Versions?
		// Owner?
	}
}

func wrapApp(a domain.App) jsonapi.Resource {
	aa := makeApp(a)
	return jsonapi.Wrap(&aa)
}

func getAppHandler(api *UserJSONAPI) jsonapirouter.JSONAPIRouteHandler {
	return func(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
		auth := api.authenticateUser(res, req)
		if !auth.Authenticated {
			return jsonapirouter.Unauthorized
		}

		appIDInt, err := strconv.Atoi(rReq.URL.ResID)
		appID := domain.AppID(appIDInt)
		app, err := api.AppModel.GetFromID(appID)
		if err != nil {
			return jsonapirouter.NotFound // or is this OK with an empty data?
		}
		if app.OwnerID != auth.UserID {
			return jsonapirouter.Unauthorized
		}

		apiApp := makeApp(*app)

		// get versions:
		versions, err := api.AppModel.GetVersionsForApp(appID)
		if err != nil {
			return jsonapirouter.Error // or is this OK with an empty data?
		}
		apiApp.Versions = make([]string, len(versions))
		for i, ver := range versions {
			apiApp.Versions[i] = appVersionID(appID, ver.Version)
			rReq.Includes.HoldResource(wrapAppVersion(*ver))
		}

		rReq.Doc.Data = jsonapi.Wrap(apiApp)

		// This adds the type and id of related resource in the data:
		rReq.Doc.RelData = map[string][]string{
			"apps": {"versions"},
		}

		return jsonapirouter.OK
	}
}

func getAppsHandler(api *UserJSONAPI) jsonapirouter.JSONAPIRouteHandler {
	return func(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
		auth := api.authenticateUser(res, req)
		if !auth.Authenticated {
			return jsonapirouter.Unauthorized
		}

		filterLabel := rReq.URL.Params.FilterLabel
		if filterLabel != "owner" { // wait shouldn't filter be filter=owner:1
			return jsonapirouter.Unauthorized
		}

		apps, err := api.AppModel.GetForOwner(auth.UserID)
		if err != nil {
			return jsonapirouter.Error // or is this OK with an empty data?
		}

		appsCol := api.router.NewCollection("apps")
		for _, app := range apps {
			apiApp := makeApp(*app)

			appVersions, err := api.AppModel.GetVersionsForApp(app.AppID)
			if err != nil {
				return jsonapirouter.Error
			}
			apiApp.Versions = make([]string, len(appVersions))
			for i, appV := range appVersions {
				apiApp.Versions[i] = appVersionID(app.AppID, appV.Version)
				rReq.Includes.HoldResource(wrapAppVersion(*appV))
			}

			appsCol.Add(jsonapi.Wrap(apiApp))
		}
		rReq.Doc.Data = appsCol

		rReq.Doc.RelData = map[string][]string{
			"apps": {"versions"},
		}

		return jsonapirouter.OK
	}
}

func getAppsLoader(api *UserJSONAPI) jsonapirouter.JSONAPIDataLoader {
	return func(ids []string, rReq *jsonapirouter.RouterReq) ([]jsonapi.Resource, error) {
		ret := make([]jsonapi.Resource, len(ids))
		for i, id := range ids {
			idInt, err := strconv.Atoi(id)
			if err != nil {
				return nil, err
			}
			app, err := api.AppModel.GetFromID(domain.AppID(idInt))
			if err != nil {
				return nil, err
			}
			ret[i] = wrapApp(*app)
		}

		return ret, nil
	}
}
