package testmocks

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// all v0 versioned interfaces

//go:generate mockgen -destination=appspacemeta_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks UsersV0

type UsersV0 interface {
	Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
	UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, permissions []string) error
	Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
	GetByDropID(appspaceID domain.AppspaceID, dropID string) (domain.AppspaceUser, error)
	GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error
}
