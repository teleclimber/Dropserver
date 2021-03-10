package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/twine"
)

//go:generate mockgen -destination=vxapi_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks VXUserModels,V0UserModel

type VXUserModels interface {
	GetV0(appspaceID domain.AppspaceID) domain.V0UserModel
}

type V0UserModel interface {
	HandleMessage(message twine.ReceivedMessageI)
	Create(proxyID domain.ProxyID, displayName string, permissions []string) error
	Update(proxyID domain.ProxyID, displayName string, permissions []string) error
	Delete(proxyID domain.ProxyID) error
	Get(proxyID domain.ProxyID) (domain.V0User, error)
	GetAll() ([]domain.V0User, error)
}
