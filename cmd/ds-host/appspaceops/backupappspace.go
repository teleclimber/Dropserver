package appspaceops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/zipfns"
)

// Do we have a db table of backups, or do we just have zip files in a dedicated folder?
// -> definitely files. Probably zipped for space and ease of organization.
// Maybe they could all live in the same location?

var fileDateFormat = "2006-01-02_1504"

type BackupAppspace struct {
	AppspaceModel interface {
		GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	SandboxManager interface {
		StopAppspace(domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceStatus interface {
		WaitTempPaused(appspaceID domain.AppspaceID, reason string) chan struct{}
		IsTempPaused(appspaceID domain.AppspaceID) bool
		LockClosed(appspaceID domain.AppspaceID) (chan struct{}, bool)
	} `checkinject:"required"`
	AppspaceMetaDB interface {
		CloseConn(appspaceID domain.AppspaceID) error
	} `checkinject:"required"`
	AppspaceLogger interface {
		Log(appspaceID domain.AppspaceID, source string, message string)
		Close(appspaceID domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceLocation2Path interface {
		Data(string) string
		Backups(string) string
		Backup(string, string) string
	} `checkinject:"required"`
}

// CreateBackup everything that an appspace might need to be re-created somewhere.
// Useful for making backups and export / transfers of appspace
// CreateBackup fetches its own pause.
func (e *BackupAppspace) CreateBackup(appspaceID domain.AppspaceID) (string, error) {
	// obtain temp pause
	pauseCh := e.AppspaceStatus.WaitTempPaused(appspaceID, "backup")
	defer close(pauseCh)

	e.AppspaceLogger.Log(appspaceID, "ds-host", "backing up appspace data")

	zipFile, err := e.createZip(appspaceID)

	return zipFile, err
}

// BackupNoPause creates a backup zip file like CreateBackup
// but it expects the appspace to be already paused
func (e *BackupAppspace) BackupNoPause(appspaceID domain.AppspaceID) (string, error) {
	if !e.AppspaceStatus.IsTempPaused(appspaceID) {
		return "", errors.New("appspace is not temp paused as expected")
	}

	e.AppspaceLogger.Log(appspaceID, "ds-host", "backing up appspace data")

	zipFile, err := e.createZip(appspaceID)

	return zipFile, err
}

// closeAll closes appspace files that might remain open after an appspace has been paused/stopped.
func (e *BackupAppspace) closeAll(appspaceID domain.AppspaceID) error {
	// appspace meta
	// appspace dbs
	// appspace logs

	e.SandboxManager.StopAppspace(appspaceID)

	err := e.AppspaceMetaDB.CloseConn(appspaceID)
	if err != nil {
		return err
	}

	e.AppspaceLogger.Close(appspaceID)

	return nil
}

func (e *BackupAppspace) getZipFilename(dir string) (string, error) {
	dateStr := time.Now().Format(fileDateFormat)
	increment := 0
	incStr := ""
	for {
		_, err := os.Stat(filepath.Join(dir, dateStr+incStr+".zip"))
		if os.IsNotExist(err) {
			break
		} else if err != nil {
			e.getLogger("getZipFilename, os.Stat").Error(err)
			return "", err
		}
		increment++
		incStr = fmt.Sprintf("_%d", increment)
	}

	return dateStr + incStr + ".zip", nil
}

func (e *BackupAppspace) createZip(appspaceID domain.AppspaceID) (string, error) {

	// generate additional data and save to data dir
	// (big json meta data and users list)

	// get closed lock
	closedCh, ok := e.AppspaceStatus.LockClosed(appspaceID)
	if !ok {
		return "", errors.New("failed to get lock closed")
	}
	defer close(closedCh)

	// close all the things
	err := e.closeAll(appspaceID)
	if err != nil {
		return "", err
	}

	// copy files / zip them...
	appspace, err := e.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		e.getLogger("createZip, get appspace").Error(err)
		return "", err
	}

	dataDir := e.AppspaceLocation2Path.Data(appspace.LocationKey)
	backupsDir := e.AppspaceLocation2Path.Backups(appspace.LocationKey)
	zipFile, err := e.getZipFilename(backupsDir)
	if err != nil {
		return "", err
	}

	err = zipfns.Zip(dataDir, e.AppspaceLocation2Path.Backup(appspace.LocationKey, zipFile))
	if err != nil {
		e.getLogger("createZip, zipfns.Zip").Error(err)
		return "", err
	}

	return zipFile, nil
}

func (e *BackupAppspace) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("BackupAppspace")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
