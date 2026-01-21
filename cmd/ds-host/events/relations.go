package events

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// Relations returns related identifiers to the passed identifier.
type Relations struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	ManageUsers interface {
		UsersForAppspace(appspaceID domain.AppspaceID) (map[domain.UserID]domain.UserIDProxyIDConflicts, error)
	} `checkinject:"required"`
}

func (r *Relations) GetAppspaceOwnerID(appspaceID domain.AppspaceID) (domain.UserID, bool) {
	// At some point it may be worth it to cache these relations
	// For now they are just looked up in the DB each time
	a, err := r.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		return domain.UserID(0), false
	}
	return a.OwnerID, true
}

// GetAppspaceUserIDs returns all instance user ids
// that are users or owners of the appspace
func (r *Relations) GetAppspaceUserIDs(appspaceID domain.AppspaceID) []domain.UserID {
	userConflicts, err := r.ManageUsers.UsersForAppspace(appspaceID)
	if err != nil {
		return nil
	}
	ret := make([]domain.UserID, len(userConflicts))
	for userID := range userConflicts {
		ret = append(ret, userID)
	}
	ownerID, ok := r.GetAppspaceOwnerID(appspaceID)
	if ok {
		found := false
		for _, u := range ret {
			if u == ownerID {
				found = true
			}
		}
		if !found {
			ret = append(ret, ownerID)
		}
	}
	return ret
}
