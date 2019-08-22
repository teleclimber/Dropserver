package userroutes

import (
	"time"
)

// attempt to create a set of types that define the API surface
// both requests and Responses , denoted by 'req' and 'resp' (?)

// user

type userResp struct {
	Email string `json:"email"`
}

type changePwData struct {
	Old string
	New string
}

// application routes:

type appMeta struct {
	AppID    int           `json:"app_id"`
	AppName  string        `json:"app_name"`
	Created  time.Time     `json:"created_dt"`
	Versions []versionMeta `json:"versions"`
}
type getAppsResp struct {
	Apps []appMeta `json:"apps"`
}

// versionListMeta is for listing versions of application code
type versionMeta struct {
	Version string    `json:"version"`
	Created time.Time `json:"created_dt"`
}

type createAppResp struct {
	AppMeta appMeta `json:"app_meta"`
}

type createVersionResp struct {
	VersionMeta versionMeta `json:"version_meta"`
}
