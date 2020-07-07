package userroutes

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// This may no longer be reelevant. We are not generating TS types.

////// User Routes:

// PatchPasswordReq is
type PatchPasswordReq struct {
	Old string `json:"old"`
	New string `json:"new"`
}

///////// appspace routes

// PostAppspacePauseReq is
type PostAppspacePauseReq struct {
	Pause bool `json:"pause"`
}

// PostAppspaceVersionReq is
type PostAppspaceVersionReq struct {
	Version domain.Version `json:"version"` // could include app_id to future proof and to verify apples-apples
}
