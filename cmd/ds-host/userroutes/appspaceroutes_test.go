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

func TestGetAppspaceCtxForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appspaceUserID := domain.UserID(7)
	reqUserID := domain.UserID(13)
	appspaceID := domain.AppspaceID(11)

	asm := testmocks.NewMockAppspaceModel(mockCtrl)
	asm.EXPECT().GetFromID(appspaceID).Return(&domain.Appspace{AppspaceID: appspaceID, OwnerID: appspaceUserID}, nil)

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

	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), reqUserID))

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected Forbidden got status %v", rr.Result().Status)
	}
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
