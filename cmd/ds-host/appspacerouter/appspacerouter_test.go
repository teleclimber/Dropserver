package appspacerouter

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestAppspaceUnavailable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().Ready(appspaceID).Return(false)

	appspaceRouter := &AppspaceRouter{
		AppspaceStatus: appspaceStatus}

	nextCalled := false
	handler := appspaceRouter.appspaceAvailable(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAppspaceData(req.Context(), domain.Appspace{AppspaceID: appspaceID}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected unavailable, got %d", rr.Code)
	}
	if nextCalled {
		t.Error("next got called when it should not have")
	}
}

func TestAppspaceAvailable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)
	appspaceStatus.EXPECT().Ready(appspaceID).Return(true)

	appspaceRouter := &AppspaceRouter{
		AppspaceStatus: appspaceStatus}

	nextCalled := false
	handler := appspaceRouter.appspaceAvailable(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAppspaceData(req.Context(), domain.Appspace{AppspaceID: appspaceID}))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected OK, got %d", rr.Code)
	}
	if !nextCalled {
		t.Error("next did not get called")
	}
}

// Somehow we can't find the app referred to by appspace.
// server 500 and log error
// func TestServeHTTPBadApp(t *testing.T) {

// 	mockCtrl := gomock.NewController(t)
// 	defer mockCtrl.Finish()

// 	appspaceID := domain.AppspaceID(7)
// 	appID := domain.AppID(11)

// 	appModel := testmocks.NewMockAppModel(mockCtrl)
// 	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
// 	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)

// 	appspaceRoutes := &AppspaceRouter{
// 		AppModel:       appModel,
// 		AppspaceModel:  appspaceModel,
// 		AppspaceStatus: appspaceStatus}
// 	appspaceRoutes.Init()

// 	req, err := http.NewRequest("GET", "/", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req.Host = "as1.ds.dev"

// 	rr := httptest.NewRecorder()

// 	appspaceModel.EXPECT().GetFromDomain("as1.ds.dev").Return(&domain.Appspace{
// 		AppspaceID: appspaceID,
// 		DomainName: "as1.ds.dev",
// 		AppID:      appID}, nil)
// 	appModel.EXPECT().GetFromID(appID).Return(nil, errors.New("some error"))
// 	appspaceStatus.EXPECT().Ready(appspaceID).Return(true)

// 	appspaceRoutes.ServeHTTP(rr, req)

// 	if rr.Code != http.StatusInternalServerError {
// 		t.Errorf("Expected 500, got %d", rr.Code)
// 	}
// }

// need to test events

// then need to test a successful route call.
func TestEmitLiveCountNoop(t *testing.T) {
	appspaceRoutes := &AppspaceRouter{}
	appspaceRoutes.liveCounter = make(map[domain.AppspaceID]int)
	appspaceRoutes.subscribers = make(map[domain.AppspaceID][]chan<- int)

	appspaceID := domain.AppspaceID(7)

	appspaceRoutes.emitLiveCount(appspaceID, 11)
}

func TestEmitLiveCount(t *testing.T) {
	appspaceRoutes := &AppspaceRouter{}
	appspaceRoutes.liveCounter = make(map[domain.AppspaceID]int)
	appspaceRoutes.subscribers = make(map[domain.AppspaceID][]chan<- int)

	appspaceID := domain.AppspaceID(7)

	subChan := make(chan int)
	appspaceRoutes.subscribers[appspaceID] = append([]chan<- int{}, subChan)

	go func() {
		appspaceRoutes.emitLiveCount(appspaceID, 11)
	}()

	count := <-subChan
	if count != 11 {
		t.Error("expected count to be 11")
	}
}

func TestSubscribe(t *testing.T) {
	appspaceRoutes := &AppspaceRouter{}
	appspaceRoutes.liveCounter = make(map[domain.AppspaceID]int)
	appspaceRoutes.subscribers = make(map[domain.AppspaceID][]chan<- int)

	appspaceID := domain.AppspaceID(7)

	subChan := make(chan int)

	count := appspaceRoutes.SubscribeLiveCount(appspaceID, subChan)
	if count != 0 {
		t.Error("count should be zero")
	}

	go func() {
		appspaceRoutes.emitLiveCount(appspaceID, 11)
	}()

	count = <-subChan
	if count != 11 {
		t.Error("expected count to be 11")
	}

	appspaceRoutes.UnsubscribeLiveCount(appspaceID, subChan)
	if len(appspaceRoutes.subscribers[appspaceID]) != 0 {
		t.Error("there should be no subscribers left")
	}
}

