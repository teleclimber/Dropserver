package appspacerouter

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

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
	dir, err := ioutil.TempDir("", "")
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
	ioutil.WriteFile(filepath.Join(p, "style.css"), fileData, 0644)

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
	ioutil.WriteFile(filepath.Join(p, "index.html"), fileData, 0644)

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
