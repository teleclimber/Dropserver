package userapi

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestGetAppspaceRoute(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().Authenticate(gomock.Any()).Return(domain.Authentication{Authenticated: true, UserID: ownerID})
	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspace1.AppspaceID).Return(&appspace1, nil)
	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersion(appVersion1.AppID, appVersion1.Version).Return(&appVersion1, nil)

	api := UserJSONAPI{
		Auth:          authenticator,
		AppspaceModel: appspaceModel,
		AppModel:      appModel,
	}
	api.Init()

	resp, payload := apiReq(&api, "GET", "/appspaces/21?include=app_version", "")

	if resp.StatusCode != http.StatusOK {
		t.Error("expected code 200")
	}

	payloadContains(t, payload, "subdomain-one", "app-version-one")

}

func TestGetAppspacesRoute(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().Authenticate(gomock.Any()).Return(domain.Authentication{Authenticated: true, UserID: ownerID})
	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetForOwner(ownerID).Return([]*domain.Appspace{&appspace1, &appspace2, &appspace3}, nil)
	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersion(appVersion1.AppID, appVersion1.Version).Return(&appVersion1, nil)
	appModel.EXPECT().GetVersion(appVersion3.AppID, appVersion3.Version).Return(&appVersion3, nil)

	api := UserJSONAPI{
		Auth:          authenticator,
		AppspaceModel: appspaceModel,
		AppModel:      appModel,
	}
	api.Init()

	resp, payload := apiReq(&api, "GET", "/appspaces?filter=owner&include=app_version", "")

	if resp.StatusCode != http.StatusOK {
		t.Error("expected code 200")
	}

	payloadContains(t, payload, "subdomain-one", "subdomain-two", "subdomain-three", "app-version-one", "app-version-three")
}

func TestPauseAppspaceRoute(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().Authenticate(gomock.Any()).Return(domain.Authentication{Authenticated: true, UserID: ownerID}).Times(2)
	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetFromID(appspace1.AppspaceID).Return(&appspace1, nil).Times(2)
	appspaceModel.EXPECT().Pause(appspace1.AppspaceID, true).Return(nil)
	appspaceModel.EXPECT().Pause(appspace1.AppspaceID, false).Return(nil)

	api := UserJSONAPI{
		Auth:          authenticator,
		AppspaceModel: appspaceModel,
	}
	api.Init()

	resp, _ := apiReq(&api, "PATCH", "/appspaces/21", `{"data":{"id":"21","type":"appspaces","attributes":{"paused":true}}}`)
	if resp.StatusCode != http.StatusOK {
		t.Error("expected code 200")
	}

	resp, _ = apiReq(&api, "PATCH", "/appspaces/21", `{"data":{"id":"21","type":"appspaces","attributes":{"paused":false}}}`)
	if resp.StatusCode != http.StatusOK {
		t.Error("expected code 200")
	}
}
