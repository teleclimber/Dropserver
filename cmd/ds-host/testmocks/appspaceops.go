package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=appspaceops_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks ManageUsers,DeleteAppspace

type ManageUsers interface {
	InstanceUser(appspaceID domain.AppspaceID, userID domain.UserID) (domain.ProxyID, error)
	AppspacesForUser(userID domain.UserID) ([]domain.AppspaceUserIDs, error)
}

type DeleteAppspace interface {
	Delete(domain.Appspace) error
}
