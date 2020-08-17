package authenticator

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestSetCookie(t *testing.T) {
	a := &Authenticator{
		Config: getConfig()}

	rr := httptest.NewRecorder()

	cookieID := "abc"
	expires := time.Date(2019, time.Month(5), 29, 6, 2, 0, 0, time.UTC)
	a.setCookie(rr, cookieID, expires, "abc")

	sch, ok := rr.HeaderMap["Set-Cookie"]
	if !ok {
		t.Error("Set Cookie Header not set", rr.HeaderMap)
	}
	if !strings.HasPrefix(sch[0], "session_token=abc;") {
		t.Error("cookie not set correctly: " + sch[0])
	}
}

func TestRefreshCookie(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().UpdateExpires("abc", gomock.Any()).Return(nil)

	a := &Authenticator{
		Config:      getConfig(),
		CookieModel: cm}

	rr := httptest.NewRecorder()

	a.refreshCookie(rr, "abc")

	sch, ok := rr.HeaderMap["Set-Cookie"]
	if !ok {
		t.Error("Set Cookie Header not set", rr.HeaderMap)
	}
	if !strings.HasPrefix(sch[0], "session_token=abc;") {
		t.Error("cookie not set correctly: " + sch[0])
	}
}

// testing for GetForAccount
func TestGetForAccountNoCookie(t *testing.T) {

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	a := &Authenticator{
		Config: getConfig()}

	cookie, err := a.Authenticate(rr, req)
	if err != nil {
		t.Error(err)
	}
	if cookie != nil {
		t.Error("No cookie should be returned")
	}
}

func TestGetForAccountNoDBCookie(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:    "session_token",
		Value:   "abc",
		Expires: time.Now().Add(time.Hour),
	})

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Get("abc").Return(nil, nil)

	a := &Authenticator{
		Config:      getConfig(),
		CookieModel: cm}

	cookie, err := a.Authenticate(rr, req)
	if err != nil {
		t.Error(err)
	}
	if cookie != nil {
		t.Error("No cookie should be returned")
	}
}

// this test is not so revealing since our more strict interpretation of "Authenticate"
func TestGetForAccountNotUser(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:    "session_token",
		Value:   "abc",
		Expires: time.Now().Add(time.Hour),
	})

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Get("abc").Return(&domain.Cookie{
		CookieID:    "abc",
		UserID:      domain.UserID(1),
		Expires:     time.Now().Add(time.Hour),
		UserAccount: false,
	}, nil)
	//cm.EXPECT().UpdateExpires("abc", gomock.Any())

	a := &Authenticator{
		Config:      getConfig(),
		CookieModel: cm}

	cookie, err := a.Authenticate(rr, req)
	if err != nil {
		t.Error(err)
	}
	if cookie == nil {
		t.Error("cookie should not be nil")
	}
	if cookie.UserAccount {
		t.Error("cookie should not be for user account")
	}
}

func TestGetForAccountExpired(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:    "session_token",
		Value:   "abc",
		Expires: time.Now().Add(time.Hour),
	})

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Get("abc").Return(&domain.Cookie{
		CookieID:    "abc",
		UserID:      domain.UserID(1),
		Expires:     time.Now().Add(-time.Hour),
		UserAccount: true,
	}, nil)

	a := &Authenticator{
		Config:      getConfig(),
		CookieModel: cm}

	cookie, err := a.Authenticate(rr, req)
	if err != nil {
		t.Error(err)
	}
	if cookie != nil {
		t.Error("cookie should be nil")
	}
}

func TestAuthenticate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:    "session_token",
		Value:   "abc",
		Expires: time.Now().Add(time.Hour),
	})

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Get("abc").Return(&domain.Cookie{
		CookieID:    "abc",
		UserID:      domain.UserID(1),
		Expires:     time.Now().Add(time.Hour),
		UserAccount: true,
	}, nil)
	//cm.EXPECT().UpdateExpires("abc", gomock.Any()).Return(nil)

	a := &Authenticator{
		Config:      getConfig(),
		CookieModel: cm}

	routeData := &domain.AppspaceRouteData{}

	cookie, err := a.Authenticate(rr, req)
	if err != nil {
		t.Error("should not error")
	}
	if cookie == nil {
		t.Error("cookie should not be nil")
	}
	if cookie.CookieID != "abc" {
		t.Error("route data not as expected", routeData)
	}
}

// Can we parametrize this test?
// input:
// - req cookie
// - req cookie expires
// - cookieModel return cookie / error
// - whther to expect updateExpires
// output:
// - ok?
// - routeData.Cookie
// - response code
// ---> all in all, too many factors to parametrize.

func TestSetForAccount(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userID := domain.UserID(1)

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Create(gomock.Any()).Return("abc", nil)

	a := Authenticator{
		Config:      getConfig(),
		CookieModel: cm}

	rr := httptest.NewRecorder()

	dsErr := a.SetForAccount(rr, userID)
	if dsErr != nil {
		t.Error(dsErr)
	}
	sch, ok := rr.HeaderMap["Set-Cookie"]
	if !ok {
		t.Error("cookie not set?", rr.HeaderMap)
	}
	if !strings.HasPrefix(sch[0], "session_token=abc;") {
		t.Error("cookie not set correctly: " + sch[0])
	}
}

func TestUnsetForAccount(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Get(gomock.Any()).Return(&domain.Cookie{
		CookieID:    "abc123",
		UserAccount: true,
		Expires:     time.Now().Add(120 * time.Second)}, nil)
	cm.EXPECT().Delete("abc123")

	a := Authenticator{
		Config:      getConfig(),
		CookieModel: cm}

	rr := httptest.NewRecorder()

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "abc123", MaxAge: 120})

	a.UnsetForAccount(rr, req)
}

func getConfig() *domain.RuntimeConfig {
	rtc := domain.RuntimeConfig{}
	rtc.Server.Host = "dropserver.org"

	return &rtc
}
