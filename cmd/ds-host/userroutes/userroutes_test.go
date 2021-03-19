package userroutes

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestIndex(t *testing.T) {
	u := UserRoutes{}

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	routeData := domain.AppspaceRouteData{
		URLTail: "/",
		Authentication: &domain.Authentication{
			Authenticated: true,
			UserAccount:   true,
		},
	}

	u.ServeHTTP(rr, req, &routeData)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Error(err)
	}

	if !strings.Contains(string(body), "<!DOCTYPE html>") {
		t.Error("body does nto contain <!DOCTYPE html>")
	}
}

func TestUserData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(1)

	um := testmocks.NewMockUserModel(mockCtrl)
	um.EXPECT().GetFromID(uid).Return(domain.User{
		UserID: uid,
		Email:  "abc@def"}, nil)
	um.EXPECT().IsAdmin(uid).Return(false)

	u := UserRoutes{
		UserModel: um}

	routeData := domain.AppspaceRouteData{
		Authentication: &domain.Authentication{
			UserID: uid,
		},
	}

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	u.getUserData(rr, req, &routeData)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Error(err)
	}

	var uData struct {
		Email string `json:"email"`
	}
	err = json.Unmarshal(body, &uData)
	if err != nil {
		t.Error(err)
	}

	if uData.Email != "abc@def" {
		t.Error("didn't get the email back")
	}
}

func TestChangePassword(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(1)

	um := testmocks.NewMockUserModel(mockCtrl)
	um.EXPECT().GetFromID(uid).Return(domain.User{
		UserID: uid,
		Email:  "abc@def"}, nil)
	um.EXPECT().GetFromEmailPassword("abc@def", "secretsauce").Return(domain.User{}, nil)
	um.EXPECT().UpdatePassword(uid, "secretspice").Return(nil)

	u := UserRoutes{
		UserModel: um}

	routeData := domain.AppspaceRouteData{
		Authentication: &domain.Authentication{
			UserID: uid,
		},
	}

	rr := httptest.NewRecorder()

	// craft json request
	jsonStr := []byte(`{"old":"secretsauce", "new":"secretspice"}`)
	req, err := http.NewRequest("GET", "/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	u.changeUserPassword(rr, req, &routeData)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

}
