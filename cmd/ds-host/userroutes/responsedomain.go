package userroutes

// This may no longer be reelevant. We are not generating TS types.

import (
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

// attempt to create a set of types that define the API surface
// both requests and Responses , denoted by 'req' and 'resp' (?)

// How to organize this?

// Follow order of files of route handlers:

////// Admin Routes:

// AdminGetUsersResp is GET
type AdminGetUsersResp struct {
	Users []UserData `json:"users"`
}

// AdminGetUserInvitationsResp is Response toGet user invitation
type AdminGetUserInvitationsResp struct {
	UserInvitations []*domain.UserInvitation `json:"user_invitations"`
}

//AdminPostUserInvitationReq is
// TODO: this one should be an interface only
type AdminPostUserInvitationReq struct {
	UserInvitation domain.UserInvitation `json:"user_invitation"`
}

// GetSettingsResp is
type GetSettingsResp struct {
	Settings domain.Settings `json:"settings"`
}

// PostSettingsReq is
// TODO: this one should be an interface only
type PostSettingsReq struct {
	Settings domain.Settings `json:"settings"`
}

////// App Routes:

// GetAppsResp is
type GetAppsResp struct {
	Apps []ApplicationMeta `json:"apps"`
}

// don't we need PostAppReq?

// PostAppResp is response to creating an application
type PostAppResp struct {
	AppMeta ApplicationMeta `json:"app_meta"`
}

// don't we need post version req?

// PostVersionResp is
type PostVersionResp struct {
	VersionMeta VersionMeta `json:"version_meta"`
}

////// Appspace Routes:

// GetAppspacesResp is
type GetAppspacesResp struct {
	Appspaces []AppspaceMeta `json:"appspaces"`
}

// PostAppspaceReq is
// TODO: this one should be an interface only
type PostAppspaceReq struct {
	AppID   domain.AppID   `json:"app_id"`
	Version domain.Version `json:"version"`
}

// PostAppspaceResp is
type PostAppspaceResp struct {
	JobID        domain.JobID `json:"job_id"`
	AppspaceMeta AppspaceMeta `json:"appspace"`
}

////// Auth Routes:

////// User Routes:

// PatchPasswordReq is
// type PatchPasswordReq struct {
// 	Old string `json:"old"`
// 	New string `json:"new"`
//}

// Live data routes

// GetStartLiveDataResp holds the token necessary to start a websocket upgraded conn
type GetStartLiveDataResp struct {
	Token string `json:"token"`
}

// MigrationJobResp describes a pending or ongoing appspace migration job
type MigrationJobResp struct {
	JobID      domain.JobID         `json:"job_id"`
	OwnerID    domain.UserID        `json:"owner_id"`
	AppspaceID domain.AppspaceID    `json:"appspace_id"`
	ToVersion  domain.Version       `json:"to_version"`
	Created    time.Time            `json:"created"`
	Started    nulltypes.NullTime   `json:"started"`
	Finished   nulltypes.NullTime   `json:"finished"`
	Priority   bool                 `json:"priority"`
	Error      nulltypes.NullString `json:"error"`
}

// MigrationStatusResp reflects the current status of the migrationJob referenced
type MigrationStatusResp struct {
	JobID        domain.JobID         `json:"job_id"`
	MigrationJob *MigrationJobResp    `json:"migration_job,omitempty"`
	Status       string               `json:"status"`
	Started      nulltypes.NullTime   `json:"started"`
	Finished     nulltypes.NullTime   `json:"finished"`
	Error        nulltypes.NullString `json:"error"`
	CurSchema    int                  `json:"cur_schema"`
}

////// Common stuff.....

// UserData is single user
type UserData struct {
	Email   string `json:"email"`
	UserID  int    `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
}

// ApplicationMeta is an application's metadata
type ApplicationMeta struct {
	AppID    int           `json:"app_id"`
	AppName  string        `json:"app_name"`
	Created  time.Time     `json:"created_dt"`
	Versions []VersionMeta `json:"versions"`
}

// VersionMeta is for listing versions of application code
type VersionMeta struct {
	AppName string         `json:"app_name"`
	Version domain.Version `json:"version"`
	Schema  int            `json:"schema"`
	Created time.Time      `json:"created_dt"`
}

//AppspaceMeta is
type AppspaceMeta struct {
	AppspaceID int            `json:"appspace_id"`
	AppID      int            `json:"app_id"`
	AppVersion domain.Version `json:"app_version"`
	Subdomain  string         `json:"subdomain"`
	Created    time.Time      `json:"created_dt"`
	Paused     bool           `json:"paused"`
}
