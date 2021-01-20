package userapi

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestGetApp(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().Authenticate(gomock.Any()).Return(domain.Authentication{Authenticated: true, UserID: ownerID})

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetFromID(appID1).Return(&app1, nil)

	appModel.EXPECT().GetVersionsForApp(appID1).Return([]*domain.AppVersion{&appVersion1, &appVersion2}, nil)

	api := UserJSONAPI{
		Auth:     authenticator,
		AppModel: appModel,
	}
	api.Init()

	resp, payload := apiReq(&api, "GET", "/apps/11", "")

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected code 200, got %v", resp.StatusCode)
	}

	payloadContains(t, payload, "app-one", "11-1.1.1", "11-2.2.2")
	payloadNotContains(t, payload, "app-version-one", "app-version-two")
}

func TestGetApps(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().Authenticate(gomock.Any()).Return(domain.Authentication{Authenticated: true, UserID: ownerID})

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetForOwner(ownerID).Return([]*domain.App{&app1, &app2}, nil)
	appModel.EXPECT().GetVersionsForApp(appID1).Return([]*domain.AppVersion{&appVersion1, &appVersion2}, nil)
	appModel.EXPECT().GetVersionsForApp(appID2).Return([]*domain.AppVersion{}, nil)

	api := UserJSONAPI{
		Auth:     authenticator,
		AppModel: appModel,
	}
	api.Init()

	resp, payload := apiReq(&api, "GET", "/apps?filter=owner", "")

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected code 200, got %v", resp.StatusCode)
	}

	payloadContains(t, payload, "app-one", "app-two", "11-1.1.1", "11-2.2.2")
	payloadNotContains(t, payload, "app-version-one", "app-version-two")
}

func TestGetAppsIncVersions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
	authenticator.EXPECT().Authenticate(gomock.Any()).Return(domain.Authentication{Authenticated: true, UserID: ownerID})

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetForOwner(ownerID).Return([]*domain.App{&app1, &app2}, nil)
	appModel.EXPECT().GetVersionsForApp(appID1).Return([]*domain.AppVersion{&appVersion1, &appVersion2}, nil)
	appModel.EXPECT().GetVersionsForApp(appID2).Return([]*domain.AppVersion{}, nil)

	api := UserJSONAPI{
		Auth:     authenticator,
		AppModel: appModel,
	}
	api.Init()

	resp, payload := apiReq(&api, "GET", "/apps?filter=owner&include=versions", "")

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected code 200, got %v", resp.StatusCode)
	}

	payloadContains(t, payload, "app-one", "app-two", "app-version-one", "app-version-two")
}

// func TestGetAppsIncAppspaces(t *testing.T) {
// 	mockCtrl := gomock.NewController(t)
// 	defer mockCtrl.Finish()

// 	authenticator := testmocks.NewMockAuthenticator(mockCtrl)
// 	authenticator.EXPECT().Authenticate(gomock.Any()).Return(domain.Authentication{Authenticated: true, UserID: ownerID})

// 	appModel := testmocks.NewMockAppModel(mockCtrl)
// 	appModel.EXPECT().GetForOwner(ownerID).Return([]*domain.App{&app1, &app2}, nil)
// 	appModel.EXPECT().GetVersionsForApp(appID1).Return([]*domain.AppVersion{&appVersion1, &appVersion2}, nil)
// 	appModel.EXPECT().GetVersionsForApp(appID2).Return([]*domain.AppVersion{}, nil)

// 	appspacemodel := testmocks.NewMockAppspaceModel(mockCtrl)
// 	appspacemodel.EXPECT().GetForAppVersion(appVersion1.AppID, appVersion1.Version).Return([]*domain.Appspace{&appspace1, &appspace2}, nil)
// 	appspacemodel.EXPECT().GetForAppVersion(appVersion2.AppID, appVersion2.Version).Return([]*domain.Appspace{}, nil)

// 	api := UserJSONAPI{
// 		Auth:          authenticator,
// 		AppModel:      appModel,
// 		AppspaceModel: appspacemodel,
// 	}
// 	api.Init()

// 	resp, payload := apiReq(&api, "GET", "/apps?filter=owner&include=versions.appspaces", "")

// 	if resp.StatusCode != http.StatusOK {
// 		t.Errorf("expected code 200, got %v", resp.StatusCode)
// 	}

// 	payloadContains(t, payload, "app-one", "app-two", "app-version-one", "app-version-two", "subdomain-one", "subdomain-two")
// }
