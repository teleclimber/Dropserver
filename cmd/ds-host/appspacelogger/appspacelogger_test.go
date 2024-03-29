package appspacelogger

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestLog(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = os.MkdirAll(filepath.Join(dir, "loc", "data", "logs"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	appspaceID := domain.AppspaceID(7)

	config := &domain.RuntimeConfig{}
	config.Exec.AppspacesPath = dir

	am := testmocks.NewMockAppspaceModel(mockCtrl)
	am.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{LocationKey: "loc"}, nil)
	as := testmocks.NewMockAppspaceStatus(mockCtrl)
	as.EXPECT().IsLockedClosed(appspaceID).Return(false)

	l := AppspaceLogger{
		AppspaceModel:  am,
		AppspaceStatus: as,
		Config:         config}
	l.Init()

	l.Log(appspaceID, "test-source", "log message")
	defer l.Close(appspaceID)

	logFile := filepath.Join(dir, "loc", "data", "logs", "log.txt")
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), ` test-source log message`) {
		t.Error("bad content: " + string(content))
	}
}

func TestGetLogger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = os.MkdirAll(filepath.Join(dir, "loc", "data", "logs"), 0700)
	if err != nil {
		t.Fatal(err)
	}

	appspaceID := domain.AppspaceID(7)

	config := &domain.RuntimeConfig{}
	config.Exec.AppspacesPath = dir

	am := testmocks.NewMockAppspaceModel(mockCtrl)
	am.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{LocationKey: "loc"}, nil)
	as := testmocks.NewMockAppspaceStatus(mockCtrl)
	as.EXPECT().IsLockedClosed(appspaceID).Return(false)

	l := AppspaceLogger{
		AppspaceModel:  am,
		AppspaceStatus: as,
		Config:         config}
	l.Init()

	logger := l.Open(appspaceID)
	logger.Log("test-source", "log message")
	defer l.Close(appspaceID)

	logFile := filepath.Join(dir, "loc", "data", "logs", "log.txt")
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), ` test-source log message`) {
		t.Error("bad content: " + string(content))
	}
}
