package appops

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type DeleteApp struct {
	AppFilesModel interface {
		Delete(string) error
	} `checkinject:"required"`
	AppModel interface {
		GetVersionsForApp(appID domain.AppID) ([]*domain.AppVersion, error)
		GetVersion(appID domain.AppID, version domain.Version) (domain.AppVersion, error)
		Delete(appID domain.AppID) error
		DeleteVersion(domain.AppID, domain.Version) error
	} `checkinject:"required"`
	AppspaceModel interface {
		GetForApp(appID domain.AppID) ([]*domain.Appspace, error)
		GetForAppVersion(appID domain.AppID, version domain.Version) ([]*domain.Appspace, error)
	} `checkinject:"required"`
	AppLogger interface {
		Forget(string)
	} `checkinject:"required"`
}

func (d *DeleteApp) Delete(appID domain.AppID) error {
	appspaces, err := d.AppspaceModel.GetForApp(appID)
	if err != nil {
		return err
	}
	if len(appspaces) != 0 {
		return domain.ErrAppVersionInUse
	}

	versions, err := d.AppModel.GetVersionsForApp(appID)
	if err != nil {
		return err
	}
	for _, appVersion := range versions {
		d.AppLogger.Forget(appVersion.LocationKey)

		err = d.AppFilesModel.Delete(appVersion.LocationKey)
		if err != nil {
			return err
		}
		err = d.AppModel.DeleteVersion(appID, appVersion.Version)
		if err != nil {
			return err
		}
	}

	err = d.AppModel.Delete(appID)
	if err != nil {
		return err
	}

	return nil
}

// DeleteVersion removes the app version code and db entry from the system.
func (d *DeleteApp) DeleteVersion(appID domain.AppID, version domain.Version) error {
	appspaces, err := d.AppspaceModel.GetForAppVersion(appID, version)
	if err != nil {
		return err
	}
	if len(appspaces) != 0 {
		return domain.ErrAppVersionInUse
	}
	appVersion, err := d.AppModel.GetVersion(appID, version)
	if err != nil {
		return err
	}

	d.AppLogger.Forget(appVersion.LocationKey)

	err = d.AppFilesModel.Delete(appVersion.LocationKey)
	if err != nil {
		return err
	}
	err = d.AppModel.DeleteVersion(appID, version)
	if err != nil {
		return err
	}
	return nil
}
