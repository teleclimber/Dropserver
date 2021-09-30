package appspaceops

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
	"github.com/teleclimber/DropServer/internal/zipfns"
)

type tokenData struct {
	tempZip     string
	tempDir     string
	timer       *time.Timer
	cancelTimer chan struct{}
}

// RestoreBackup replaces an appspace's data files
type RestoreAppspace struct {
	Config    *domain.RuntimeConfig `checkinject:"required"`
	InfoModel interface {
		GetAppspaceMetaInfo(dataPath string) (domain.AppspaceMetaInfo, error)
	} `checkinject:"required"`
	AppspaceModel interface {
		GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceFilesModel interface {
		CheckDataFiles(dataDir string) error
		ReplaceData(appspace domain.Appspace, source string) error
	} `checkinject:"required"`
	AppspaceStatus interface {
		WaitTempPaused(appspaceID domain.AppspaceID, reason string) chan struct{}
		IsTempPaused(appspaceID domain.AppspaceID) bool
		LockClosed(appspaceID domain.AppspaceID) (chan struct{}, bool)
	} `checkinject:"required"`
	AppspaceMetaDB interface {
		CloseConn(appspaceID domain.AppspaceID) error
	} `checkinject:"required"`
	AppspaceDB interface {
		CloseAppspace(appspaceID domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceLogger interface {
		Log(appspaceID domain.AppspaceID, source string, message string)
		EjectLogger(appspaceID domain.AppspaceID)
	} `checkinject:"required"`

	tokensMux sync.Mutex
	tokens    map[string]tokenData
}

func (r *RestoreAppspace) Init() {
	r.tokens = make(map[string]tokenData)
}

// Prepare an io.Reader pointing to a zip file
// to restore to an appspace
func (r *RestoreAppspace) Prepare(reader io.Reader) (string, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "ds-temp-*")
	if err != nil {
		r.getLogger("Prepare, os.MkdirTemp()").Error(err)
		return "", err //internal error
	}

	tok := r.newToken()

	r.tokensMux.Lock()
	tokData := r.tokens[tok]
	tokData.tempZip = dir
	r.tokens[tok] = tokData
	r.tokensMux.Unlock()

	zipFile := filepath.Join(dir, "restore.zip")
	f, err := os.Create(zipFile)
	if err != nil {
		f.Close()
		return "", err //internal error
	}
	_, err = io.Copy(f, reader)
	f.Close() // cant' use defer because unzip will need to open file
	if err != nil {
		return "", err //internal error
	}

	err = r.unzipFile(tok, zipFile)
	if err != nil {
		return "", err
	}

	return tok, nil
}

// PrepareBackup prepares an appspace's existing backup to restore
func (r *RestoreAppspace) PrepareBackup(appspaceID domain.AppspaceID, backupFile string) (string, error) {
	tok := r.newToken()

	// need to get apppsace, so we can get location key
	appspace, err := r.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		r.getLogger("PrepareBackup, get appspace").Error(err)
		return "", err
	}

	err = validator.AppspaceBackupFile(backupFile)
	if err != nil {
		r.getLogger("PrepareBackup, validator.AppspaceBackupFile").Error(err)
		return "", err
	}
	zipFile := filepath.Join(r.Config.Exec.AppspacesPath, appspace.LocationKey, "backups", backupFile)

	err = r.unzipFile(tok, zipFile)
	if err != nil {
		return "", err
	}

	return tok, nil
}

// unzipFile takes a file (assumed zip) and unzips it
// into a temporary directory and verifies its contents
// it returns a token that can be used to commit the restore
func (r *RestoreAppspace) unzipFile(tok string, filePath string) error {
	dir, err := os.MkdirTemp(os.TempDir(), "ds-temp-*")
	if err != nil {
		r.getLogger("unzipFile, os.MkdirTemp()").Error(err)
		return err //internal error
	}

	r.tokensMux.Lock()
	tokData, ok := r.tokens[tok]
	if !ok {
		r.tokensMux.Unlock()
		return domain.ErrTokenNotFound
	}
	tokData.tempDir = dir
	r.tokens[tok] = tokData
	r.tokensMux.Unlock()

	// then unzip
	err = zipfns.Unzip(filePath, dir)
	if err != nil {
		r.getLogger("unzipFile, zipfns.Unzip()").Error(err)
		// error unzipping. Very possibly a bad input zip
		// Could be other things, like no space left.
		return fmt.Errorf("input error: %w", err)
	}

	return nil
}

// CheckAppspaceDataValid does basic checking of uploaded data
func (r *RestoreAppspace) CheckAppspaceDataValid(tok string) error {
	r.tokensMux.Lock()
	defer r.tokensMux.Unlock()
	tokData, ok := r.tokens[tok]
	if !ok {
		return domain.ErrTokenNotFound
	}
	err := r.AppspaceFilesModel.CheckDataFiles(tokData.tempDir)
	return err
}

// GetMetaInfo gets the info stored in appspace data
// Any error can be assumed to be the result of bad input
func (r *RestoreAppspace) GetMetaInfo(tok string) (domain.AppspaceMetaInfo, error) {
	r.tokensMux.Lock()
	defer r.tokensMux.Unlock()
	tokData, ok := r.tokens[tok]
	if !ok {
		return domain.AppspaceMetaInfo{}, domain.ErrTokenNotFound
	}

	// after zip, read appspace meta data
	metaInfo, err := r.InfoModel.GetAppspaceMetaInfo(tokData.tempDir)
	if err != nil {
		// likely a bad DB, or badly named, or something....
		// report that back to user
		return domain.AppspaceMetaInfo{}, fmt.Errorf("input error: failed to get appspace meta data: %w", err)
	}

	return metaInfo, nil
}

// ReplaceData stops the appspace and replaces the data files
func (r *RestoreAppspace) ReplaceData(tok string, appspaceID domain.AppspaceID) error {
	closedCh, ok := r.AppspaceStatus.LockClosed(appspaceID)
	if !ok {
		return errors.New("failed to get lock closed")
	}
	defer close(closedCh)

	// close all the things
	err := r.closeAll(appspaceID)
	if err != nil {
		return err
	}

	appspace, err := r.AppspaceModel.GetFromID(appspaceID)
	if err != nil {
		r.getLogger("ReplaceData, get appspace").Error(err)
		return err
	}

	// do we also load the app version to check that schema is compatible?
	// -> no, appspace status will catch that. So user will have a chance to fix.
	//   ..also UI should warn about this before committing

	defer r.delete(tok)

	r.tokensMux.Lock()
	defer r.tokensMux.Unlock()

	tokData, ok := r.tokens[tok]
	if !ok {
		// probably a sentinel error to say that the token is no longer valid
		return domain.ErrTokenNotFound
	}

	err = r.AppspaceFilesModel.ReplaceData(*appspace, tokData.tempDir)
	if err != nil {
		return err
	}

	return nil
}

// closeAll closes appspace files that might remain open after an appspace has been paused/stopped.
func (r *RestoreAppspace) closeAll(appspaceID domain.AppspaceID) error {
	// appspace meta
	// appspace dbs
	// appspace logs

	// this should really be in status or something?

	err := r.AppspaceMetaDB.CloseConn(appspaceID)
	if err != nil {
		return err
	}

	r.AppspaceDB.CloseAppspace(appspaceID)

	r.AppspaceLogger.EjectLogger(appspaceID)

	return nil
}

func (r *RestoreAppspace) newToken() string {
	r.tokensMux.Lock()
	defer r.tokensMux.Unlock()
	var tok string
	for {
		tok = randomToken()
		_, found := r.tokens[tok]
		if !found {
			break
		}
	}

	t := time.NewTimer(15 * time.Minute)
	ct := make(chan struct{})
	r.tokens[tok] = tokenData{
		timer:       t,
		cancelTimer: ct,
	}

	go func() {
		select {
		case <-t.C:
			r.delete(tok)
		case <-ct:
		}
	}()

	return tok
}

func (r *RestoreAppspace) delete(tok string) (err error) {
	r.tokensMux.Lock()
	defer r.tokensMux.Unlock()
	tokData, ok := r.tokens[tok]
	if !ok {
		return nil
	}

	delete(r.tokens, tok)
	tokData.timer.Stop()
	tokData.cancelTimer <- struct{}{}

	// Remove both and return one error after
	if tokData.tempZip != "" {
		errZ := os.RemoveAll(tokData.tempZip)
		if errZ != nil {
			r.getLogger("delete token, os.RemoveAll(tempZip)").Error(errZ)
			err = errZ
		}
	}
	errD := os.RemoveAll(tokData.tempDir)
	if errD != nil {
		r.getLogger("delete token, os.RemoveAll(tempDir)").Error(errD)
		err = errD
	}
	return
}

func (r *RestoreAppspace) DeleteAll() {
	for tok := range r.tokens {
		r.delete(tok)
	}
}

func (r *RestoreAppspace) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("RestoreAppspace")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

// probably need a DeleteAll so that temp stuff is not preserved between restarts?

////////////
// random string
const chars36 = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand3 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomToken() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars36[seededRand3.Intn(len(chars36))]
	}
	return string(b)
}
