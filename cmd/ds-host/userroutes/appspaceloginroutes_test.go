package userroutes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestOwnerLogin(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ownerID := domain.UserID(11)
	appspaceID := domain.AppspaceID(7)
	appspaceDomain := "some.appspace.com"
	dropID := "dropid.domain.com/alice"
	token := "abcdefghiabcdefghiabcdefghi"

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromDomain(appspaceDomain).Return(&domain.Appspace{
		AppspaceID: appspaceID,
		OwnerID:    ownerID,
		DropID:     dropID}, nil)

	v0tokenManager := testmocks.NewMockV0TokenManager(mockCtrl)
	v0tokenManager.EXPECT().GetForOwner(appspaceID, dropID).Return(token)

	m := &AppspaceLoginRoutes{
		Config:         &domain.RuntimeConfig{},
		AppspaceModel:  appspaceModel,
		V0TokenManager: v0tokenManager,
	}

	query := make(url.Values)
	query.Add("appspace", appspaceDomain)
	req, err := http.NewRequest(http.MethodGet, "?"+query.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	routeData := &domain.AppspaceRouteData{
		Authentication: &domain.Authentication{
			UserID: ownerID}}

	m.getTokenForRedirect(rr, req, routeData)

	if rr.Result().StatusCode != http.StatusTemporaryRedirect {
		t.Error("expected redirect")
	}
	location := rr.Result().Header.Get("Location")
	if !strings.Contains(location, appspaceDomain) {
		t.Error("expected appspace domain in Location")
	}
	if !strings.Contains(location, token) {
		t.Error("expected redirect token in Location")
	}
}

func TestRemoteLogin(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userID := domain.UserID(11)
	appspaceDomain := "some.appspace.com"
	sessionID := "session-id-abc"
	token := "abcdefghiabcdefghiabcdefghi"

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromDomain(appspaceDomain).Return(nil, nil)

	ds2ds := testmocks.NewMockDS2DS(mockCtrl)
	ds2ds.EXPECT().GetRemoteAPIVersion(appspaceDomain).Return(0, nil)

	v0requestToken := testmocks.NewMockV0RequestToken(mockCtrl)
	v0requestToken.EXPECT().RequestToken(context.Background(), userID, appspaceDomain, sessionID).Return(token, nil)

	m := &AppspaceLoginRoutes{
		Config:         &domain.RuntimeConfig{},
		AppspaceModel:  appspaceModel,
		V0RequestToken: v0requestToken,
		DS2DS:          ds2ds,
	}

	query := make(url.Values)
	query.Add("appspace", appspaceDomain)
	req, err := http.NewRequest(http.MethodGet, "?"+query.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	routeData := &domain.AppspaceRouteData{
		Authentication: &domain.Authentication{
			UserID:   userID,
			CookieID: sessionID}}

	m.getTokenForRedirect(rr, req, routeData)

	if rr.Result().StatusCode != http.StatusTemporaryRedirect {
		t.Error("expected redirect")
	}
	location := rr.Result().Header.Get("Location")
	if !strings.Contains(location, appspaceDomain) {
		t.Error("expected appspace domain in Location")
	}
	if !strings.Contains(location, token) {
		t.Error("expected redirect token in Location")
	}
}
