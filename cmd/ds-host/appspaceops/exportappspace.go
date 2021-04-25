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

type ExportAppspace struct {
	Config        *domain.RuntimeConfig
	AppspaceModel interface {
		GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, error)
	}
	AppspaceStatus interface {
		WaitTempPaused(appspaceID domain.AppspaceID, reason string) chan struct{}
		IsTempPaused(appspaceID domain.AppspaceID) bool
		LockClosed(appspaceID domain.AppspaceID) (chan struct{}, bool)
	}
	AppspaceMetaDB interface {
		CloseConn(appspaceID domain.AppspaceID) error
	}
	AppspaceDB interface {
		CloseAppspace(appspaceID domain.AppspaceID)
	}
	AppspaceLogger interface {
		Log(appspaceID domain.AppspaceID, source string, message string)
		EjectLogger(appspaceID domain.AppspaceID)
	}
}

// Export everything that an appspace might need to be re-created somewhere.
// Useful for making backups and export / transfers of appspace
// Export fetches its own pause.
func (e *ExportAppspace) Export(appspaceID domain.AppspaceID) (string, error) {
	// obtain temp pause
	pauseCh := e.AppspaceStatus.WaitTempPaused(appspaceID, "export")
	defer close(pauseCh)

	e.AppspaceLogger.Log(appspaceID, "ds-host", "exporting appspace data")

	zipFile, err := e.createZip(appspaceID)

	return zipFile, err
}

// Backup creates an export zip file
// Backup expects that the appspace is already paused
func (e *ExportAppspace) Backup(appspaceID domain.AppspaceID) (string, error) {
	if !e.AppspaceStatus.IsTempPaused(appspaceID) {
		return "", errors.New("appspace is not temp paused as expected")
	}

	e.AppspaceLogger.Log(appspaceID, "ds-host", "backing up appspace data")

	zipFile, err := e.createZip(appspaceID)

	return zipFile, err
}

func (e *ExportAppspace) RestoreBackup(appspaceID domain.AppspaceID, zipFile string) error {
	if !e.AppspaceStatus.IsTempPaused(appspaceID) {
		return errors.New("appspace is not temp paused as expected")
	}

	e.AppspaceLogger.Log(appspaceID, "ds-host", "restoring backup appspace data")

	err := e.restoreZip(appspaceID, zipFile)
	if err != nil {
		return err
	}
	return nil
}

// closeAll closes appspace files that might remain open after an appspace has been paused/stopped.
func (e *ExportAppspace) closeAll(appspaceID domain.AppspaceID) error {
	// appspace meta
	// appspace dbs
	// appspace logs

	err := e.AppspaceMetaDB.CloseConn(appspaceID)
	if err != nil {
		return err
	}

	e.AppspaceDB.CloseAppspace(appspaceID)

	e.AppspaceLogger.EjectLogger(appspaceID)

	return nil
}

func (e *ExportAppspace) getZipFilename(dir string) (string, error) {
	zipFile := filepath.Join(dir, time.Now().Format(fileDateFormat))
	increment := 0
	incStr := ""
	for {
		_, err := os.Stat(zipFile + incStr + ".zip")
		if os.IsNotExist(err) {
			break
		} else if err != nil {
			e.getLogger("getZipFilename, os.Stat").Error(err)
			return "", err
		}
		increment++
		incStr = fmt.Sprintf("_%d", increment)
	}
	return zipFile + incStr + ".zip", nil
}

func (e *ExportAppspace) createZip(appspaceID domain.AppspaceID) (string, error) {

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

	locationDir := filepath.Join(e.Config.Exec.AppspacesPath, appspace.LocationKey)
	dataDir := filepath.Join(locationDir, "data")
	zipFile, err := e.getZipFilename(filepath.Join(locationDir, "backups"))
	if err != nil {
		return "", err
	}

	err = zipfns.Zip(dataDir, zipFile)
	if err != nil {
		e.getLogger("Export, zipFiles").Error(err)
		return "", err
	}

	return zipFile, nil
}

func (e *ExportAppspace) restoreZip(appspaceID domain.AppspaceID, zipFile string) error {
	// get closed lock
	closedCh, ok := e.AppspaceStatus.LockClosed(appspaceID)
	if !ok {
		return errors.New("failed to get lock closed")
	}
	defer close(closedCh)

	// close all the things
	err := e.closeAll(appspaceID)
	if err != nil {
		return err
	}

	appspace, err := e.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		e.getLogger("restoreZip, get appspace").Error(err)
		return err
	}

	locationDir := filepath.Join(e.Config.Exec.AppspacesPath, appspace.LocationKey)
	dataDir := filepath.Join(locationDir, "data")

	err = os.RemoveAll(dataDir)
	if err != nil {
		e.getLogger("restoreZip, os.RemoveAll").Error(err)
		return err
	}

	//

	err = zipfns.Unzip(zipFile, dataDir)
	if err != nil {
		e.getLogger("restoreZip, Unzip").Error(err)
		return err
	}

	return nil
}

func (e *ExportAppspace) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("ExportAppspace")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
