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
	"github.com/teleclimber/DropServer/internal/dserror"
)

// login POST handling

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

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmailPassword(gomock.Any(), gomock.Any()).Return(domain.User{}, sql.ErrNoRows)

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

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().GetFromEmailPassword(gomock.Any(), gomock.Any()).Return(domain.User{
		UserID: userID,
		Email:  email}, nil)

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
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
		URLTail: "/abc",
	}

	a.loginPost(rr, req, routeData)

	if rr.Code != http.StatusFound {
		t.Error("wrong status", rr.Code)
	}
}

// Signup post handling

func TestSignupPostBadEmail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(dserror.New(dserror.InputValidationError))

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SettingsModel: sm,
		Views:         views,
		Validator:     validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req, &domain.AppspaceRouteData{})
}

func TestSignupPostNotInvited(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: false}, nil)

	im := domain.NewMockUserInvitationModel(mockCtrl)
	im.EXPECT().Get(email).Return(nil, dserror.New(dserror.NoRowsInResultSet))

	a := &AuthRoutes{
		SettingsModel:       sm,
		UserInvitationModel: im,
		Views:               views,
		Validator:           validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req, &domain.AppspaceRouteData{})
}

func TestSignupPostBadPassword(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(dserror.New(dserror.InputValidationError))

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SettingsModel: sm,
		Views:         views,
		Validator:     validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req, &domain.AppspaceRouteData{})
}

func TestSignupPostPasswordMismatch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(nil)

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SettingsModel: sm,
		Views:         views,
		Validator:     validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	form.Add("password2", password+"zzz")
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req, &domain.AppspaceRouteData{})
}

func TestSignupPostEmailExists(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(nil)

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().Create(email, password).Return(domain.User{}, usermodel.ErrEmailExists)

	a := &AuthRoutes{
		Views:         views,
		SettingsModel: sm,
		UserModel:     userModel,
		Validator:     validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	form.Add("password2", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req, &domain.AppspaceRouteData{})
}

func TestSignupPost(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"
	userID := domain.UserID(100)

	validator := domain.NewMockValidator(mockCtrl)
	validator.EXPECT().Email(email).Return(nil)
	validator.EXPECT().Password(password).Return(nil)

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
		Authenticator: authenticator,
		Validator:     validator}

	rr := httptest.NewRecorder()

	form := url.Values{}
	form.Add("email", email)
	form.Add("password", password)
	form.Add("password2", password)
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	a.postSignup(rr, req, &domain.AppspaceRouteData{})
}

// reg closed email in
// reg closed email out

// routes
func TestGetLoginRoute(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Login(gomock.Any(), gomock.Any())

	a := &AuthRoutes{
		Views: views}

	req := httptest.NewRequest("GET", "/login", nil)

	rr := httptest.NewRecorder()

	a.ServeHTTP(rr, req, &domain.AppspaceRouteData{URLTail: "login"})
}

func TestGetSignupRoute(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	views := domain.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SettingsModel: sm,
		Views:         views}

	req := httptest.NewRequest("GET", "/signup", nil)

	rr := httptest.NewRecorder()

	a.ServeHTTP(rr, req, &domain.AppspaceRouteData{URLTail: "signup"})
}

// could test post routes but that involves setting up as much as for post handlers above.
