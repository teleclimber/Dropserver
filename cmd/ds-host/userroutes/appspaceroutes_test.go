package userroutes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestGetAppspaceCtx(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(7)
	appspaceID := domain.AppspaceID(11)

	asm := testmocks.NewMockAppspaceModel(mockCtrl)
	asm.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{AppspaceID: appspaceID, OwnerID: uid}, nil)

	a := AppspaceRoutes{
		AppspaceModel: asm,
	}

	router := chi.NewMux()
	router.With(a.appspaceCtx).Get("/{appspace}", func(w http.ResponseWriter, r *http.Request) {
		appspace, ok := domain.CtxAppspaceData(r.Context())
		if !ok {
			t.Error("expected appspace data")
		}
		if appspace.AppspaceID != appspaceID {
			t.Error("did not get the app data expected")
		}
	})

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/%v", appspaceID), nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), uid))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK got status %v", rr.Result().Status)
	}
}

func TestRouterAppspaceOwnerAuthorization(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceOwnerID := domain.UserID(7)
	nonOwnerUserID := domain.UserID(13)
	appspaceID := domain.AppspaceID(11)

	asm := testmocks.NewMockAppspaceModel(mockCtrl)
	asm.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{AppspaceID: appspaceID, OwnerID: appspaceOwnerID}, nil).Times(2)

	das := testmocks.NewMockDeleteAppspace(mockCtrl)
	das.EXPECT().Delete(gomock.Any()).Return(nil)

	a := AppspaceRoutes{
		AppspaceModel:         asm,
		DeleteAppspace:        das,
		AppspaceUserRoutes:    &mockSubRoutes{},
		AppspaceExportRoutes:  &mockSubRoutes{},
		AppspaceRestoreRoutes: &mockSubRoutes{},
	}

	// Test forbidden access for non-owner
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/%v", appspaceID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), nonOwnerUserID))

	rr := httptest.NewRecorder()

	router := a.subRouter()
	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected Forbidden for non-owner, got status %v", rr.Result().Status)
	}

	// Test authorized access for owner
	req, err = http.NewRequest(http.MethodDelete, fmt.Sprintf("/%v", appspaceID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), appspaceOwnerID))

	rr = httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK for owner, got status %v", rr.Result().Status)
	}
}

func TestUserIsAppspaceUserOrOwner(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceOwnerID := domain.UserID(7)
	authorizedUserID := domain.UserID(13)
	unauthorizedUserID := domain.UserID(99)
	appspaceID := domain.AppspaceID(11)
	appID := domain.AppID(5)
	appVersion := domain.Version("1.0.0")

	asm := testmocks.NewMockAppspaceModel(mockCtrl)
	asm.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{
		AppspaceID: appspaceID,
		OwnerID:    appspaceOwnerID,
		AppID:      appID,
		AppVersion: appVersion,
	}, nil).Times(3)

	mu := testmocks.NewMockManageUsers(mockCtrl)
	// Authorized user exists
	mu.EXPECT().GetConflictsForUserID(appspaceID, authorizedUserID).Return(domain.UserIDProxyIDConflicts{}, nil)
	// Unauthorized user does not exist
	mu.EXPECT().GetConflictsForUserID(appspaceID, unauthorizedUserID).Return(domain.UserIDProxyIDConflicts{}, domain.ErrNoRowsInResultSet)

	am := testmocks.NewMockAppModel(mockCtrl)
	am.EXPECT().GetVersion(appID, appVersion).Return(domain.AppVersion{LocationKey: "test-location"}, nil).Times(2)

	afm := testmocks.NewMockAppFilesModel(mockCtrl)
	afm.EXPECT().GetLinkPath("test-location", "app-icon").Return("/path/to/icon.png").Times(2)

	a := AppspaceRoutes{
		AppspaceModel:         asm,
		ManageUsers:           mu,
		AppModel:              am,
		AppFilesModel:         afm,
		AppspaceUserRoutes:    &mockSubRoutes{},
		AppspaceExportRoutes:  &mockSubRoutes{},
		AppspaceRestoreRoutes: &mockSubRoutes{},
	}

	router := a.subRouter()

	// Test 1: Owner has access
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/%v/app-icon", appspaceID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), appspaceOwnerID))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode == http.StatusForbidden {
		t.Errorf("expected owner to have access, got Forbidden")
	}

	// Test 2: Authorized user has access
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("/%v/app-icon", appspaceID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), authorizedUserID))

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode == http.StatusForbidden {
		t.Errorf("expected authorized user to have access, got Forbidden")
	}

	// Test 3: Unauthorized user is forbidden
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("/%v/app-icon", appspaceID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), unauthorizedUserID))

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected unauthorized user to be forbidden, got status %v", rr.Result().Status)
	}
}

type mockSubRoutes struct{}

func (m *mockSubRoutes) subRouter() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

func TestAppspaceInIDs(t *testing.T) {
	ids := []domain.AppspaceUserIDs{
		{
			UserID:     domain.UserID(1),
			AppspaceID: domain.AppspaceID(5),
			ProxyID:    domain.ProxyID("abc"),
		},
		{
			UserID:     domain.UserID(2),
			AppspaceID: domain.AppspaceID(11),
			ProxyID:    domain.ProxyID("def"),
		},
		{
			UserID:     domain.UserID(3),
			AppspaceID: domain.AppspaceID(20),
			ProxyID:    domain.ProxyID("ghi"),
		},
	}

	tests := []struct {
		name        string
		appspaceID  domain.AppspaceID
		ids         []domain.AppspaceUserIDs
		expectedRes bool
	}{
		{
			name:        "found in middle",
			appspaceID:  domain.AppspaceID(11),
			ids:         ids,
			expectedRes: true,
		},
		{
			name:        "found at beginning",
			appspaceID:  domain.AppspaceID(5),
			ids:         ids,
			expectedRes: true,
		},
		{
			name:        "found at end",
			appspaceID:  domain.AppspaceID(20),
			ids:         ids,
			expectedRes: true,
		},
		{
			name:        "not found",
			appspaceID:  domain.AppspaceID(99),
			ids:         ids,
			expectedRes: false,
		},
		{
			name:        "empty slice",
			appspaceID:  domain.AppspaceID(11),
			ids:         []domain.AppspaceUserIDs{},
			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appspaceInIDs(tt.appspaceID, tt.ids)
			if result != tt.expectedRes {
				t.Errorf("appspaceInIDs() = %v, want %v", result, tt.expectedRes)
			}
		})
	}
}

func appspaceInIDs(appspaceID domain.AppspaceID, ids []domain.AppspaceUserIDs) bool {
	for _, id := range ids {
		if id.AppspaceID == appspaceID {
			return true
		}
	}
	return false
}
