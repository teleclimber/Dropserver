package userroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestMustBeAdminNoAuth(t *testing.T) {
	a := AdminRoutes{}
	router := chi.NewMux()
	router.Use(a.mustBeAdmin)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected Unauthorized got status %v", rr.Result().Status)
	}
}

func TestMustBeAdminForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reqUid := domain.UserID(7)

	um := testmocks.NewMockUserModel(mockCtrl)
	um.EXPECT().IsAdmin(reqUid).Return(false)
	a := AdminRoutes{
		UserModel: um}

	router := chi.NewMux()
	router.Use(a.mustBeAdmin)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {})

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(domain.CtxWithAuthUserID(req.Context(), reqUid))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected Forbidden got status %v", rr.Result().Status)
	}
}
