package appspacerouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
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

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)

	appspaceRoutes := &AppspaceRouter{
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

	appspaceID := domain.AppspaceID(7)
	appID := domain.AppID(11)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceStatus := testmocks.NewMockAppspaceStatus(mockCtrl)

	appspaceRoutes := &AppspaceRouter{
		AppModel:       appModel,
		AppspaceModel:  appspaceModel,
		AppspaceStatus: appspaceStatus}
	appspaceRoutes.Init()

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/abc",
		Subdomains: &[]string{"as1"},
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	appspaceModel.EXPECT().GetFromSubdomain("as1").Return(&domain.Appspace{
		AppspaceID: appspaceID,
		Subdomain:  "as1",
		AppID:      appID}, nil)
	appModel.EXPECT().GetFromID(appID).Return(nil, dserror.New(dserror.InternalError))
	appspaceStatus.EXPECT().Ready(appspaceID).Return(true)

	appspaceRoutes.ServeHTTP(rr, req, routeData)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", rr.Code)
	}
}

// need to test events

// then need to test a successful route call.
func TestEmitLiveCountNoop(t *testing.T) {
	appspaceRoutes := &AppspaceRouter{}
	appspaceRoutes.Init()

	appspaceID := domain.AppspaceID(7)

	appspaceRoutes.emitLiveCount(appspaceID, 11)
}

func TestEmitLiveCount(t *testing.T) {
	appspaceRoutes := &AppspaceRouter{}
	appspaceRoutes.Init()

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
	appspaceRoutes.Init()

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
	appspaceRoutes.Init()

	appspaceID := domain.AppspaceID(7)

	appspaceRoutes.incrementLiveCount(appspaceID)
	appspaceRoutes.incrementLiveCount(appspaceID)

	subChan := make(chan int)
	count := appspaceRoutes.SubscribeLiveCount(appspaceID, subChan)
	if count != 2 {
		t.Error("count should be two")
	}

	go func() {
		appspaceRoutes.decrementLiveCount(appspaceID)
		appspaceRoutes.decrementLiveCount(appspaceID)
	}()

	for count = range subChan {
		if count == 0 {
			appspaceRoutes.UnsubscribeLiveCount(appspaceID, subChan)
			close(subChan)
		}
	}
}

func getASR(mockCtrl *gomock.Controller) *AppspaceRouter {
	appModel := testmocks.NewMockAppModel(mockCtrl)
	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)

	return &AppspaceRouter{
		AppModel:      appModel,
		AppspaceModel: appspaceModel}
}
