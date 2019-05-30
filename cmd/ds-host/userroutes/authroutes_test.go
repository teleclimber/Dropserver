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

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(dserror.New(dserror.InputValidationError))

	a := &AuthRoutes{
		Validator: validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.loginPost(rr, req, &domain.AppspaceRouteData{})

	// TODO: improve tests when we have more complete return data from this function
}

func TestLoginPostBadPassword(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(dserror.New(dserror.InputValidationError))

	a := &AuthRoutes{
		Validator: validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.loginPost(rr, req, &domain.AppspaceRouteData{})

	// TODO: improve tests when we have more complete return data from this function
}

func TestLoginPostNoRows(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(nil)

	userModel := domain.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmailPassword(gomock.Any(), gomock.Any()).Return(nil, dserror.New(dserror.NoRowsInResultSet))

	a := &AuthRoutes{
		Validator: validator,
		UserModel: userModel}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.loginPost(rr, req, &domain.AppspaceRouteData{})

	// TODO: improve tests when we have more complete return data from this function
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

// Test that we get correct response when pw validation fails
// ..lots of places where auth can fail, should really test all of them.
