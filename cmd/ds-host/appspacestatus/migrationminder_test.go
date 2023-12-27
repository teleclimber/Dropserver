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
		desc       string
		curVersion domain.Version
		remote     map[domain.Version]domain.AppListingVersion
		isRemote   bool
		newVersion domain.Version
	}{
		{"no local, empty remote", domain.Version("0.9.0"), nil, false, domain.Version("")},
		{"local upgrade, empty remote", domain.Version("0.2.0"), nil, false, domain.Version("0.7.0")},
		{"no local, remote", domain.Version("0.9.0"), map[domain.Version]domain.AppListingVersion{
			domain.Version("0.10.0"): {},
		}, true, domain.Version("0.10.0")},
		{"local and remote, local higher", domain.Version("0.5.0"), map[domain.Version]domain.AppListingVersion{
			domain.Version("0.6.0"): {},
		}, false, domain.Version("0.7.0")},
		{"local and remote, remote higher", domain.Version("0.5.0"), map[domain.Version]domain.AppListingVersion{
			domain.Version("0.10.0"): {},
		}, true, domain.Version("0.10.0")},
	}

	for _, c := range cases {
		t.Run(string(c.desc), func(t *testing.T) {
			if c.remote == nil {
				appModel.EXPECT().GetAppUrlListing(appID).Return(domain.AppListing{}, domain.AppURLData{}, domain.ErrNoRowsInResultSet)
			} else {
				appModel.EXPECT().GetAppUrlListing(appID).Return(domain.AppListing{Versions: c.remote}, domain.AppURLData{}, nil)
			}
			appspace.AppVersion = c.curVersion
			v, remote, err := mm.GetForAppspace(appspace)
			if err != nil {
				t.Error(err)
			}
			if remote != c.isRemote {
				t.Errorf("is remote wrong: expected %v got %v", c.isRemote, remote)
			}
			if v != c.newVersion {
				t.Errorf("got wrong version: %v", c.newVersion)
			}
		})
	}
}
