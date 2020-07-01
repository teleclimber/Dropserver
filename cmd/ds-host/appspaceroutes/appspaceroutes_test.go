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

	appspaceRoutes := &AppspaceRoutes{
		AppModel:      appModel,
		AppspaceModel: appspaceModel}

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

	appspaceRoutes.ServeHTTP(rr, req, routeData)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rr.Code)
	}
}

// Somehow we can't find the app referred to by appspace.
// server 500 and log error
func TestServeHTTPBadApp(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appModel := domain.NewMockAppModel(mockCtrl)
	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)

	appspaceRoutes := &AppspaceRoutes{
		AppModel:      appModel,
		AppspaceModel: appspaceModel}

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

	appspaceRoutes.ServeHTTP(rr, req, routeData)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", rr.Code)
	}
}

func getASR(mockCtrl *gomock.Controller) *AppspaceRoutes {
	appModel := domain.NewMockAppModel(mockCtrl)
	appspaceModel := domain.NewMockAppspaceModel(mockCtrl)

	return &AppspaceRoutes{
		AppModel:      appModel,
		AppspaceModel: appspaceModel}
}
