package userapi

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestGetAppVersionRoute(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().Authenticate(gomock.Any()).Return(domain.Authentication{Authenticated: true, UserID: ownerID})

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetFromID(appID1).Return(&app1, nil)
	appModel.EXPECT().GetVersion(appID1, appVersion1.Version).Return(&appVersion1, nil)

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetForAppVersion(appID1, appVersion1.Version).Return([]*domain.Appspace{&appspace1, &appspace2}, nil)

	api := UserJSONAPI{
		Auth:          authenticator,
		AppspaceModel: appspaceModel,
		AppModel:      appModel,
	}
	api.Init()

	resp, payload := apiReq(&api, "GET", "/app_versions/11-1.1.1?include=appspaces", "")

	if resp.StatusCode != http.StatusOK {
		t.Error("expected code 200")
	}

	payloadContains(t, payload, "app-version-one", "subdomain-one", "subdomain-two")
}
