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
	MigrationJobModel interface {
		DeleteForAppspace(domain.AppspaceID) error
	} `checkinject:"required"`
	SandboxManager interface {
		StopAppspace(domain.AppspaceID)
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

	// This is where I'd like to be able to pass a transaction around
	// so we can do all these deletions ..or none at all.

	// Delete from cookies table?

	err := d.MigrationJobModel.DeleteForAppspace(appspace.AppspaceID)
	if err != nil {
		return err
	}

	err = d.AppspaceModel.Delete(appspace.AppspaceID)
	if err != nil {
		return err
	}

	// then delete the files
	err = d.AppspaceFilesModel.DeleteLocation(appspace.LocationKey)
	if err != nil {
		return err
	}

	return nil
}
