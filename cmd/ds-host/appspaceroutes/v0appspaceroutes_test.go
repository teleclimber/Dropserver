package appspaceroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

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
