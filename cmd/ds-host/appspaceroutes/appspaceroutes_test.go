package appspaceroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// test ServeHTTP
// - gets appspace, fails if not there
// - recognizes /dropserver/ as path and forwards accordingly
// - gets app and fails if none
// - ...

// inputs to function: res, req, routeData
// - res:
// - req: Host? (not used directly)
// - routeData:
//   - subdomains -> appspace name
//   - urlTail  -> whether it's /dropserver

// start with one test: subdomain has an unknown appspace
// That's a 404.
func TestServeHTTPBadAppspace(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appModel := domain.NewMockAppModel(mockCtrl)
	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)
	dropserverRoutes := domain.NewMockRouteHandler(mockCtrl)
	sandboxProxy := domain.NewMockRouteHandler(mockCtrl)
	logger := domain.NewMockLogCLientI(mockCtrl)

	appspaceRoutes := &AppspaceRoutes{
		AppModel:         appModel,
		AppspaceModel:    appspaceModel,
		DropserverRoutes: dropserverRoutes,
		SandboxProxy:     sandboxProxy,
		Logger:           logger}

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/abc",
		Subdomains: &[]string{"as1"},
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	appspaceModel.EXPECT().GetFromSubdomain("as1").Return(nil, dserror.New(dserror.NoRowsInResultSet))
	logger.EXPECT().Log(domain.ERROR, gomock.Any(), gomock.Any())

	appspaceRoutes.ServeHTTP(rr, req, routeData)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rr.Code)
	}
}

// path is dropserver, does it forward to dropserver route?
func TestServeHTTPDropserverRoute(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appModel := domain.NewMockAppModel(mockCtrl)
	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)
	dropserverRoutes := domain.NewMockRouteHandler(mockCtrl)
	sandboxProxy := domain.NewMockRouteHandler(mockCtrl)
	logger := domain.NewMockLogCLientI(mockCtrl)

	appspaceRoutes := &AppspaceRoutes{
		AppModel:         appModel,
		AppspaceModel:    appspaceModel,
		DropserverRoutes: dropserverRoutes,
		SandboxProxy:     sandboxProxy,
		Logger:           logger}

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/dropserver",
		Subdomains: &[]string{"as1"},
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	appspaceModel.EXPECT().GetFromSubdomain("as1").Return(&domain.Appspace{Subdomain: "as1", AppID: domain.AppID(1)}, nil)
	dropserverRoutes.EXPECT().ServeHTTP(rr, req, routeData)

	appspaceRoutes.ServeHTTP(rr, req, routeData)
}

// Somehow we can't find the app referred to by appspace.
// server 500 and log error
func TestServeHTTPBadApp(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appModel := domain.NewMockAppModel(mockCtrl)
	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)
	dropserverRoutes := domain.NewMockRouteHandler(mockCtrl)
	sandboxProxy := domain.NewMockRouteHandler(mockCtrl)
	logger := domain.NewMockLogCLientI(mockCtrl)

	appspaceRoutes := &AppspaceRoutes{
		AppModel:         appModel,
		AppspaceModel:    appspaceModel,
		DropserverRoutes: dropserverRoutes,
		SandboxProxy:     sandboxProxy,
		Logger:           logger}

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/abc",
		Subdomains: &[]string{"as1"},
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	appspaceModel.EXPECT().GetFromSubdomain("as1").Return(&domain.Appspace{Subdomain: "as1", AppID: domain.AppID(1)}, nil)
	appModel.EXPECT().GetFromID(gomock.Any()).Return(nil, dserror.New(dserror.InternalError))
	logger.EXPECT().Log(domain.ERROR, gomock.Any(), gomock.Any())

	appspaceRoutes.ServeHTTP(rr, req, routeData)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", rr.Code)
	}
}

// with appspace and route assume route is proxy
// -> calls proxy
func TestServeHTTPProxyRoute(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appModel := domain.NewMockAppModel(mockCtrl)
	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)
	asRoutesModel := domain.NewMockASRoutesModel(mockCtrl)
	dropserverRoutes := domain.NewMockRouteHandler(mockCtrl)
	sandboxProxy := domain.NewMockRouteHandler(mockCtrl)
	logger := domain.NewMockLogCLientI(mockCtrl)

	appspaceRoutes := &AppspaceRoutes{
		AppModel:         appModel,
		AppspaceModel:    appspaceModel,
		ASRoutesModel:    asRoutesModel,
		DropserverRoutes: dropserverRoutes,
		SandboxProxy:     sandboxProxy,
		Logger:           logger}

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/abc",
		Subdomains: &[]string{"as1"},
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	appspaceModel.EXPECT().GetFromSubdomain("as1").Return(&domain.Appspace{Subdomain: "as1", AppID: domain.AppID(1)}, nil)
	appModel.EXPECT().GetFromID(gomock.Any()).Return(&domain.App{Name: "app1"}, nil)
	asRoutesModel.EXPECT().GetRouteConfig(gomock.Any(), "GET", "/abc").Return(&domain.RouteConfig{Type: "function"}, nil)
	sandboxProxy.EXPECT().ServeHTTP(rr, req, routeData)

	// ^^ here we are checking against routeData, which is a pointer
	// so it's not testing whether the call populated routeData correctly.

	appspaceRoutes.ServeHTTP(rr, req, routeData)

	// TODO: check routeData was properly augmented (app, appspace)
}

func getASR(mockCtrl *gomock.Controller) *AppspaceRoutes {
	appModel := domain.NewMockAppModel(mockCtrl)
	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)
	dropserverRoutes := domain.NewMockRouteHandler(mockCtrl)
	sandboxProxy := domain.NewMockRouteHandler(mockCtrl)
	logger := domain.NewMockLogCLientI(mockCtrl)

	return &AppspaceRoutes{
		AppModel:         appModel,
		AppspaceModel:    appspaceModel,
		DropserverRoutes: dropserverRoutes,
		SandboxProxy:     sandboxProxy,
		Logger:           logger}
}
