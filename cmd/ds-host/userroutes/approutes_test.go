package userroutes

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

// inputs: userID in ctx, article ID in URL, app model (mocked)
// cases: wrong user id, aritcle id missing, art id not found...
func TestAppCtx(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(7)
	appID := domain.AppID(33)

	m := testmocks.NewMockAppModel(mockCtrl)
	m.EXPECT().GetFromID(appID).Return(domain.App{AppID: appID, OwnerID: uid}, nil)

	a := ApplicationRoutes{
		AppModel: m,
	}

	router := chi.NewMux()
	router.With(a.applicationCtx).Get("/{application}", func(w http.ResponseWriter, r *http.Request) {
		app, ok := domain.CtxAppData(r.Context())
		if !ok {
			t.Error("expected app data")
		}
		if app.AppID != appID {
			t.Error("did not get the app data expected")
		}
	})

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/%v", appID), nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), uid))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
}

func TestAppCtxUnauthorized(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appUid := domain.UserID(7)
	reqUid := domain.UserID(13)
	appID := domain.AppID(33)

	m := testmocks.NewMockAppModel(mockCtrl)
	m.EXPECT().GetFromID(appID).Return(domain.App{AppID: appID, OwnerID: appUid}, nil)

	a := ApplicationRoutes{
		AppModel: m,
	}

	router := chi.NewMux()
	router.With(a.applicationCtx).Get("/{application}", func(w http.ResponseWriter, r *http.Request) {
		t.Error("Route handler should not have been called")
	})

	req, err := http.NewRequest(http.MethodGet, "/33", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), reqUid))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected Forbidden got status %v", rr.Result().Status)
	}
}

func TestAppCtxNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(7)

	m := testmocks.NewMockAppModel(mockCtrl)
	m.EXPECT().GetFromID(domain.AppID(123)).Return(domain.App{}, domain.ErrNoRowsInResultSet)

	a := ApplicationRoutes{
		AppModel: m,
	}

	router := chi.NewMux()
	router.With(a.applicationCtx).Get("/{application}", func(w http.ResponseWriter, r *http.Request) {
		t.Error("Route handler should not have been called")
	})

	req, err := http.NewRequest(http.MethodGet, "/123", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), uid))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected Not Found got status %v", rr.Result().Status)
	}
}

func TestAppVersionCtx(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(7)
	appID := domain.AppID(33)
	app := domain.App{AppID: appID, OwnerID: uid}
	version := domain.Version("1.2.3")

	m := testmocks.NewMockAppModel(mockCtrl)
	m.EXPECT().GetVersion(appID, version).Return(domain.AppVersion{Version: version}, nil)

	a := ApplicationRoutes{
		AppModel: m,
	}

	router := chi.NewMux()
	router.With(a.appVersionCtx).Get("/{application}/version/{app-version}", func(w http.ResponseWriter, r *http.Request) {
		appVersion, ok := domain.CtxAppVersionData(r.Context())
		if !ok {
			t.Error("expected app version data")
		}
		if appVersion.Version != version {
			t.Error("did not get the app version data expected")
		}
	})

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/%v/version/%v", appID, version), nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(domain.CtxWithAppData(req.Context(), app))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
}

func TestInvalidAppVersionCtx(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(7)
	appID := domain.AppID(33)
	app := domain.App{AppID: appID, OwnerID: uid}

	m := testmocks.NewMockAppModel(mockCtrl)
	m.EXPECT().GetVersion(appID, gomock.Any()).Return(domain.AppVersion{}, domain.ErrNoRowsInResultSet)

	a := ApplicationRoutes{
		AppModel: m,
	}

	router := chi.NewMux()
	router.With(a.appVersionCtx).Get("/{application}/version/{app-version}", func(w http.ResponseWriter, r *http.Request) {
		t.Error("route handler should not be called")
	})

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/%v/version/20394jfmfsh", appID), nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(domain.CtxWithAppData(req.Context(), app))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected Not Found got status %v", rr.Result().Status) // we should really validate vesion string and return bad request instead.
	}
}

// ///////////
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
	fileData, err := appRoutes.extractFiles(req)
	if err != nil {
		t.Error(err)
	}

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
	fileData, err := appRoutes.extractFiles(req)
	if err != nil {
		t.Error(err)
	}

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
	fileData, err := appRoutes.extractFiles(req)
	if err != nil {
		t.Error(err)
	}

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

// //////
func TestDeleteVersion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	v := domain.Version("0.1.2")
	loc := "test-loc"

	deleteApp := testmocks.NewMockDeleteApp(mockCtrl)
	deleteApp.EXPECT().DeleteVersion(appID, v).Return(nil)

	a := ApplicationRoutes{
		DeleteApp: deleteApp,
	}

	req, _ := http.NewRequest("get", "/", nil)
	req = req.WithContext(domain.CtxWithAppVersionData(req.Context(), domain.AppVersion{AppID: appID, Version: v, LocationKey: loc}))

	rr := httptest.NewRecorder()

	a.deleteVersion(rr, req)

	if rr.Code != http.StatusOK {
		t.Error("http not OK")
	}
}

func TestDeleteVersionInUse(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	v := domain.Version("0.1.2")

	deleteApp := testmocks.NewMockDeleteApp(mockCtrl)
	deleteApp.EXPECT().DeleteVersion(appID, v).Return(domain.ErrAppVersionInUse)

	a := ApplicationRoutes{
		DeleteApp: deleteApp,
	}

	req, _ := http.NewRequest("get", "/", nil)
	req = req.WithContext(domain.CtxWithAppVersionData(req.Context(), domain.AppVersion{AppID: appID, Version: v}))

	rr := httptest.NewRecorder()

	a.deleteVersion(rr, req)

	if rr.Code != http.StatusConflict {
		t.Error("http should be Conflict")
	}
}
