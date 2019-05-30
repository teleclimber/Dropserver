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
	a := &Authenticator{}

	rr := httptest.NewRecorder()

	cookieID := "abc"
	expires := time.Date(2019, time.Month(5), 29, 6, 2, 0, 0, time.UTC)
	a.setCookie(rr, cookieID, expires)

	sch, ok := rr.HeaderMap["Set-Cookie"]
	if !ok {
		t.Error("Set Cookie Header not set", rr.HeaderMap)
	}
	if !strings.HasPrefix(sch[0], "session_token=abc; Expires=") {
		t.Error("cookie not set correctly: " + sch[0])
	}
}

func TestRefreshCookie(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cm := domain.NewMockCookieModel(mockCtrl)
	cm.EXPECT().UpdateExpires("abc", gomock.Any()).Return(nil)

	a := &Authenticator{
		CookieModel: cm}

	rr := httptest.NewRecorder()

	a.refreshCookie(rr, "abc")

	sch, ok := rr.HeaderMap["Set-Cookie"]
	if !ok {
		t.Error("Set Cookie Header not set", rr.HeaderMap)
	}
	if !strings.HasPrefix(sch[0], "session_token=abc; Expires=") {
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

	a := &Authenticator{}

	ok := a.GetForAccount(rr, req, &domain.AppspaceRouteData{})
	if ok {
		t.Error("should Not be ok")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Error("status of response not as expected", rr)
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
		CookieModel: cm}

	ok := a.GetForAccount(rr, req, &domain.AppspaceRouteData{})
	if ok {
		t.Error("should not be ok")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Error("status of response not as expected", rr)
	}
}

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

	a := &Authenticator{
		CookieModel: cm}

	ok := a.GetForAccount(rr, req, &domain.AppspaceRouteData{})
	if ok {
		t.Error("should not be ok")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Error("status of response not as expected", rr)
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
		CookieModel: cm}

	ok := a.GetForAccount(rr, req, &domain.AppspaceRouteData{})
	if ok {
		t.Error("should not be ok")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Error("status of response not as expected", rr)
	}
}

func TestGetForAccount(t *testing.T) {
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
	cm.EXPECT().UpdateExpires("abc", gomock.Any()).Return(nil)

	a := &Authenticator{
		CookieModel: cm}

	routeData := &domain.AppspaceRouteData{}

	ok := a.GetForAccount(rr, req, routeData)
	if !ok {
		t.Error("should be ok")
	}

	if routeData.Cookie.CookieID != "abc" {
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
	if !strings.HasPrefix(sch[0], "session_token=abc; Expires=") {
		t.Error("cookie not set correctly: " + sch[0])
	}
}
