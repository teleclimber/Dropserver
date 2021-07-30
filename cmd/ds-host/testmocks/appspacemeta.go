package testmocks

import (
	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=appspacemeta_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks AppspaceMetaDB,AppspaceInfoModel,UsersV0,AppspaceDB,AppspaceLogger

type AppspaceMetaDB interface {
	Create(domain.AppspaceID, int) error
	GetHandle(domain.AppspaceID) (*sqlx.DB, error)
	CloseConn(domain.AppspaceID) error
}

// AppspaceInfoModel caches and dishes AppspaceInfoModels
type AppspaceInfoModel interface {
	GetSchema(domain.AppspaceID) (int, error)
	SetSchema(domain.AppspaceID, int) error
}

type UsersV0 interface {
	Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
	UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, permissions []string) error
	Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
	GetByDropID(appspaceID domain.AppspaceID, dropID string) (domain.AppspaceUser, error)
	GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
	Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error
}

type AppspaceDB interface {
	CloseAppspace(appspaceID domain.AppspaceID)
}

type AppspaceLogger interface {
	Log(appspaceID domain.AppspaceID, source string, message string)
	EjectLogger(appspaceID domain.AppspaceID)
}
