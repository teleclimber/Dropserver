package appops

import (
	"testing"

	"github.com/blang/semver/v4"
	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestSetKey(t *testing.T) {
	g := &AppGetter{}
	g.Init()

	key := g.set(AppGetData{locationKey: "abc"})

	data, ok := g.keys[key]
	if !ok {
		t.Error("key not set")
	}
	if data.locationKey != "abc" {
		t.Error("data not set correctly")
	}
}

func TestValidateAppMeta(t *testing.T) {
	g := &AppGetter{}
	cases := []struct {
		meta   domain.AppFilesMetadata
		numErr int
	}{
		{domain.AppFilesMetadata{AppName: "blah", AppVersion: "0.0.1"}, 0},
		{domain.AppFilesMetadata{AppVersion: "0.0.1"}, 1},
		{domain.AppFilesMetadata{AppName: "blah"}, 1},
	}

	for _, c := range cases {
		errs, err := g.validateVersion(&c.meta)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != c.numErr {
			t.Log(errs)
			t.Error("Error count mismatch")
		}
	}
}

func TestValidateUserPermissions(t *testing.T) {
	g := &AppGetter{}
	meta := domain.AppFilesMetadata{AppName: "blah", AppVersion: "0.0.1"}
	cases := []struct {
		perms  []domain.AppspaceUserPermission
		numErr int
	}{
		{[]domain.AppspaceUserPermission{{Key: "abc"}}, 0},
		{[]domain.AppspaceUserPermission{{Key: ""}}, 1},
		{[]domain.AppspaceUserPermission{{Key: "abc"}, {Key: "abc"}}, 1},
		{[]domain.AppspaceUserPermission{{Key: "abc"}, {Key: "def"}}, 0},
	}

	for _, c := range cases {
		meta.UserPermissions = c.perms
		errs, err := g.validateVersion(&meta)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != c.numErr {
			t.Log(errs)
			t.Error("Error count mismatch")
		}
	}
}

func TestVersionSort(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	ver, _ := semver.New("0.5.0")

	appModel := testmocks.NewMockAppModel(mockCtrl)

	g := &AppGetter{
		AppModel: appModel,
	}

	// basic sorting:
	appModel.EXPECT().GetVersionsForApp(appID).Return([]*domain.AppVersion{
		{Version: domain.Version("0.8.1")},
		{Version: domain.Version("0.2.1")},
	}, nil)
	vers, errs, err := g.getVersions(appID, *ver)
	if err != nil {
		t.Error(err)
	}
	if len(errs) != 0 {
		t.Log(errs)
		t.Error("got unexpected errors")
	}
	if vers[0].appVersion.Version != domain.Version("0.2.1") {
		t.Error("sort order is wrong")
	}

	// dupe version
	appModel.EXPECT().GetVersionsForApp(appID).Return([]*domain.AppVersion{
		{Version: domain.Version("0.8.1")},
		{Version: domain.Version("0.5.0")},
	}, nil)
	_, errs, err = g.getVersions(appID, *ver)
	if err != nil {
		t.Error(err)
	}
	if len(errs) != 1 {
		t.Log(errs)
		t.Error("expected an error")
	}
}

func TestValidateSequence(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	filesMeta := domain.AppFilesMetadata{
		AppVersion:    domain.Version("0.5.0"),
		SchemaVersion: 1,
	}

	appModel := testmocks.NewMockAppModel(mockCtrl)

	g := &AppGetter{
		AppModel: appModel,
	}

	cases := []struct {
		desc        string
		appVersions []*domain.AppVersion
		numErr      int
	}{
		{"incrementing schema", []*domain.AppVersion{
			{Version: domain.Version("0.8.1"), Schema: 2},
			{Version: domain.Version("0.2.1"), Schema: 0},
		}, 0},
		{"same schema", []*domain.AppVersion{
			{Version: domain.Version("0.8.1"), Schema: 1},
			{Version: domain.Version("0.2.1"), Schema: 1},
		}, 0},
		{"next has lower schema", []*domain.AppVersion{
			{Version: domain.Version("0.8.1"), Schema: 0},
			{Version: domain.Version("0.2.1"), Schema: 1},
		}, 1},
		{"prev has higher schema", []*domain.AppVersion{
			{Version: domain.Version("0.8.1"), Schema: 1},
			{Version: domain.Version("0.2.1"), Schema: 2},
		}, 1},
		{"prev only increment schema", []*domain.AppVersion{
			{Version: domain.Version("0.2.1"), Schema: 0},
		}, 0},
		{"next only increment schema", []*domain.AppVersion{
			{Version: domain.Version("0.8.1"), Schema: 2},
		}, 0},
	}

	for _, c := range cases {
		appGetMeta := domain.AppGetMeta{}
		appModel.EXPECT().GetVersionsForApp(appID).Return(c.appVersions, nil)
		err := g.validateVersionSequence(appID, &filesMeta, &appGetMeta)
		if err != nil {
			t.Error(c.desc, err)
		}
		if len(appGetMeta.Errors) != c.numErr {
			t.Log(appGetMeta.Errors)
			t.Error(c.desc, "got unexpected errors")
		}
	}
}
