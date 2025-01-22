package events

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

// Relations returns related identifiers to the passed identifier.
type Relations struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
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
