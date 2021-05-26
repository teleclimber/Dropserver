package appspacerouter

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestAuthorizePublic(t *testing.T) {
	routeConfig := domain.AppspaceRouteConfig{
		Auth: domain.AppspaceRouteAuth{
			Allow: "public",
		},
	}

	v0 := &V0{}

	nextCalled := false
	handler := v0.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	routeConfig := domain.AppspaceRouteConfig{
		Auth: domain.AppspaceRouteAuth{
			Allow: "authorized",
		},
	}

	v0 := &V0{}

	nextCalled := false
	handler := v0.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	routeConfig := domain.AppspaceRouteConfig{
		Auth: domain.AppspaceRouteAuth{
			Allow: "authorized",
		},
	}

	user := domain.AppspaceUser{}

	v0 := &V0{}

	nextCalled := false
	handler := v0.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	routeConfig := domain.AppspaceRouteConfig{
		Auth: domain.AppspaceRouteAuth{
			Allow:      "authorized",
			Permission: "delete",
		},
	}

	user := domain.AppspaceUser{Permissions: "create,update"}

	v0 := &V0{}

	nextCalled := false
	handler := v0.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	routeConfig := domain.AppspaceRouteConfig{
		Auth: domain.AppspaceRouteAuth{
			Allow:      "authorized",
			Permission: "delete",
		},
	}

	user := domain.AppspaceUser{Permissions: "create,update,delete"}

	v0 := &V0{}

	nextCalled := false
	handler := v0.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestLoginTokenNoToken(t *testing.T) {
	v0 := &V0{}

	nextCalled := false
	handler := v0.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
	if !nextCalled {
		t.Error("middleware did not call next")
	}
}

func TestLoginTokenTwoTokens(t *testing.T) {
	v0 := &V0{}

	nextCalled := false
	handler := v0.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/?dropserver-login-token=aaaa&dropserver-login-token=bbbbbbb", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected Bad Request got status %v", rr.Result().Status)
	}
	if nextCalled {
		t.Error("middleware called next")
	}
}

func TestLoginTokenNotfound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	v0TokenManager := testmocks.NewMockV0TokenManager(mockCtrl)
	v0TokenManager.EXPECT().CheckToken("abcd").Return(domain.V0AppspaceLoginToken{}, false)

	v0 := &V0{
		V0TokenManager: v0TokenManager,
	}

	nextCalled := false
	handler := v0.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/?dropserver-login-token=abcd", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
	if !nextCalled {
		t.Error("middleware did not call next")
	}
}

func TestLoginToken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proxyID := domain.ProxyID("uvw")
	appspaceID := domain.AppspaceID(7)
	domainName := "as1.ds.dev"

	v0TokenManager := testmocks.NewMockV0TokenManager(mockCtrl)
	v0TokenManager.EXPECT().CheckToken("abcd").Return(domain.V0AppspaceLoginToken{AppspaceID: appspaceID, ProxyID: proxyID}, true)

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().SetForAppspace(gomock.Any(), proxyID, appspaceID, domainName).Return("cid", nil)

	v0 := &V0{
		V0TokenManager: v0TokenManager,
		Authenticator:  authenticator,
	}

	nextCalled := false
	handler := v0.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqProxyID, ok := domain.CtxAppspaceUserProxyID(r.Context())
		if !ok {
			t.Error("no proxy id set")
		}
		if reqProxyID != proxyID {
			t.Error("wrong proxy id")
		}

		reqCookieID, ok := domain.CtxSessionID(r.Context())
		if !ok {
			t.Error("no cookie id")
		}
		if reqCookieID != "cid" {
			t.Error("wrong cookie id")
		}

		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/?dropserver-login-token=abcd", nil)
	req = req.WithContext(domain.CtxWithAppspaceData(req.Context(), domain.Appspace{AppspaceID: appspaceID, DomainName: domainName}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
	if !nextCalled {
		t.Error("middleware did not call next")
	}
}

func TestGetFilePath(t *testing.T) {
	config := &domain.RuntimeConfig{}
	config.Exec.AppsPath = "/data-dir/apps-path"

	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}

	routeConfig := domain.AppspaceRouteConfig{
		Path: "/some-files",
		Handler: domain.AppspaceRouteHandler{
			Type: "file",
			Path: "@app/static-files/",
		}}

	v0 := &V0{
		Config: config,
	}

	req, _ := http.NewRequest(http.MethodGet, "/some-files/css/style.css", nil)
	ctx := domain.CtxWithAppVersionData(req.Context(), appVersion)
	ctx = domain.CtxWithRouteConfig(ctx, routeConfig)

	p, err := v0.getFilePath(req.WithContext(ctx))
	if err != nil {
		t.Error(err)
	}
	expected := "/data-dir/apps-path/app-version-123/static-files/css/style.css"
	if p != expected {
		t.Error("expected " + expected)
	}

	// now try illegal path:
	req, _ = http.NewRequest(http.MethodGet, "/some-files/../../gotcha.txt", nil)
	ctx = domain.CtxWithAppVersionData(req.Context(), appVersion)
	ctx = domain.CtxWithRouteConfig(ctx, routeConfig)
	p, err = v0.getFilePath(req.WithContext(ctx))
	if err == nil {
		t.Error("expected error, got " + p)
	}

	// now try a specific file for route path
	routeConfig.Path = "/some-files/css/style.css"
	routeConfig.Handler.Path = "@app/static-files/style.css"
	req, _ = http.NewRequest(http.MethodGet, "/some-files/css/style.css", nil)
	ctx = domain.CtxWithAppVersionData(req.Context(), appVersion)
	ctx = domain.CtxWithRouteConfig(ctx, routeConfig)
	p, err = v0.getFilePath(req.WithContext(ctx))
	if err != nil {
		t.Error(err)
	}
	expected = "/data-dir/apps-path/app-version-123/static-files/style.css"
	if p != expected {
		t.Error("expected " + expected)
	}
}

func TestServeFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	config := &domain.RuntimeConfig{}
	config.Exec.AppsPath = dir

	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}
	routeConfig := domain.AppspaceRouteConfig{
		Path: "/some-files",
		Handler: domain.AppspaceRouteHandler{
			Type: "file",
			Path: "@app/static-files/",
		}}

	v0 := &V0{
		Config: config,
	}

	p := filepath.Join(dir, "app-version-123", "static-files", "css")
	err = os.MkdirAll(p, 0755)
	if err != nil {
		t.Error(err)
	}
	fileData := []byte("BODY { color: red; }")
	ioutil.WriteFile(filepath.Join(p, "style.css"), fileData, 0644)

	req, _ := http.NewRequest("GET", "/some-files/css/style.css", nil)
	ctx := domain.CtxWithAppVersionData(req.Context(), appVersion)
	ctx = domain.CtxWithRouteConfig(ctx, routeConfig)
	rr := httptest.NewRecorder()

	v0.serveFile(rr, req.WithContext(ctx))

	respString := rr.Body.String()
	if respString != string(fileData) {
		t.Error("expected file data, got " + respString)
	}
}

func TestServeFileOverlapPath(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	config := &domain.RuntimeConfig{}
	config.Exec.AppsPath = dir
	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}
	routeConfig := domain.AppspaceRouteConfig{
		Path: "/",
		Handler: domain.AppspaceRouteHandler{
			Type: "file",
			Path: "@app/static-files/index.html",
		},
	}

	v0 := &V0{
		Config: config,
	}

	p := filepath.Join(dir, "app-version-123", "static-files")
	err = os.MkdirAll(p, 0755)
	if err != nil {
		t.Error(err)
	}
	fileData := []byte("<h1>hello world</h1")
	ioutil.WriteFile(filepath.Join(p, "index.html"), fileData, 0644)

	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	ctx := domain.CtxWithAppVersionData(req.Context(), appVersion)
	ctx = domain.CtxWithRouteConfig(ctx, routeConfig)
	rr := httptest.NewRecorder()

	v0.serveFile(rr, req.WithContext(ctx))

	if rr.Result().StatusCode != http.StatusNotFound {
		t.Error("expected 404")
	}
}

// with appspace and route assume route is proxy
// -> calls proxy
// func TestServeHTTPProxyRoute(t *testing.T) {

// 	mockCtrl := gomock.NewController(t)
// 	defer mockCtrl.Finish()

// 	appspaceID := domain.AppspaceID(7)
// 	//proxyID := domain.ProxyID("abc")

// 	routesV0 := domain.NewMockV0RouteModel(mockCtrl)
// 	routesV0.EXPECT().Match("GET", "/abc").Return(&domain.AppspaceRouteConfig{
// 		Handler: domain.AppspaceRouteHandler{Type: "function"},
// 		Auth:    domain.AppspaceRouteAuth{Allow: "public"},
// 	}, nil)
// 	asRoutesModel := domain.NewMockAppspaceRouteModels(mockCtrl)
// 	asRoutesModel.EXPECT().GetV0(appspaceID).Return(routesV0)
// 	// asUserModel := testmocks.NewMockAppspaceUserModel(mockCtrl)
// 	// asUserModel.EXPECT().Get(appspaceID, proxyID).Return(domain.AppspaceUser{AppspaceID: appspaceID, ProxyID: proxyID}, nil)
// 	sandboxProxy := domain.NewMockRouteHandler(mockCtrl)

// 	v0 := &V0{
// 		AppspaceRouteModels: asRoutesModel,
// 		//AppspaceUserModel:   asUserModel,
// 		SandboxProxy: sandboxProxy}

// 	routeData := &domain.AppspaceRouteData{
// 		URLTail: "/abc",
// 		Appspace: &domain.Appspace{
// 			AppspaceID: appspaceID,
// 		},
// 	}

// 	req, err := http.NewRequest("GET", "/", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	rr := httptest.NewRecorder()

// 	sandboxProxy.EXPECT().ServeHTTP(gomock.Any(), req, routeData)

// 	// ^^ here we are checking against routeData, which is a pointer
// 	// so it's not testing whether the call populated routeData correctly.

// 	v0.ServeHTTP(rr, req)

// 	// TODO: check routeData was properly augmented (app, appspace)
// }
