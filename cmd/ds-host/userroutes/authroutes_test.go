package userroutes

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/usermodel"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

// login POST handling

func TestLoginPostBadEmail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Login(gomock.Any(), gomock.Any())

	a := &AuthRoutes{
		Views: views}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postLogin(rr, req)
}

func TestLoginPostBadPassword(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password"

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Login(gomock.Any(), gomock.Any())

	a := &AuthRoutes{
		Views: views}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postLogin(rr, req)
}

func TestLoginPostNoRows(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Login(gomock.Any(), gomock.Any())

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmailPassword(gomock.Any(), gomock.Any()).Return(domain.User{}, sql.ErrNoRows)

	a := &AuthRoutes{
		Views:     views,
		UserModel: userModel}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postLogin(rr, req)
}

func TestLoginPost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"
	userID := domain.UserID(1)

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmailPassword(gomock.Any(), gomock.Any()).Return(domain.User{
		UserID: userID,
		Email:  email}, nil)

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().SetForAccount(gomock.Any(), userID)

	a := &AuthRoutes{
		Authenticator: authenticator,
		UserModel:     userModel}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postLogin(rr, req)

	if rr.Code != http.StatusFound {
		t.Error("wrong status", rr.Code)
	}
}

// Signup post handling

func TestSignupPostBadEmail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SettingsModel: sm,
		Views:         views}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req)
}

func TestSignupPostNotInvited(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: false}, nil)

	im := testmocks.NewMockUserInvitationModel(mockCtrl)
	im.EXPECT().Get(email).Return(domain.UserInvitation{}, sql.ErrNoRows)

	a := &AuthRoutes{
		SettingsModel:       sm,
		UserInvitationModel: im,
		Views:               views}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req)
}

func TestSignupPostBadPassword(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SettingsModel: sm,
		Views:         views}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req)
}

func TestSignupPostPasswordMismatch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SettingsModel: sm,
		Views:         views}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	form.Add("password2", password+"zzz")
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req)
}

func TestSignupPostEmailExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().Create(email, password).Return(domain.User{}, usermodel.ErrEmailExists)

	a := &AuthRoutes{
		Views:         views,
		SettingsModel: sm,
		UserModel:     userModel}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	form.Add("password2", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req)
}

func TestSignupPost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"
	userID := domain.UserID(100)

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().Create(email, password).Return(domain.User{
		UserID: userID,
		Email:  email}, nil)

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().SetForAccount(gomock.Any(), userID)

	a := &AuthRoutes{
		SettingsModel: sm,
		UserModel:     userModel,
		Authenticator: authenticator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	form.Add("password2", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req)
}
