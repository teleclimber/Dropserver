package vxservices

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

type serviceGetter interface {
	GetService(appspace *domain.Appspace) domain.ReverseServiceI
}

// VXServices holds the structs necessary to create a service for any api version
type VXServices struct {
	AppspaceUsersV0 interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	}
	V0AppspaceDB serviceGetter
}

//Get returns a reverse service for the appspace
func (x *VXServices) Get(appspace *domain.Appspace, api domain.APIVersion) (service domain.ReverseServiceI) {
	switch api {
	case 0:
		service = &V0Services{
			UsersModel: &UsersV0{
				AppspaceUsersV0: x.AppspaceUsersV0,
				appspaceID:      appspace.AppspaceID},
			AppspaceDB: x.V0AppspaceDB.GetService(appspace),
		}
	}
	return
}
