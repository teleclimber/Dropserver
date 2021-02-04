package appspacestatus

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestGetForAppspace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersionsForApp(appID).Return([]*domain.AppVersion{
		{Version: "0.2.0"},
		{Version: "0.5.0"},
		{Version: "0.7.0"},
	}, nil).AnyTimes()

	appspace := domain.Appspace{
		AppID:      appID,
		AppVersion: domain.Version("0.5.0"),
	}

	mm := &MigrationMinder{
		AppModel: appModel,
	}

	cases := []struct {
		appspaceVersion domain.Version
		ok              bool
		appVersion      domain.Version
	}{
		{domain.Version("0.2.0"), true, domain.Version("0.7.0")},
		{domain.Version("0.5.0"), true, domain.Version("0.7.0")},
		{domain.Version("0.7.0"), false, domain.Version("")},
	}

	for _, c := range cases {
		t.Run(string(c.appspaceVersion), func(t *testing.T) {
			appspace.AppVersion = c.appspaceVersion
			v, ok, err := mm.GetForAppspace(appspace)
			if err != nil {
				t.Error(err)
			}
			if ok != c.ok {
				t.Error("unexpected OK")
			}
			if ok && v.Version != c.appVersion {
				t.Errorf("got wrong version: %v", c.appVersion)
			}
		})
	}
}

func TestGetAllForOwner(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ownerID := domain.UserID(11)
	appID := domain.AppID(7)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersionsForApp(appID).Return([]*domain.AppVersion{
		{Version: "0.2.0"},
		{Version: "0.5.0"},
		{Version: "0.7.0"},
	}, nil).AnyTimes()

	appspaceModel := testmocks.NewMockAppspaceModel(mockCtrl)
	appspaceModel.EXPECT().GetForOwner(ownerID).Return([]*domain.Appspace{
		{AppspaceID: domain.AppspaceID(20), AppID: appID, AppVersion: domain.Version("0.5.0")},
		{AppspaceID: domain.AppspaceID(21), AppID: appID, AppVersion: domain.Version("0.7.0")},
	}, nil)

	mm := &MigrationMinder{
		AppModel:      appModel,
		AppspaceModel: appspaceModel,
	}

	migrations, err := mm.GetAllForOwner(ownerID)
	if err != nil {
		t.Error(err)
	}
	if len(migrations) != 1 {
		t.Log(migrations)
		t.Error("expected on migreation")
	}
	appVersion, ok := migrations[domain.AppspaceID(20)]
	if !ok {
		t.Log(migrations)
		t.Error("expected appspace 20 in migrations")
	}
	if appVersion.Version != domain.Version("0.7.0") {
		t.Log(migrations)
		t.Error("expected app version 0.7.0")
	}
}
