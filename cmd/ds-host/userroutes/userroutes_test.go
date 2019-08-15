package userroutes

import (
	"io/ioutil"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/golang/mock/gomock"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestUserData(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(1)

	um := domain.NewMockUserModel(mockCtrl)
	um.EXPECT().GetFromID(uid).Return(&domain.User{
		UserID: uid,
		Email: "abc@def"}, nil)

	u := UserRoutes{
		UserModel: um }

	routeData := domain.AppspaceRouteData{
		Cookie: &domain.Cookie{
			UserID: uid,
		},
	}

	rr := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	u.userData(rr, req, &routeData)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body, err := ioutil.ReadAll(rr.Body)
    if err != nil {
        t.Error(err)
	}
	
	var uData struct{
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