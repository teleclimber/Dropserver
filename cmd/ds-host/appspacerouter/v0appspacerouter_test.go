package appspacerouter

import (
	"errors"
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

func TestAuthorize(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ownerID := domain.UserID(7)
	proxyID := domain.ProxyID("abc")
	appspaceID := domain.AppspaceID(11)

	v0 := &V0{}

	routeData := domain.AppspaceRouteData{
		RouteConfig: &domain.AppspaceRouteConfig{
			Auth: domain.AppspaceRouteAuth{
				Allow: "public",
			},
		},
	}
	a := v0.authorizeAppspace(&routeData, domain.Authentication{}, domain.AppspaceUser{})
	if !a {
		t.Error("expected public route authorized")
	}

	routeData.RouteConfig.Auth.Allow = "authorized"
	routeData.Appspace = &domain.Appspace{OwnerID: ownerID, AppspaceID: appspaceID}
	a = v0.authorizeAppspace(&routeData, domain.Authentication{}, domain.AppspaceUser{})
	if a {
		t.Error("expected unauthorized because no auth")
	}

	auth := domain.Authentication{
		Authenticated: true,
		UserID:        domain.UserID(13),
		ProxyID:       proxyID}

	a = v0.authorizeAppspace(&routeData, auth, domain.AppspaceUser{})
	if a {
		t.Error("expected unauthorized because wrong user for auth")
	}

	auth.UserID = ownerID
	auth.AppspaceID = domain.AppspaceID(33)
	a = v0.authorizeAppspace(&routeData, auth, domain.AppspaceUser{})
	if a {
		t.Error("expected unauthorized because wrong appspace ID")
	}

	auth.AppspaceID = appspaceID
	a = v0.authorizeAppspace(&routeData, auth, domain.AppspaceUser{AppspaceID: appspaceID})
	if !a {
		t.Error("expected route authorized")
	}
}

func TestAuthorizePermissions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	v0 := &V0{}

	a := v0.authorizePermission("delete", domain.AppspaceUser{})
	if a {
		t.Error("expected route unauthorized (no permissions)")
	}

	a = v0.authorizePermission("delete", domain.AppspaceUser{Permissions: "create,archive"})
	if a {
		t.Error("expected route unauthorized (incorrect permissions)")
	}

	a = v0.authorizePermission("delete", domain.AppspaceUser{Permissions: "create,delete"})
	if !a {
		t.Error("expected route authorized")
	}
}

// Test login tokens and its failure modes.
func TestProcessLoginTokenNone(t *testing.T) {
	req, err := http.NewRequest("GET", "/some-files/css/style.css", nil)
	if err != nil {
		t.Fatal(err)
	}

	v0 := &V0{}

	auth, err := v0.processLoginToken(httptest.NewRecorder(), req, &domain.AppspaceRouteData{})
	if err != nil {
		t.Error(err)
	}
	if auth.Authenticated {
		t.Error("expected authenticated to be false")
	}
}

func TestProcessLoginToken(t *testing.T) {
	req, err := http.NewRequest("GET", "/some-files/css/style.css?dropserver-login-token=abc&dropserver-login-token=def", nil)
	if err != nil {
		t.Fatal(err)
	}

	v0 := &V0{}

	auth, err := v0.processLoginToken(httptest.NewRecorder(), req, &domain.AppspaceRouteData{})
	if err == nil {
		t.Error("expected error due to multiple tokens")
	}
	if auth.Authenticated {
		t.Error("expected authenticated to be false")
	}
}

func TestProcessLoginTokenBadToken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceLogin := testmocks.NewMockAppspaceLogin(mockCtrl)
	appspaceLogin.EXPECT().CheckRedirectToken("abc").Return(domain.AppspaceLoginToken{}, errors.New("No valid token"))

	req, err := http.NewRequest("GET", "/some-files/css/style.css?dropserver-login-token=abc", nil)
	if err != nil {
		t.Fatal(err)
	}

	v0 := &V0{
		AppspaceLogin: appspaceLogin,
	}

	auth, err := v0.processLoginToken(httptest.NewRecorder(), req, &domain.AppspaceRouteData{})
	if err == nil {
		t.Error("expected error")
	}
	if auth.Authenticated {
		t.Error("expected authenticated to be false")
	}
}

