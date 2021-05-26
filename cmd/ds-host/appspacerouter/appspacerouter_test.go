package appspacerouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
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
func TestLoadAppspaceNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromDomain("as1.ds.dev").Return(nil, nil)

	appspaceRouter := &AppspaceRouter{
		AppspaceModel: appspaceModel}

	nextCalled := false
	handler := appspaceRouter.loadAppspace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "as1.ds.dev"

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rr.Code)
	}
	if nextCalled {
		t.Error("next got called when it should not have")
	}
}

func TestLoadAppspace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromDomain("as1.ds.dev").Return(&domain.Appspace{AppVersion: "1.2.3"}, nil)

	appspaceRouter := &AppspaceRouter{
		AppspaceModel: appspaceModel}

	nextCalled := false
	handler := appspaceRouter.loadAppspace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqAppspace, ok := domain.CtxAppspaceData(r.Context())
		if !ok {
			t.Error("no appspace on request")
		}
		if reqAppspace.AppVersion != "1.2.3" {
			t.Error("did not get the appspace data we expected")
		}
		nextCalled = true
	}))

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "as1.ds.dev"

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected OK, got %d", rr.Code)
	}
	if !nextCalled {
		t.Error("next did not get called")
	}
}

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
	appspaceRoutes.liveCounter = make(map[domain.AppspaceID]int)
	appspaceRoutes.subscribers = make(map[domain.AppspaceID][]chan<- int)

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
