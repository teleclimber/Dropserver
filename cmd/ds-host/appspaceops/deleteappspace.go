package appspaceops

import (
	"errors"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type DeleteAppspace struct {
	AppspaceStatus interface {
		WaitTempPaused(appspaceID domain.AppspaceID, reason string) chan struct{}
		LockClosed(appspaceID domain.AppspaceID) (chan struct{}, bool)
	} `checkinject:"required"`
	AppspaceModel interface {
		Delete(domain.AppspaceID) error
	} `checkinject:"required"`
	AppspaceFilesModel interface {
		DeleteLocation(string) error
	} `checkinject:"required"`
	AppspaceTSNetModel interface {
		Delete(appspaceID domain.AppspaceID) error
	} `checkinject:"required"`
	DomainController interface {
		StopManaging(string)
	} `checkinject:"required"`
	MigrationJobModel interface {
		DeleteForAppspace(domain.AppspaceID) error
	} `checkinject:"required"`
	SandboxManager interface {
		StopAppspace(domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceLogger interface {
		Forget(domain.AppspaceID)
	} `checkinject:"required"`
}

// Delete permanently deletes all data associated with an appspace
func (d *DeleteAppspace) Delete(appspace domain.Appspace) error {
	pauseCh := d.AppspaceStatus.WaitTempPaused(appspace.AppspaceID, "delete")
	defer close(pauseCh)

	closedCh, ok := d.AppspaceStatus.LockClosed(appspace.AppspaceID)
	if !ok {
		return errors.New("failed to get lock closed")
	}
	defer close(closedCh)

	d.SandboxManager.StopAppspace(appspace.AppspaceID)

	d.AppspaceTSNetModel.Delete(appspace.AppspaceID)

	// Delete from cookies table?

	err := d.MigrationJobModel.DeleteForAppspace(appspace.AppspaceID)
	if err != nil {
		return err
	}

	err = d.AppspaceModel.Delete(appspace.AppspaceID)
	if err != nil {
		return err
	}

	d.AppspaceLogger.Forget(appspace.AppspaceID)

	// then delete the files
	err = d.AppspaceFilesModel.DeleteLocation(appspace.LocationKey)
	if err != nil {
		return err
	}

	d.DomainController.StopManaging(appspace.DomainName)

	return nil
}
