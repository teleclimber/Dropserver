package vxservices

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

type serviceGetter interface {
	GetService(appspace *domain.Appspace) domain.ReverseServiceI
}

// VXServices holds the structs necessary to create a service for any api version
type VXServices struct {
	RouteModels interface {
		GetV0(appspaceID domain.AppspaceID) domain.V0RouteModel
	}
	V0AppspaceDB serviceGetter
}

//Get returns a reverse service for the appspace
func (x *VXServices) Get(appspace *domain.Appspace, api domain.APIVersion) (service domain.ReverseServiceI) {
	switch api {
	case 0:
		service = &V0Services{
			RouteModel: x.RouteModels.GetV0(appspace.AppspaceID),
			AppspaceDB: x.V0AppspaceDB.GetService(appspace),
		}
	}
	return
}
