package userapi

// JSON:API
// v0 because it ought to be versioned.
// One big thing? a separate package? How to organize?

import (
	"fmt"
	"net/http"

	"github.com/mfcochauxlaberge/jsonapi"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/jsonapirouter"
)

// Our types:
// - user (current user)
// - apps
// - appversions
// - appspaces (owner-only appspaces or remotes too?)
// - contacts
// - appspaceusers? (because appspace-contacts relationship is only part of the users eventually)

// relationships:
// - apps
//    versions: *appversions
// - appversions
//    appspaces: *appspaces
// - appspaces
//    appversion: appversion
//    known-users: *contacts
//    self-reg-users: *appspaceusers?
// - contacts
//    appspaces: *appspaces
//    [authentication mechanisms]
//

// UserJSONAPI handles requests for the json api
type UserJSONAPI struct {
	schema *jsonapi.Schema
	router *jsonapirouter.JSONAPIRouter
	Auth   interface {
		Authenticate(req *http.Request) domain.Authentication
		// authorized?
	}
	AppspaceModel interface {
		GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, domain.Error)
		GetForOwner(userID domain.UserID) ([]*domain.Appspace, domain.Error)
		GetForAppVersion(appID domain.AppID, version domain.Version) ([]*domain.Appspace, domain.Error)
	}
	AppModel interface {
		GetFromID(appID domain.AppID) (*domain.App, error)
		GetForOwner(userID domain.UserID) ([]*domain.App, error)
		GetVersionsForApp(appID domain.AppID) ([]*domain.AppVersion, error)
		GetVersion(appID domain.AppID, version domain.Version) (*domain.AppVersion, error)
	}
}

// Init creates the schema
func (api *UserJSONAPI) Init() {
	api.schema = &jsonapi.Schema{}
	api.router = jsonapirouter.NewJSONAPIRouter(api.schema)

	api.schema.AddType(jsonapi.MustBuildType(App{}))
	api.router.AddLoader("apps", getAppsLoader(api))
	api.router.GetCollection("apps", getAppsHandler(api))
	api.router.GetResource("apps", getAppHandler(api))

	api.schema.AddType(jsonapi.MustBuildType(Appspace{}))
	api.router.AddLoader("appspaces", getAppspacesLoader(api))
	api.router.GetCollection("appspaces", getAppspacesHandler(api))
	api.router.GetResource("appspaces", getAppspaceHandler(api))

	// api.router.GetRelated("appspaces", "app-version", api.getAppspaceAppVersion)
	// api.router.GetRelationships("appspaces", "app-version", api.getAppspaceAppVersionRel)

	api.schema.AddType(jsonapi.MustBuildType(AppVersion{}))
	api.router.AddLoader("app_versions", getAppVersionsLoader(api))
	api.router.GetCollection("app_versions", getAppVersionsHandler(api))
	api.router.GetResource("app_versions", getAppVersionHandler(api))
	//api.router.GetRelated("app_versions", "appspaces", api.getAppVersionAppspaces)

	errs := api.schema.Check()
	if len(errs) != 0 {
		for _, e := range errs {
			fmt.Printf("%v \n", e)
		}
		panic(errs)
	}

}

// ServeHTTP forwards the http request to the router
func (api *UserJSONAPI) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	api.router.ServeHTTP(res, req)
}

func (api *UserJSONAPI) authenticateUser(res http.ResponseWriter, req *http.Request) domain.Authentication {
	auth := api.Auth.Authenticate(req)
	if !auth.Authenticated {
		res.WriteHeader(http.StatusUnauthorized)
		// is there anything mandated by json:api?
	}
	return auth
}

// also do AuthenticateAdmin