func TestProcessLoginTokenWrongAppspace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceLogin := testmocks.NewMockAppspaceLogin(mockCtrl)
	appspaceLogin.EXPECT().CheckRedirectToken("abc").Return(domain.AppspaceLoginToken{AppspaceID: domain.AppspaceID(13)}, nil)

	req, err := http.NewRequest("GET", "/some-files/css/style.css?dropserver-login-token=abc", nil)
	if err != nil {
		t.Fatal(err)
	}

	v0 := &V0{
		AppspaceLogin: appspaceLogin,
	}

	auth, err := v0.processLoginToken(httptest.NewRecorder(), req, &domain.AppspaceRouteData{
		Appspace: &domain.Appspace{AppspaceID: domain.AppspaceID(7)}})
	if err == nil {
		t.Error("expected error")
	}
	if auth.Authenticated {
		t.Error("expected authenticated to be false")
	}
}

func TestProcessLoginTokenOK(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	proxyID := domain.ProxyID("proxy-ideee")
	appspaceID := domain.AppspaceID(7)

	appspaceLogin := testmocks.NewMockAppspaceLogin(mockCtrl)
	appspaceLogin.EXPECT().CheckRedirectToken("abc").Return(domain.AppspaceLoginToken{
		ProxyID:    proxyID,
		AppspaceID: appspaceID}, nil)

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().SetForAppspace(gomock.Any(), proxyID, appspaceID, "some.host").Return("somecookie", nil)

	req, err := http.NewRequest("GET", "/some-files/css/style.css?dropserver-login-token=abc", nil)
	if err != nil {
		t.Fatal(err)
	}

	v0 := &V0{
		Authenticator: authenticator,
		AppspaceLogin: appspaceLogin,
	}

	auth, err := v0.processLoginToken(httptest.NewRecorder(), req, &domain.AppspaceRouteData{
		Appspace: &domain.Appspace{
			AppspaceID: appspaceID,
			DomainName: "some.host",
		}})
	if err != nil {
		t.Error(err)
	}
	if !auth.Authenticated {
		t.Error("expected authenticated to be true")
	}
	if auth.AppspaceID != appspaceID {
		t.Error("wrong appspace ID")
	}
	if auth.CookieID != "somecookie" {
		t.Error("wrong cookie ID")
	}
	if auth.UserAccount {
		t.Error("should not be for user account")
	}
	if auth.ProxyID != proxyID {
		t.Error("wrong proxy id")
	}
}

