package userroutes

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