func TestIncrement(t *testing.T) {
	appspaceRoutes := &AppspaceRouter{}
	appspaceRoutes.subscribers = make(map[domain.AppspaceID][]chan<- int)

	appspaceID := domain.AppspaceID(7)

	appspaceRoutes.liveCounterMux.Lock()
	appspaceRoutes.liveCounter = make(map[domain.AppspaceID]int)
	appspaceRoutes.liveCounter[appspaceID] = 2
	appspaceRoutes.liveCounterMux.Unlock()

	subChan := make(chan int)
	count := appspaceRoutes.SubscribeLiveCount(appspaceID, subChan)
	if count != 2 {
		t.Error("count should be two")
	}

	go func() {
		appspaceRoutes.decrementLiveCount(appspaceID)
	}()
	go func() {
		appspaceRoutes.decrementLiveCount(appspaceID)
	}()

	done := make(chan struct{})
	unsubscribed := false
	for count = range subChan {
		if count == 0 && !unsubscribed {
			unsubscribed = true // don't double-unsubscribe, but mostly don't close chan twice.
			go func() {         // don't unsubscribe in chan listener loop, deadlocks can occur.
				appspaceRoutes.UnsubscribeLiveCount(appspaceID, subChan)
				done <- struct{}{}
			}()
		}
	}
	<-done
}

func TestAuthorizePublic(t *testing.T) {
	routeConfig := domain.AppRoute{
		Auth: domain.AppspaceRouteAuth{
			Allow: "public",
		},
	}

	ar := &AppspaceRouter{}

	nextCalled := false
	handler := ar.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(domain.CtxWithRouteConfig(req.Context(), routeConfig))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
	if !nextCalled {
		t.Error("middleware did not call next")
	}
}

func TestAuthorizeForbidden(t *testing.T) {
	routeConfig := domain.AppRoute{
		Auth: domain.AppspaceRouteAuth{
			Allow: "authorized",
		},
	}

	ar := &AppspaceRouter{}

	nextCalled := false
	handler := ar.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(domain.CtxWithRouteConfig(req.Context(), routeConfig))

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected Forbidden got status %v", rr.Result().Status)
	}
	if nextCalled {
		t.Error("middleware should not call next")
	}
}

func TestAuthorizedUser(t *testing.T) {
	routeConfig := domain.AppRoute{
		Auth: domain.AppspaceRouteAuth{
			Allow: "authorized",
		},
	}

	user := domain.AppspaceUser{}

	ar := &AppspaceRouter{}

	nextCalled := false
	handler := ar.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	ctx := domain.CtxWithRouteConfig(req.Context(), routeConfig)
	ctx = domain.CtxWithAppspaceUserData(ctx, user)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req.WithContext(ctx))

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
	if !nextCalled {
		t.Error("middleware did not call next")
	}
}

func TestAuthorizePermissionDenied(t *testing.T) {
	routeConfig := domain.AppRoute{
		Auth: domain.AppspaceRouteAuth{
			Allow:      "authorized",
			Permission: "delete",
		},
	}

	user := domain.AppspaceUser{Permissions: []string{"create", "update"}}

	ar := &AppspaceRouter{}

	nextCalled := false
	handler := ar.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	ctx := domain.CtxWithRouteConfig(req.Context(), routeConfig)
	ctx = domain.CtxWithAppspaceUserData(ctx, user)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req.WithContext(ctx))

	if rr.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected Forbidden got status %v", rr.Result().Status)
	}
	if nextCalled {
		t.Error("middleware should not call next")
	}
}

func TestAuthorizePermissionAllowed(t *testing.T) {
	routeConfig := domain.AppRoute{
		Auth: domain.AppspaceRouteAuth{
			Allow:      "authorized",
			Permission: "delete",
		},
	}

	user := domain.AppspaceUser{Permissions: []string{"create", "update", "delete"}}

	ar := &AppspaceRouter{}

	nextCalled := false
	handler := ar.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	ctx := domain.CtxWithRouteConfig(req.Context(), routeConfig)
	ctx = domain.CtxWithAppspaceUserData(ctx, user)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req.WithContext(ctx))

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
	if !nextCalled {
		t.Error("middleware did not call next")
	}
}

