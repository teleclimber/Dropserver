package appspaceops

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestDelete(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	r := &RestoreAppspace{}
	r.Init()

	tok := "abc"
	timer := time.NewTimer(15 * time.Minute)
	cancelTimer := make(chan struct{})
	r.tokens[tok] = tokenData{
		tempZip:     "",
		tempDir:     dir,
		timer:       timer,
		cancelTimer: cancelTimer,
	}

	go func() {
		r.delete(tok)
		_, err = os.Stat(dir)
		if !os.IsNotExist(err) {
			t.Error("expected no exist error")
		}
	}()

	<-cancelTimer
}

func TestReplaceData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	asID := domain.AppspaceID(7)
	loc := "abc"
	tempDir := "/uvw/xyz/"
	appspace := domain.Appspace{AppspaceID: asID, LocationKey: loc}

	closedChan := make(chan struct{})

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().LockClosed(asID).Return(closedChan, true)

	appspaceMetaDB := testmocks.NewMockAppspaceMetaDB(mockCtrl)
	appspaceMetaDB.EXPECT().CloseConn(asID).Return(nil)

	appspaceDB := testmocks.NewMockAppspaceDB(mockCtrl)
	appspaceDB.EXPECT().CloseAppspace(asID)

	appspaceLogger := testmocks.NewMockAppspaceLogger(mockCtrl)
	appspaceLogger.EXPECT().Close(asID)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(asID).Return(&appspace, nil)

	appspaceFilesModels := testmocks.NewMockAppspaceFilesModel(mockCtrl)
	appspaceFilesModels.EXPECT().ReplaceData(appspace, tempDir).Return(nil)

	r := &RestoreAppspace{
		AppspaceStatus:     appspaceStatus,
		AppspaceMetaDB:     appspaceMetaDB,
		AppspaceDB:         appspaceDB,
		AppspaceLogger:     appspaceLogger,
		AppspaceModel:      appspaceModel,
		AppspaceFilesModel: appspaceFilesModels,
	}
	r.Init()

	tok := "abc"
	timer := time.NewTimer(15 * time.Minute)
	cancelTimer := make(chan struct{})
	r.tokens[tok] = tokenData{
		tempZip:     "",
		tempDir:     tempDir,
		timer:       timer,
		cancelTimer: cancelTimer,
	}

	go func() {
		err := r.ReplaceData(tok, asID)
		if err != nil {
			t.Error(err)
		}
	}()

	<-cancelTimer

}
