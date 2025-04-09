package appspacerouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

// start with one test: subdomain has an unknown appspace
// That's a 404.
func TestLoadAppspaceNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromDomain("as1.ds.dev").Return(nil, nil)

	router := &FromPublic{
		AppspaceModel: appspaceModel}

	nextCalled := false
	handler := router.loadAppspace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	router := &FromPublic{
		AppspaceModel: appspaceModel}

	nextCalled := false
	handler := router.loadAppspace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestLoginTokenNoToken(t *testing.T) {
	r := &FromPublic{}

	nextCalled := false
	handler := r.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	r := &FromPublic{}

	nextCalled := false
	handler := r.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	r := &FromPublic{
		V0TokenManager: v0TokenManager,
	}

	nextCalled := false
	handler := r.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	r := &FromPublic{
		V0TokenManager: v0TokenManager,
		Authenticator:  authenticator,
	}

	nextCalled := false
	handler := r.processLoginToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