func TestGetFilePath(t *testing.T) {
	config := &domain.RuntimeConfig{}
	config.Exec.AppsPath = "/data-dir/apps-path"
	routeData := &domain.AppspaceRouteData{
		AppVersion: &domain.AppVersion{
			LocationKey: "app-version-123",
		},
		RouteConfig: &domain.AppspaceRouteConfig{
			Path: "/some-files",
			Handler: domain.AppspaceRouteHandler{
				Type: "file",
				Path: "@app/static-files/",
			},
		},
		URLTail: "/some-files/css/style.css",
	}

	v0 := &V0{
		Config: config,
	}

	p, err := v0.getFilePath(routeData)
	if err != nil {
		t.Error(err)
	}
	expected := "/data-dir/apps-path/app-version-123/static-files/css/style.css"
	if p != expected {
		t.Error("expected " + expected)
	}

	// now try illegal path:
	routeData.URLTail = "/some-files/../../gotcha.txt"
	p, err = v0.getFilePath(routeData)
	if err == nil {
		t.Error("expected error, got " + p)
	}

	// now try a specific file for route path
	routeData.URLTail = "/some-files/css/style.css"
	routeData.RouteConfig.Path = "/some-files/css/style.css"
	routeData.RouteConfig.Handler.Path = "@app/static-files/style.css"
	p, err = v0.getFilePath(routeData)
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
	routeData := &domain.AppspaceRouteData{
		AppVersion: &domain.AppVersion{
			LocationKey: "app-version-123",
		},
		RouteConfig: &domain.AppspaceRouteConfig{
			Path: "/some-files",
			Handler: domain.AppspaceRouteHandler{
				Type: "file",
				Path: "@app/static-files/",
			},
		},
		URLTail: "/some-files/css/style.css",
	}

	v0 := &V0{
		Config: config,
	}

	p := filepath.Join(dir, "app-version-123", "static-files", "css")
	t.Log("writing css to: " + p)
	err = os.MkdirAll(p, 0755)
	if err != nil {
		t.Error(err)
	}
	fileData := []byte("BODY { color: red; }")
	ioutil.WriteFile(filepath.Join(p, "style.css"), fileData, 0644)

	req, err := http.NewRequest("GET", "/some-files/css/style.css", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	v0.serveFile(rr, req, routeData)

	respString := string(rr.Body.Bytes())
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
	routeData := &domain.AppspaceRouteData{
		AppVersion: &domain.AppVersion{
			LocationKey: "app-version-123",
		},
		RouteConfig: &domain.AppspaceRouteConfig{
			Path: "/",
			Handler: domain.AppspaceRouteHandler{
				Type: "file",
				Path: "@app/static-files/index.html",
			},
		},
		URLTail: "/favicon.ico",
	}

	v0 := &V0{
		Config: config,
	}

	p := filepath.Join(dir, "app-version-123", "static-files")
	t.Log("writing html to: " + p)
	err = os.MkdirAll(p, 0755)
	if err != nil {
		t.Error(err)
	}
	fileData := []byte("<h1>hello world</h1")
	ioutil.WriteFile(filepath.Join(p, "index.html"), fileData, 0644)

	req, err := http.NewRequest("GET", "/favicon.ico", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	v0.serveFile(rr, req, routeData)

	if rr.Result().StatusCode != http.StatusNotFound {
		t.Error("expected 404")
	}
}

// path is dropserver, does it forward to dropserver route?
func TestServeHTTPDropserverRoute(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dropserverRoutes := domain.NewMockRouteHandler(mockCtrl)

	v0 := &V0{
		DropserverRoutes: dropserverRoutes}

	routeData := &domain.AppspaceRouteData{
		URLTail: "/dropserver",
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	dropserverRoutes.EXPECT().ServeHTTP(gomock.Any(), req, routeData)

	v0.ServeHTTP(rr, req, routeData)
}

// with appspace and route assume route is proxy
// -> calls proxy
func TestServeHTTPProxyRoute(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)
	//proxyID := domain.ProxyID("abc")

	routesV0 := domain.NewMockV0RouteModel(mockCtrl)
	routesV0.EXPECT().Match("GET", "/abc").Return(&domain.AppspaceRouteConfig{
		Handler: domain.AppspaceRouteHandler{Type: "function"},
		Auth:    domain.AppspaceRouteAuth{Allow: "public"},
	}, nil)
	asRoutesModel := domain.NewMockAppspaceRouteModels(mockCtrl)
	asRoutesModel.EXPECT().GetV0(appspaceID).Return(routesV0)
	// asUserModel := testmocks.NewMockAppspaceUserModel(mockCtrl)
	// asUserModel.EXPECT().Get(appspaceID, proxyID).Return(domain.AppspaceUser{AppspaceID: appspaceID, ProxyID: proxyID}, nil)
	sandboxProxy := domain.NewMockRouteHandler(mockCtrl)

	v0 := &V0{
		AppspaceRouteModels: asRoutesModel,
		//AppspaceUserModel:   asUserModel,
		SandboxProxy: sandboxProxy}

	routeData := &domain.AppspaceRouteData{
		URLTail: "/abc",
		Appspace: &domain.Appspace{
			AppspaceID: appspaceID,
		},
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	sandboxProxy.EXPECT().ServeHTTP(gomock.Any(), req, routeData)

	// ^^ here we are checking against routeData, which is a pointer
	// so it's not testing whether the call populated routeData correctly.

	v0.ServeHTTP(rr, req, routeData)

	// TODO: check routeData was properly augmented (app, appspace)
}