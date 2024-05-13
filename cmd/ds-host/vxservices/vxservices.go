package vxservices

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

type serviceGetter interface {
	GetService(appspace *domain.Appspace) domain.ReverseServiceI
}

// VXServices holds the structs necessary to create a service for any api version
type VXServices struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	}
}

// Get returns a reverse service for the appspace
func (x *VXServices) Get(appspace *domain.Appspace, api domain.APIVersion) (service domain.ReverseServiceI) {
	switch api {
	case 0:
		service = &V0Services{ // yes, there is a v0 services, to match a v0 API that the app is using.
			UsersModel: &UsersService{
				AppspaceUserModel: x.AppspaceUserModel,
				appspaceID:        appspace.AppspaceID},
		}
	}
	return
}
