package userroutes

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestGetAppFromPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(7)
	routeData := &domain.AppspaceRouteData{
		URLTail: "/123",
		Cookie: &domain.Cookie{
			UserID: uid}}

	m := testmocks.NewMockAppModel(mockCtrl)
	m.EXPECT().GetFromID(domain.AppID(123)).Return(&domain.App{OwnerID: uid}, nil)

	a := ApplicationRoutes{
		AppModel: m,
	}

	app, dsErr := a.getAppFromPath(routeData)
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if app == nil {
		t.Fatal("app should not be nil")
	}
}
func TestGetAppFromPathNil(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	routeData := &domain.AppspaceRouteData{
		URLTail: "/"}

	a := ApplicationRoutes{}

	app, dsErr := a.getAppFromPath(routeData)
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if app != nil {
		t.Fatal("app should be nil")
	}
}
func TestGetAppFromPathUnauthorized(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(7)
	routeData := &domain.AppspaceRouteData{
		URLTail: "/123",
		Cookie: &domain.Cookie{
			UserID: uid}}

	m := testmocks.NewMockAppModel(mockCtrl)
	m.EXPECT().GetFromID(domain.AppID(123)).Return(&domain.App{OwnerID: domain.UserID(13)}, nil)

	a := ApplicationRoutes{
		AppModel: m,
	}

	app, dsErr := a.getAppFromPath(routeData)
	if dsErr == nil {
		t.Fatal("should have gotten error")
	}
	if app != nil {
		t.Fatal("app should be nil")
	}
}

func TestGetVersionFromPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	routeData := &domain.AppspaceRouteData{
		URLTail: "/0.1.2"}

	appID := domain.AppID(7)

	m := testmocks.NewMockAppModel(mockCtrl)
	m.EXPECT().GetVersion(appID, domain.Version("0.1.2")).Return(&domain.AppVersion{}, nil)

	a := ApplicationRoutes{
		AppModel: m,
	}

	version, dsErr := a.getVersionFromPath(routeData, appID)
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if version == nil {
		t.Fatal("version should not be nil")
	}
}

func TestGetVersionFromPathNil(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	routeData := &domain.AppspaceRouteData{
		URLTail: "/"}

	a := ApplicationRoutes{}

	version, dsErr := a.getVersionFromPath(routeData, domain.AppID(7))
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if version != nil {
		t.Fatal("version should be nil")
	}
}

/////////////
func TestExtractFiles(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	part, err := writer.CreateFormFile("app_dir", "foo.txt")
	if err != nil {
		panic(err)
	}
	fakeFoo := newFakeFile(200)
	if _, err = io.Copy(part, fakeFoo); err != nil {
		panic(err)
	}

	// another file
	part, err = writer.CreateFormFile("app_dir", "bar.txt")
	if err != nil {
		panic(err)
	}
	fakeBar := newFakeFile(1000)
	if _, err = io.Copy(part, fakeBar); err != nil {
		panic(err)
	}

	err = writer.Close()
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/", buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	appRoutes := &ApplicationRoutes{}
	fileData := appRoutes.extractFiles(req)

	if len(*fileData) != 2 {
		t.Error("fileData should have 2 files", fileData)
	} else {
		if !fakeFoo.matches((*fileData)["foo.txt"]) || !fakeBar.matches((*fileData)["bar.txt"]) {
			t.Error("filedata does not match", fileData)
		}
	}

}

// Test that extract files doesn't fail with empty body
func TestExtractFilesEmptyBody(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	err := writer.Close()
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/", buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	appRoutes := &ApplicationRoutes{}
	fileData := appRoutes.extractFiles(req)

	if len(*fileData) != 0 {
		t.Error("filedata shouldbe zero length", fileData)
	}

}

// check taht a file with no content doesn't muck things up.
func TestExtractFilesEmptyFile(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	_, err := writer.CreateFormFile("app_dir", "foo.txt")
	if err != nil {
		panic(err)
	}

	err = writer.Close()
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/", buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	appRoutes := &ApplicationRoutes{}
	fileData := appRoutes.extractFiles(req)

	if len(*fileData) != 1 {
		t.Error("filedata should be have 1 file", fileData)
	} else if len((*fileData)["foo.txt"]) != 0 {
		t.Error("length of file should be 0", fileData)
	}
}

// from https://play.golang.org/p/9BbS54d8pb
// and https://stackoverflow.com/questions/28174970/implementing-reader-interface

type fakeFile struct {
	// stash supposed file size and currently read amount
	size int
	read int
}

func newFakeFile(size int) *fakeFile { // need to pass size I suppose
	return &fakeFile{
		size: size,
		read: 0}
}

func (f *fakeFile) eof() bool {
	return f.read >= f.size
}

func (f *fakeFile) Read(p []byte) (n int, err error) {
	if f.eof() {
		err = io.EOF
		return
	}
	if l := len(p); l > 0 {
		for n < l {
			p[n] = []byte("A")[0]
			f.read++
			n++
			if f.eof() {
				break
			}
		}
	}
	return
}

// matches compares the bytes from the args
// with the bytes that would be produced by fake file
func (f *fakeFile) matches(b []byte) bool {
	if f.size != len(b) {
		return false
	}

	theByte := []byte("A")[0]
	for i := 0; i < f.size; i++ {
		if b[i] != theByte {
			return false
		}
	}
	return true
}

////////
func TestDeleteVersion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	v := domain.Version("0.1.2")

	appspace := domain.Appspace{
		AppID:      appID,
		AppVersion: domain.Version("0.0.1")}

	asModel := testmocks.NewMockAppspaceModel(mockCtrl)
	asModel.EXPECT().GetForApp(appID).Return([]*domain.Appspace{&appspace}, nil)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().DeleteVersion(appID, v).Return(nil)

	afModel := domain.NewMockAppFilesModel(mockCtrl)
	afModel.EXPECT().Delete("foobar").Return(nil)

	a := ApplicationRoutes{
		AppModel:      appModel,
		AppFilesModel: afModel,
		AppspaceModel: asModel,
	}

	rr := httptest.NewRecorder()

	a.deleteVersion(rr, &domain.AppVersion{AppID: appID, Version: v, LocationKey: "foobar"})

	if rr.Code != http.StatusOK {
		t.Error("http not OK")
	}
}

func TestDeleteVersionInUse(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	v := domain.Version("0.1.2")

	appspace := domain.Appspace{
		AppID:      appID,
		AppVersion: v}

	asModel := testmocks.NewMockAppspaceModel(mockCtrl)
	asModel.EXPECT().GetForApp(appID).Return([]*domain.Appspace{&appspace}, nil)

	a := ApplicationRoutes{
		AppspaceModel: asModel,
	}

	rr := httptest.NewRecorder()

	a.deleteVersion(rr, &domain.AppVersion{AppID: appID, Version: v, LocationKey: "foobar"})

	if rr.Code != http.StatusConflict {
		t.Error("http should be Conflict")
	}
}
