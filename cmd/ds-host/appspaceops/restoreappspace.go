package appspaceops

import (
	"errors"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/zipfns"
)

type tokenData struct {
	filePath    string
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

// PrepareFile takes a file (assumed zip) and unzips it
// into a temporary directory and verifies its contents
// it returns a token that can be used to commit the restore
// It should also return a basic struct describing what is known about the appspace
func (r *RestoreAppspace) PrepareFile(appspaceID domain.AppspaceID, filePath string) (string, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "ds-temp-*")
	if err != nil {
		r.getLogger("PrepareFile, os.MkdirTemp()").Error(err)
		return "", err //internal error
	}

	// then unzip
	err = zipfns.Unzip(filePath, dir)
	if err != nil {
		os.RemoveAll(dir)
		r.getLogger("PrepareFile, zipfns.Unzip()").Error(err)
		// error unzipping. Very possibly a bad input zip
		// Could be other things, like no space left.
		return "", err // input error
	}

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
		filePath:    filePath,
		tempDir:     dir,
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

	return tok, nil
}

// Probably need a basic check? That dirs are laid out as expected
// maybe check for existence of symlinks or whatever
// check on size (and report)
//

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
		return domain.AppspaceMetaInfo{}, errors.New("failed to get appspace meta data")
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

func (r *RestoreAppspace) delete(tok string) error {
	r.tokensMux.Lock()
	defer r.tokensMux.Unlock()
	tokData, ok := r.tokens[tok]
	if !ok {
		return nil
	}

	delete(r.tokens, tok)
	tokData.timer.Stop()
	tokData.cancelTimer <- struct{}{}

	err := os.RemoveAll(tokData.tempDir)
	if err != nil {
		r.getLogger("delete token, os.RemoveAll").Error(err)
		return err
	}
	return nil
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
