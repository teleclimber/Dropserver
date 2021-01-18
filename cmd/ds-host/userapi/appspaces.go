package userapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/mfcochauxlaberge/jsonapi"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/jsonapirouter"
)

//Appspace is
type Appspace struct {
	ID         string    `json:"id" api:"appspaces"`
	Subdomain  string    `json:"subdomain" api:"attr"`
	Created    time.Time `json:"created_dt" api:"attr"`
	Paused     bool      `json:"paused" api:"attr"`
	AppVersion string    `json:"app_version" api:"rel,app_versions,appspaces"`
	//Owner string `json:"owner" api:"rel,..`
	// UpgradeAvailable string // attribute that is an optional field? Or it needs its own endpoint? Or it's a relationship?
}

// owner id? -> as string / relation?

func makeAppspace(a domain.Appspace) Appspace {
	return Appspace{
		ID:         fmt.Sprint(a.AppspaceID),
		Subdomain:  a.Subdomain,
		Created:    a.Created,
		Paused:     a.Paused,
		AppVersion: appVersionID(a.AppID, a.AppVersion),
	}
}
func wrapAppspace(a domain.Appspace) jsonapi.Resource {
	aa := makeAppspace(a)
	return jsonapi.Wrap(&aa)
}

// /appspaces/1
func getAppspaceHandler(api *UserJSONAPI) jsonapirouter.JSONAPIRouteHandler {
	return func(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
		auth := api.authenticateUser(res, req)
		if !auth.Authenticated {
			return jsonapirouter.Forbidden
		}

		appspaceID, err := strconv.Atoi(rReq.URL.ResID) // Here we could have a loadAppspaces that accept a string ID.
		if err != nil {
			return jsonapirouter.Error
		}
		appspace, err := api.AppspaceModel.GetFromID(domain.AppspaceID(appspaceID))
		if err != nil {
			return jsonapirouter.NotFound
		}
		if appspace.OwnerID != auth.UserID {
			return jsonapirouter.Forbidden
		}

		rReq.Doc.Data = wrapAppspace(*appspace)

		// This adds the type and id of related resource in the data:
		rReq.Doc.RelData = map[string][]string{
			"appspaces": {"app_version"},
		}

		return jsonapirouter.OK
	}
}

func getAppspacesHandler(api *UserJSONAPI) jsonapirouter.JSONAPIRouteHandler {
	return func(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
		auth := api.authenticateUser(res, req)
		if !auth.Authenticated {
			return jsonapirouter.Forbidden
		}

		filterLabel := rReq.URL.Params.FilterLabel
		if filterLabel != "owner" { // wait shouldn't filter be filter=owner:1
			return jsonapirouter.Forbidden
		}
		// do different filters later,
		// .. or allow unfiltered results for admin

		appspaces, err := api.AppspaceModel.GetForOwner(auth.UserID)
		if err != nil {
			return jsonapirouter.Error
		}

		appspacesCol := api.router.NewCollection("appspaces")
		for _, appspace := range appspaces {
			appspacesCol.Add(wrapAppspace(*appspace))
		}
		rReq.Doc.Data = appspacesCol

		// This adds the type and id of related resource in the data:
		rReq.Doc.RelData = map[string][]string{
			"appspaces": {"app_version"},
		}

		return jsonapirouter.OK
	}
}

// /appspaces/1/relationships/app_version
// func (api *UserJSONAPI) getAppspaceAppVersionRel(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
// 	auth := api.authenticateUser(res, req)
// 	if !auth.Authenticated {
// 		return jsonapirouter.Forbidden
// 	}

// 	return jsonapirouter.Error
// }

// func (api *UserJSONAPI) getAppspaceAppVersion(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
// 	return jsonapirouter.Error
// }

// func (api *UserJSONAPI) getAppVersionAppspaces(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
// 	return jsonapirouter.Error
// }

// PATCH /appspaces/1
func updateAppspaceHandler(api *UserJSONAPI) jsonapirouter.JSONAPIRouteHandler {
	return func(res http.ResponseWriter, req *http.Request, rReq *jsonapirouter.RouterReq) jsonapirouter.Status {
		auth := api.authenticateUser(res, req)
		if !auth.Authenticated {
			return jsonapirouter.Forbidden
		}

		appspaceID, err := strconv.Atoi(rReq.URL.ResID) // Here we could have a loadAppspaces that accept a string ID.
		if err != nil {
			return jsonapirouter.Error
		}
		appspace, err := api.AppspaceModel.GetFromID(domain.AppspaceID(appspaceID))
		if err != nil {
			return jsonapirouter.NotFound
		}
		if appspace.OwnerID != auth.UserID {
			return jsonapirouter.Forbidden
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return jsonapirouter.Error
		}
		doc, err := jsonapi.UnmarshalDocument(body, api.schema)
		if err != nil {
			return jsonapirouter.Error
		}
		resource, _ := doc.Data.(jsonapi.Resource)
		paused := resource.Get("paused").(bool)

		err = api.AppspaceModel.Pause(appspace.AppspaceID, paused)
		if err != nil {
			return jsonapirouter.Error
		}

		return jsonapirouter.OK
	}
}

func getAppspacesLoader(api *UserJSONAPI) jsonapirouter.JSONAPIDataLoader {
	return func(ids []string, rReq *jsonapirouter.RouterReq) ([]jsonapi.Resource, error) {
		ret := make([]jsonapi.Resource, len(ids))
		for i, id := range ids {
			idInt, err := strconv.Atoi(id)
			if err != nil {
				return nil, err
			}
			appspace, err := api.AppspaceModel.GetFromID(domain.AppspaceID(idInt))
			if err != nil {
				return nil, err
			}
			ret[i] = wrapAppspace(*appspace)
		}

		return ret, nil
	}
}