func TestGetConfigPath(t *testing.T) {
	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}
	ar := &AppspaceRouter{
		AppLocation2Path: &l2p{appFiles: "/data-dir/apps-path"},
	}

	cases := []struct {
		configP string
		expP    string
		err     bool
	}{
		{
			"@app/static-files/",
			"/data-dir/apps-path/app-version-123/app/static-files",
			false},
		{
			"@app/../",
			"",
			true},
		{
			"@nonsense/static-files/",
			"",
			true},
		// Could add cases to ensure @appspace and others work correctly.
	}

	for _, c := range cases {
		t.Run(c.configP+" -> "+c.expP, func(t *testing.T) {
			routeConfig := domain.AppRoute{
				Options: domain.AppRouteOptions{
					Path: c.configP,
				}}

			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			ctx := domain.CtxWithAppVersionData(req.Context(), appVersion)
			ctx = domain.CtxWithRouteConfig(ctx, routeConfig)

			p, err := ar.getConfigPath(req.WithContext(ctx))
			if err == nil && c.err {
				t.Error("Expected error, got nil")
			}
			if err != nil && !c.err {
				t.Errorf("Unexpected error: %v", err)
			}
			if p != c.expP {
				t.Errorf("expected %v, got %v", c.expP, p)
			}
		})
	}
}

func TestJoinBaseToRequest(t *testing.T) {
	ar := &AppspaceRouter{}

	basePath := "/base/path/"
	cases := []struct {
		routeP domain.AppRoutePath
		reqP   string
		expP   string
		err    bool
	}{
		{
			domain.AppRoutePath{Path: "/static/", End: false},
			"/static/style/app.css",
			"/base/path/style/app.css",
			false},
		{
			domain.AppRoutePath{Path: "/../../", End: false}, // An attempt to use config to break out of /base/path/
			"/not-secrets.txt",
			"/base/path/not-secrets.txt",
			false},
		{
			domain.AppRoutePath{Path: "/static/", End: false}, // using request path to break out of /base/path
			"/static/../../secrets.txt",
			"",
			true},
	}

	for _, c := range cases {
		t.Run(c.reqP, func(t *testing.T) {
			routeConfig := domain.AppRoute{
				Path: c.routeP}

			req, _ := http.NewRequest(http.MethodGet, c.reqP, nil)
			ctx := domain.CtxWithRouteConfig(req.Context(), routeConfig)

			p, err := ar.joinBaseToRequest(basePath, req.WithContext(ctx))
			if err == nil && c.err {
				t.Error("Expected error, got nil")
			}
			if err != nil && !c.err {
				t.Errorf("Unexpected error: %v", err)
			}
			if p != c.expP {
				t.Errorf("expected %v, got %v", c.expP, p)
			}
		})
	}
}

// TODO: improve serveFile tests by setting up an env (files, routes, ...) and using cases to try serving different files
func TestServeFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}
	routeConfig := domain.AppRoute{
		Path: domain.AppRoutePath{Path: "/some-files", End: true},
		Type: "static",
		Options: domain.AppRouteOptions{
			Path: "@app/static-files/",
		}}

	ar := &AppspaceRouter{
		AppLocation2Path: &l2p{appFiles: dir},
	}

	p := filepath.Join(dir, "app-version-123", "app", "static-files", "css")
	err = os.MkdirAll(p, 0755)
	if err != nil {
		t.Error(err)
	}
	fileData := []byte("BODY { color: red; }")
	os.WriteFile(filepath.Join(p, "style.css"), fileData, 0644)

	req, _ := http.NewRequest("GET", "/some-files/css/style.css", nil)
	ctx := domain.CtxWithAppVersionData(req.Context(), appVersion)
	ctx = domain.CtxWithRouteConfig(ctx, routeConfig)
	rr := httptest.NewRecorder()

	ar.serveFile(rr, req.WithContext(ctx))

	respString := rr.Body.String()
	if respString != string(fileData) {
		t.Error("expected file data, got " + respString)
	}
}

func TestServeFileOverlapPath(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	// config := &domain.RuntimeConfig{}
	// config.Exec.AppsPath = dir

	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}
	routeConfig := domain.AppRoute{
		Path: domain.AppRoutePath{Path: "/", End: true},
		Type: "static",
		Options: domain.AppRouteOptions{
			Path: "@app/static-files/index.html",
		},
	}

	ar := &AppspaceRouter{
		AppLocation2Path: &l2p{appFiles: dir},
		//Config: config,
	}

	p := filepath.Join(dir, "app-version-123", "static-files")
	err = os.MkdirAll(p, 0755)
	if err != nil {
		t.Error(err)
	}
	fileData := []byte("<h1>hello world</h1")
	os.WriteFile(filepath.Join(p, "index.html"), fileData, 0644)

	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	ctx := domain.CtxWithAppVersionData(req.Context(), appVersion)
	ctx = domain.CtxWithRouteConfig(ctx, routeConfig)
	rr := httptest.NewRecorder()

	ar.serveFile(rr, req.WithContext(ctx))

	if rr.Result().StatusCode != http.StatusNotFound {
		t.Error("expected 404")
	}
}

type l2p struct {
	appFiles string
}

func (l *l2p) Files(loc string) string {
	return filepath.Join(l.appFiles, loc, "app")
}
