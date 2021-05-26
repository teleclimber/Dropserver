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

// TODO: need tests for appspace user auth

func TestSetCookie(t *testing.T) {
	a := &Authenticator{
		Config: getConfig()}

	rr := httptest.NewRecorder()

	cookieID := "abc"
	expires := time.Date(2019, time.Month(5), 29, 6, 2, 0, 0, time.UTC)
	a.setCookie(rr, cookieID, expires, "abc")

	sch, ok := rr.Result().Header["Set-Cookie"]
	if !ok {
		t.Error("Set Cookie Header not set", rr.Result().Header)
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

	sch, ok := rr.Result().Header["Set-Cookie"]
	if !ok {
		t.Error("Set Cookie Header not set", rr.Result().Header)
	}
	if !strings.HasPrefix(sch[0], "session_token=abc;") {
		t.Error("cookie not set correctly: " + sch[0])
	}
}

func TestAccountUserNoCookie(t *testing.T) {
	a := &Authenticator{
		Config: getConfig()}

	nextCalled := false
	handler := a.AccountUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, ok := domain.CtxAuthUserID(r.Context())
		if ok {
			t.Error("there should not be anuth user")
		}
		nextCalled = true
	}))

	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Error("should have 200 status")
	}
	if !nextCalled {
		t.Error("next was not called")
	}
}

func TestAccountUserNoDBCookie(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Get("abc").Return(nil, nil)

	a := &Authenticator{
		Config:      getConfig(),
		CookieModel: cm}

	nextCalled := false
	handler := a.AccountUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, ok := domain.CtxAuthUserID(r.Context())
		if ok {
			t.Error("there should not be anuth user")
		}
		nextCalled = true
	}))

	req, _ := http.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:    "session_token",
		Value:   "abc",
		Expires: time.Now().Add(time.Hour),
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Error("should have 200 status")
	}
	if !nextCalled {
		t.Error("next was not called")
	}
}

func TestAccountUserWithAppspaceCookie(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Get("abc").Return(&domain.Cookie{
		CookieID:    "abc",
		Expires:     time.Now().Add(time.Hour),
		UserAccount: false,
	}, nil)

	a := &Authenticator{
		Config:      getConfig(),
		CookieModel: cm}
	nextCalled := false
	handler := a.AccountUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, ok := domain.CtxAuthUserID(r.Context())
		if ok {
			t.Error("there should not be an auth user")
		}
		_, ok = domain.CtxSessionID(r.Context())
		if ok {
			t.Error("there should not be a cookie id")
		}

		nextCalled = true
	}))

	req, _ := http.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:    "session_token",
		Value:   "abc",
		Expires: time.Now().Add(time.Hour),
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Error("should have 200 status")
	}
	if !nextCalled {
		t.Error("next was not called")
	}
}

func TestAccountUserExpired(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

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

	nextCalled := false
	handler := a.AccountUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, ok := domain.CtxAuthUserID(r.Context())
		if ok {
			t.Error("there should not be anuth user")
		}
		nextCalled = true
	}))

	req, _ := http.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:    "session_token",
		Value:   "abc",
		Expires: time.Now().Add(time.Hour),
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Error("should have 200 status")
	}
	if !nextCalled {
		t.Error("next was not called")
	}
}

func TestAccountUser(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userID := domain.UserID(7)

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().Get("abc").Return(&domain.Cookie{
		CookieID:    "abc",
		UserID:      userID,
		Expires:     time.Now().Add(time.Hour),
		UserAccount: true,
	}, nil)

	a := &Authenticator{
		Config:      getConfig(),
		CookieModel: cm}
	nextCalled := false
	handler := a.AccountUser(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		reqUserID, ok := domain.CtxAuthUserID(r.Context())
		if !ok {
			t.Error("there should be an auth user")
		}
		if reqUserID != userID {
			t.Error("wrong user id in request context")
		}
		reqSessionID, ok := domain.CtxSessionID(r.Context())
		if !ok {
			t.Error("there should be a cookie id")
		}
		if reqSessionID != "abc" {
			t.Error("wrong cookie id")
		}
		nextCalled = true
	}))

	req, _ := http.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:    "session_token",
		Value:   "abc",
		Expires: time.Now().Add(time.Hour),
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Result().StatusCode != 200 {
		t.Error("should have 200 status")
	}
	if !nextCalled {
		t.Error("next was not called")
	}
}

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
	sch, ok := rr.Result().Header["Set-Cookie"]
	if !ok {
		t.Error("cookie not set?", rr.Result().Header)
	}
	if !strings.HasPrefix(sch[0], "session_token=abc;") {
		t.Error("cookie not set correctly: " + sch[0])
	}
}

func TestUnset(t *testing.T) {
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

	a.Unset(rr, req)
}

func getConfig() *domain.RuntimeConfig {
	rtc := domain.RuntimeConfig{}
	rtc.Server.Host = "dropserver.org"

	return &rtc
}
