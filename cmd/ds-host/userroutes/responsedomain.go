package userroutes

// This may no longer be reelevant. We are not generating TS types.

// Indeed , move relevant types to be close to where they are used.

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
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

////// Common stuff.....

// UserData is single user
type UserData struct {
	Email   string `json:"email"`
	UserID  int    `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
}
