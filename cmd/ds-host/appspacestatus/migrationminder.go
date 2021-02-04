package appspacestatus

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// TODO: instead of loading everything from the DB, cache migrations and listen to events from app model to update.

// MigrationMinder determines if there are potential migrations available for appspaces
// It will start the migrations (create a migration job) when we implement auto-upadte for appspaces
// It can return migration potential data for all or select appspaces much like a model
// Does it use events or polling?
type MigrationMinder struct {
	AppModel interface {
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
	}
	AppspaceModel interface {
		GetForOwner(domain.UserID) ([]*domain.Appspace, error)
	}
}

// GetAllForOwner checks each appspace for potential migration available
// Rerutns a map containing newest app version for appspaces that have one
func (m *MigrationMinder) GetAllForOwner(ownerID domain.UserID) (map[domain.AppspaceID]domain.AppVersion, error) {
	appspaces, err := m.AppspaceModel.GetForOwner(ownerID)
	if err != nil {
		return nil, err
	}

	ret := make(map[domain.AppspaceID]domain.AppVersion)
	for _, appspace := range appspaces {
		appVersion, ok, err := m.GetForAppspace(*appspace)
		if err != nil {
			return nil, err
		}
		if ok {
			ret[appspace.AppspaceID] = appVersion
		}
	}
	return ret, nil
}

// GetForAppspace returns the latest app version for an appspace
// OK is false if appspace is on the latest version
func (m *MigrationMinder) GetForAppspace(appspace domain.Appspace) (domain.AppVersion, bool, error) {
	var latest domain.AppVersion

	versions, err := m.AppModel.GetVersionsForApp(appspace.AppID)
	if err != nil {
		return latest, false, err
	}

	latest = *versions[len(versions)-1]
	if appspace.AppVersion != latest.Version {
		// migration possible.
		return latest, true, nil
	}
	return latest, false, nil
}
