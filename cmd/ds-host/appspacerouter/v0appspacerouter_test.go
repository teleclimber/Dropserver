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
	routeConfig := domain.V0AppRoute{
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
	req = req.WithContext(domain.CtxWithV0RouteConfig(req.Context(), routeConfig))

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
	routeConfig := domain.V0AppRoute{
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
	req = req.WithContext(domain.CtxWithV0RouteConfig(req.Context(), routeConfig))

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
	routeConfig := domain.V0AppRoute{
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
	ctx := domain.CtxWithV0RouteConfig(req.Context(), routeConfig)
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
	routeConfig := domain.V0AppRoute{
		Auth: domain.AppspaceRouteAuth{
			Allow:      "authorized",
			Permission: "delete",
		},
	}

	user := domain.AppspaceUser{Permissions: []string{"create", "update"}}

	v0 := &V0{}

	nextCalled := false
	handler := v0.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	ctx := domain.CtxWithV0RouteConfig(req.Context(), routeConfig)
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
	routeConfig := domain.V0AppRoute{
		Auth: domain.AppspaceRouteAuth{
			Allow:      "authorized",
			Permission: "delete",
		},
	}

	user := domain.AppspaceUser{Permissions: []string{"create", "update", "delete"}}

	v0 := &V0{}

	nextCalled := false
	handler := v0.authorizeRoute(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	ctx := domain.CtxWithV0RouteConfig(req.Context(), routeConfig)
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

	appspaceID := domain.AppspaceID(7)

	v0TokenManager := testmocks.NewMockV0TokenManager(mockCtrl)
	v0TokenManager.EXPECT().CheckToken(appspaceID, "abcd").Return(domain.V0AppspaceLoginToken{}, false)

	v0 := &V0{
		V0TokenManager: v0TokenManager,
	}

	nextCalled := false
	handler := v0.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, _ := http.NewRequest(http.MethodGet, "/?dropserver-login-token=abcd", nil)
	req = req.WithContext(domain.CtxWithAppspaceData(req.Context(), domain.Appspace{AppspaceID: appspaceID}))
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
	v0TokenManager.EXPECT().CheckToken(appspaceID, "abcd").Return(domain.V0AppspaceLoginToken{AppspaceID: appspaceID, ProxyID: proxyID}, true)

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

func TestGetConfigPath(t *testing.T) {
	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}
	v0 := &V0{
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
			routeConfig := domain.V0AppRoute{
				Options: domain.V0AppRouteOptions{
					Path: c.configP,
				}}

			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			ctx := domain.CtxWithAppVersionData(req.Context(), appVersion)
			ctx = domain.CtxWithV0RouteConfig(ctx, routeConfig)

			p, err := v0.getConfigPath(req.WithContext(ctx))
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
	v0 := &V0{}

	basePath := "/base/path/"
	cases := []struct {
		routeP domain.V0AppRoutePath
		reqP   string
		expP   string
		err    bool
	}{
		{
			domain.V0AppRoutePath{Path: "/static/", End: false},
			"/static/style/app.css",
			"/base/path/style/app.css",
			false},
		{
			domain.V0AppRoutePath{Path: "/../../", End: false}, // An attempt to use config to break out of /base/path/
			"/not-secrets.txt",
			"/base/path/not-secrets.txt",
			false},
		{
			domain.V0AppRoutePath{Path: "/static/", End: false}, // using request path to break out of /base/path
			"/static/../../secrets.txt",
			"",
			true},
	}

	for _, c := range cases {
		t.Run(c.reqP, func(t *testing.T) {
			routeConfig := domain.V0AppRoute{
				Path: c.routeP}

			req, _ := http.NewRequest(http.MethodGet, c.reqP, nil)
			ctx := domain.CtxWithV0RouteConfig(req.Context(), routeConfig)

			p, err := v0.joinBaseToRequest(basePath, req.WithContext(ctx))
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
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}
	routeConfig := domain.V0AppRoute{
		Path: domain.V0AppRoutePath{Path: "/some-files", End: true},
		Type: "static",
		Options: domain.V0AppRouteOptions{
			Path: "@app/static-files/",
		}}

	v0 := &V0{
		AppLocation2Path: &l2p{appFiles: dir},
	}

	p := filepath.Join(dir, "app-version-123", "app", "static-files", "css")
	err = os.MkdirAll(p, 0755)
	if err != nil {
		t.Error(err)
	}
	fileData := []byte("BODY { color: red; }")
	ioutil.WriteFile(filepath.Join(p, "style.css"), fileData, 0644)

	req, _ := http.NewRequest("GET", "/some-files/css/style.css", nil)
	ctx := domain.CtxWithAppVersionData(req.Context(), appVersion)
	ctx = domain.CtxWithV0RouteConfig(ctx, routeConfig)
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

	// config := &domain.RuntimeConfig{}
	// config.Exec.AppsPath = dir

	appVersion := domain.AppVersion{
		LocationKey: "app-version-123",
	}
	routeConfig := domain.V0AppRoute{
		Path: domain.V0AppRoutePath{Path: "/", End: true},
		Type: "static",
		Options: domain.V0AppRouteOptions{
			Path: "@app/static-files/index.html",
		},
	}

	v0 := &V0{
		AppLocation2Path: &l2p{appFiles: dir},
		//Config: config,
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
	ctx = domain.CtxWithV0RouteConfig(ctx, routeConfig)
	rr := httptest.NewRecorder()

	v0.serveFile(rr, req.WithContext(ctx))

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
