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
