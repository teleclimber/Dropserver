package userroutes

import (
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// attempt to create a set of types that define the API surface
// both requests and Responses , denoted by 'req' and 'resp' (?)

// user

// single user
type userResp struct {
	Email   string `json:"email"`
	UserID  int    `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
}

type adminGetUsersResp struct {
	Users []userResp `json:"users"`
}

type getUserInvitationsResp struct {
	UserInvitations []*domain.UserInvitation `json:"user_invitations"`
}
type postUserInvitationReq struct {
	UserInvitation domain.UserInvitation `json:"user_invitation"`
}

type changePwData struct {
	Old string
	New string
}

// settings:
type getSettingsResp struct {
	Settings domain.Settings `json:"settings"`
}

type postSettingsReq struct {
	Settings domain.Settings `json:"settings"`
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
	Schema  int       `json:"schema"`
	Created time.Time `json:"created_dt"`
}

type createAppResp struct {
	AppMeta appMeta `json:"app_meta"`
}

type createVersionResp struct {
	VersionMeta versionMeta `json:"version_meta"`
}

// appspaces:
type appspaceMeta struct {
	AppspaceID int       `json:"appspace_id"`
	AppID      int       `json:"app_id"`
	AppVersion string    `json:"app_version"`
	Subdomain  string    `json:"subdomain"`
	Created    time.Time `json:"created_dt"`
	Paused     bool      `json:"paused"`
}

type getAppspacesResp struct {
	Appspaces []appspaceMeta `json:"appspaces"`
}

type postAppspaceReq struct {
	AppID   domain.AppID   `json:"app_id"`
	Version domain.Version `json:"version"`
}

type postAppspaceResp struct {
	AppspaceMeta appspaceMeta `json:"appspace"`
}

type changeVersionReq struct {
	Version domain.Version `json:"version"`
}
