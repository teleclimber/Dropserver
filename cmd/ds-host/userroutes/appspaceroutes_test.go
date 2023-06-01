package userroutes

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetAppspacesForApp(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userID := domain.UserID(7)
	appID := domain.AppID(11)

	am := testmocks.NewMockAppModel(mockCtrl)
	am.EXPECT().GetFromID(appID).Return(domain.App{OwnerID: userID}, nil)

	asm := testmocks.NewMockAppspaceModel(mockCtrl)
	asm.EXPECT().GetForApp(appID).Return([]*domain.Appspace{{DomainName: "appspace.sub.domain", AppID: appID, OwnerID: userID}}, nil)

	a := AppspaceRoutes{
		AppModel:      am,
		AppspaceModel: asm,
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/?app=%v", appID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), userID))

	rr := httptest.NewRecorder()

	a.getAppspaces(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Errorf("expected OK status, got %v", rr.Result().Status)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(rr.Result().Body)
	if !strings.Contains(buf.String(), "appspace.sub.domain") {
		t.Error("expected JSON response to contain the appspace domain")
	}
}

func TestGetAppspacesForAppForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userID := domain.UserID(7)
	appID := domain.AppID(11)

	am := testmocks.NewMockAppModel(mockCtrl)
	am.EXPECT().GetFromID(appID).Return(domain.App{OwnerID: domain.UserID(13)}, nil)

	a := AppspaceRoutes{
		AppModel: am,
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/?app=%v", appID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), userID))

	rr := httptest.NewRecorder()

	a.getAppspaces(rr, req)

	if rr.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected Forbidden status, got %v", rr.Result().Status)
	}
}
