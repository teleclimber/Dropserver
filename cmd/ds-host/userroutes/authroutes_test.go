package userroutes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

func TestLoginPostBadEmail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Login(gomock.Any(), gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(dserror.New(dserror.InputValidationError))

	a := &AuthRoutes{
		Views:     views,
		Validator: validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.loginPost(rr, req, &domain.AppspaceRouteData{})
}

func TestLoginPostBadPassword(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Login(gomock.Any(), gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(dserror.New(dserror.InputValidationError))

	a := &AuthRoutes{
		Views:     views,
		Validator: validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.loginPost(rr, req, &domain.AppspaceRouteData{})
}

func TestLoginPostNoRows(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Login(gomock.Any(), gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(nil)

	userModel := domain.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmailPassword(gomock.Any(), gomock.Any()).Return(nil, dserror.New(dserror.NoRowsInResultSet))

	a := &AuthRoutes{
		Views:     views,
		Validator: validator,
		UserModel: userModel}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.loginPost(rr, req, &domain.AppspaceRouteData{})
}

func TestLoginPost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"
	userID := domain.UserID(1)

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(nil)

	userModel := domain.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmailPassword(gomock.Any(), gomock.Any()).Return(&domain.User{
		UserID: userID,
		Email:  email}, nil)

	authenticator := domain.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().SetForAccount(gomock.Any(), userID)

	a := &AuthRoutes{
		Validator:     validator,
		Authenticator: authenticator,
		UserModel:     userModel}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	routeData := &domain.AppspaceRouteData{
		URLTail:    "/abc",
		Subdomains: &[]string{"as1"},
	}

	a.loginPost(rr, req, routeData)

	if rr.Code != http.StatusMovedPermanently {
		t.Error("wrong status", rr.Code)
	}
}
