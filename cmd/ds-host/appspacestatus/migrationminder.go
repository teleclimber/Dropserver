package appspacestatus

import (
	"github.com/blang/semver/v4"
	"github.com/teleclimber/DropServer/cmd/ds-host/appops"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// TODO: instead of loading everything from the DB, cache migrations and listen to events from app model to update.

// MigrationMinder determines if there are potential migrations available for appspaces
// It will start the migrations (create a migration job) when we implement auto-upadte for appspaces
// It can return migration potential data for all or select appspaces much like a model
// Does it use events or polling?
type MigrationMinder struct {
	AppModel interface {
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
		GetAppUrlListing(domain.AppID) (domain.AppListing, domain.AppURLData, error)
	} `checkinject:"required"`
}

// GetForAppspace looks at installed versions and remote listing versions
// and returns the latest available version for the appspace.
func (m *MigrationMinder) GetForAppspace(appspace domain.Appspace) (domain.Version, bool, error) {
	var latest domain.Version
	isRemote := false

	cmpSemver, err := semver.Parse(string(appspace.AppVersion))
	if err != nil {
		m.getLogger("GetForAppspace() Parse current").Error(err)
		return domain.Version(""), isRemote, err
	}

	versions, err := m.AppModel.GetVersionsForApp(appspace.AppID)
	if err != nil {
		return domain.Version(""), isRemote, err
	}
	if len(versions) != 0 {
		li := *versions[len(versions)-1]
		latestInstalled, err := semver.Parse(string(li.Version))
		if err != nil {
			m.getLogger("GetForAppspace() Parse latest installed").Error(err)
			return domain.Version(""), isRemote, err
		}
		if latestInstalled.GT(cmpSemver) {
			cmpSemver = latestInstalled
			latest = li.Version
		}
	}

	// Now get remote listing versions, if any, and see if there is anything new
	listing, _, err := m.AppModel.GetAppUrlListing(appspace.AppID)
	if err == domain.ErrNoRowsInResultSet {
		// no-op
	} else if err != nil {
		return domain.Version(""), isRemote, err
	} else {
		remote, err := appops.GetLatestVersion(listing.Versions) // TODO remove dependence on appops!
		if err != nil {
			m.getLogger("GetForAppspace() GetLatestVersion").Error(err)
			return domain.Version(""), isRemote, err
		}
		latestRemote, err := semver.Parse(string(remote))
		if err != nil {
			m.getLogger("GetForAppspace() parse latest remote").Error(err)
			return domain.Version(""), isRemote, err
		}
		if latestRemote.GT(cmpSemver) {
			latest = remote
			isRemote = true
		}
	}

	return latest, isRemote, nil
}

func (m *MigrationMinder) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("MigrationMinder")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
