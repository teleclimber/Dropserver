package testmocks

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//go:generate mockgen -destination=appspaceops_mocks.go -package=testmocks -self_package=github.com/teleclimber/DropServer/cmd/ds-host/testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks ManageUsers,AppspaceUsersCache,DeleteAppspace

type ManageUsers interface {
	GetProxyIDForUserID(appspaceID domain.AppspaceID, userID domain.UserID) (domain.ProxyID, error)
	GetConflictsForUserID(appspaceID domain.AppspaceID, userID domain.UserID) (domain.UserIDProxyIDConflicts, error)
	AppspacesForUser(userID domain.UserID) (map[domain.AppspaceID]domain.UserIDProxyIDConflicts, error)
	UserInAppspace(userID domain.UserID, auths []domain.AppspaceUserAuthBare, appspaceID domain.AppspaceID) (domain.UserIDProxyIDConflicts, error)
	ConflictsForAppspace(appspaceID domain.AppspaceID) (map[domain.UserProxyTuple]domain.UserIDProxyIDConflicts, error)
}

type AppspaceUsersCache interface {
	AppspacesForUser(userID domain.UserID) (map[domain.AppspaceID]domain.UserIDProxyIDConflicts, error)
	UsersForAppspace(appspaceID domain.AppspaceID) (map[domain.UserID]domain.UserIDProxyIDConflicts, error)
	ProxyIDsForAppspace(appspaceID domain.AppspaceID) (map[domain.ProxyID]domain.UserIDProxyIDConflicts, error)
}

type DeleteAppspace interface {
	Delete(domain.Appspace) error
}
