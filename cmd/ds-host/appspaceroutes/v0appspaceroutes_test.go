package appspaceroutes

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

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

// path is dropserver, does it forward to dropserver route?
func TestServeHTTPDropserverRoute(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dropserverRoutes := domain.NewMockRouteHandler(mockCtrl)

	v0 := &V0{
		DropserverRoutes: dropserverRoutes}

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/dropserver",
		Subdomains: &[]string{"as1"},
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	dropserverRoutes.EXPECT().ServeHTTP(rr, req, routeData)

	v0.ServeHTTP(rr, req, routeData)
}

// with appspace and route assume route is proxy
// -> calls proxy
func TestServeHTTPProxyRoute(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceID := domain.AppspaceID(7)

	routesV0 := domain.NewMockRouteModelV0(mockCtrl)
	routesV0.EXPECT().Match("GET", "/abc").Return(&domain.AppspaceRouteConfig{
		Handler: domain.AppspaceRouteHandler{Type: "function"},
	}, nil)
	asRoutesModel := domain.NewMockAppspaceRouteModels(mockCtrl)
	asRoutesModel.EXPECT().GetV0(appspaceID).Return(routesV0)
	sandboxProxy := domain.NewMockRouteHandler(mockCtrl)

	v0 := &V0{
		AppspaceRouteModels: asRoutesModel,
		SandboxProxy:        sandboxProxy}

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/abc",
		Subdomains: &[]string{"as1"},
		Appspace: &domain.Appspace{
			AppspaceID: appspaceID,
		},
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	sandboxProxy.EXPECT().ServeHTTP(rr, req, routeData)

	// ^^ here we are checking against routeData, which is a pointer
	// so it's not testing whether the call populated routeData correctly.

	v0.ServeHTTP(rr, req, routeData)

	// TODO: check routeData was properly augmented (app, appspace)
}
