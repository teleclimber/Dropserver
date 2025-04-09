package userroutes

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
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

	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().Return(false, nil)

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SetupKey:      sk,
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

	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().Return(false, nil)

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: false}, nil)

	im := testmocks.NewMockUserInvitationModel(mockCtrl)
	im.EXPECT().Get(email).Return(domain.UserInvitation{}, sql.ErrNoRows)

	a := &AuthRoutes{
		SetupKey:            sk,
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

	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().Return(false, nil)

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SetupKey:      sk,
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

	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().Return(false, nil)

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	a := &AuthRoutes{
		SetupKey:      sk,
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

	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().Return(false, nil)

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().CreateWithEmail(email, password).Return(domain.User{}, domain.ErrIdentifierExists)

	a := &AuthRoutes{
		SetupKey:      sk,
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

	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().Return(false, nil)

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: true}, nil)

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().CreateWithEmail(email, password).Return(domain.User{
		UserID: userID,
		Email:  email}, nil)

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().SetForAccount(gomock.Any(), userID)

	a := &AuthRoutes{
		SetupKey:      sk,
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

func TestSignupPostSetupKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	email := "oy@foo.bar"
	password := "password123"
	userID := domain.UserID(100)

	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().Return(true, nil)
	sk.EXPECT().Get().Return("abcdef", nil)
	sk.EXPECT().Delete().Return(nil)

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: false}, nil)

	userModel := testmocks.NewMockUserModel(mockCtrl)
	userModel.EXPECT().CreateWithEmail(email, password).Return(domain.User{
		UserID: userID,
		Email:  email}, nil)
	userModel.EXPECT().MakeAdmin(userID).Return(nil)

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().SetForAccount(gomock.Any(), userID)

	a := &AuthRoutes{
		SetupKey:      sk,
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

func TestSignupRoutesWithSetupKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	key := "abcdef"
	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().AnyTimes().DoAndReturn(func() (bool, error) {
		return key != "", nil
	})
	sk.EXPECT().Get().AnyTimes().DoAndReturn(func() (string, error) {
		return key, nil
	})

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any()).AnyTimes()

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().AnyTimes().Return(domain.Settings{RegistrationOpen: false}, nil)

	a := &AuthRoutes{
		SetupKey:      sk,
		SettingsModel: sm,
		Views:         views,
	}

	r := chi.NewRouter()
	r.Group(a.routeGroup)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := testRequest(t, ts, http.MethodGet, "/signup", nil)
	if resp.StatusCode != 404 {
		t.Error("expected 404 on /signup when setup key exists, got " + fmt.Sprint(resp.StatusCode))
	}
	resp, _ = testRequest(t, ts, http.MethodGet, "/"+key, nil)
	if resp.StatusCode != 200 {
		t.Error("expected 200 on /signup when setup key exists, got " + fmt.Sprint(resp.StatusCode))
	}

	// now let's "delete" the key and test these routes again:
	key = ""
	resp, _ = testRequest(t, ts, http.MethodGet, "/signup", nil)
	if resp.StatusCode != 200 {
		t.Error("expected 200 on /signup when setup key exists, got " + fmt.Sprint(resp.StatusCode))
	}
	resp, _ = testRequest(t, ts, http.MethodGet, "/"+key, nil)
	if resp.StatusCode != 404 {
		t.Error("expected 404 on /signup when setup key exists, got " + fmt.Sprint(resp.StatusCode))
	}
}

func TestSignupRoutesWithoutSetupKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sk := testmocks.NewMockSetupKey(mockCtrl)
	sk.EXPECT().Has().AnyTimes().Return(false, nil)
	sk.EXPECT().Get().AnyTimes().Return("", nil)

	views := testmocks.NewMockViews(mockCtrl)
	views.EXPECT().Signup(gomock.Any(), gomock.Any())

	sm := testmocks.NewMockSettingsModel(mockCtrl)
	sm.EXPECT().Get().Return(domain.Settings{RegistrationOpen: false}, nil)

	a := &AuthRoutes{
		SetupKey:      sk,
		SettingsModel: sm,
		Views:         views,
	}

	r := chi.NewRouter()
	r.Group(a.routeGroup)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := testRequest(t, ts, http.MethodGet, "/signup", nil)
	if resp.StatusCode != 200 {
		t.Error("expected 200 on /signup when setup key does not exist, got " + fmt.Sprint(resp.StatusCode))
	}
}

// borrowed from chi project: chi/middleware/middleware_test.go
func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}
