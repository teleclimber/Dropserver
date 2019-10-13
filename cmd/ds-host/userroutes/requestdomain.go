package userroutes

//go:generate struct2ts -i -o ../../../frontend/generated-types/userroutes-interfaces.ts userroutes.PatchPasswordReq

//"time"

//"github.com/teleclimber/DropServer/cmd/ds-host/domain"

////// User Routes:

// PatchPasswordReq is
type PatchPasswordReq struct {
	Old string `json:"old"`
	New string `json:"new"`
}
