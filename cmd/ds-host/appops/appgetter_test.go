package appops

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestSetKey(t *testing.T) {
	g := &AppGetter{}
	g.Init()

	d := g.set(appGetData{locationKey: "abc"})

	data, ok := g.keys[d.key]
	if !ok {
		t.Error("key not set")
	}
	if data.locationKey != "abc" {
		t.Error("data not set correctly")
	}
}

func TestValidatePackageFile(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"/", "", false},
		{".", "", false},
		{"abc/../def/", "def", true},
		{"abc/../../def/", "", false},
		{"/abc/def", "abc/def", true},
		{"abc\\def", "abc\\def", false},
	}
	for _, c := range cases {
		p, ok := validatePackagePath(c.input)
		if c.ok != ok {
			t.Errorf("mismatchin OK value: expected: %v, got: %v", c.ok, ok)
		}
		if c.ok && c.expected != p {
			t.Errorf("expected entrypoint does not match: %v %v", c.expected, p)
		}
	}
}

type l2p struct {
	appFiles string
}

func (l *l2p) Files(loc string) string {
	return l.appFiles
}

func TestGetDefaultEntryPoint(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := &AppGetter{
		AppLocation2Path: &l2p{appFiles: dir},
	}
	m := domain.AppGetMeta{
		Errors:          make([]string, 0),
		VersionManifest: domain.AppVersionManifest{},
	}

	// getEntrypoint when there are no files:
	err = g.getEntrypoint("", &m)
	if err != nil {
		t.Error(err)
	}
	if len(m.Errors) != 1 {
		t.Error("expected Meta.Error because no app js or ts in files")
	}

	// now put app.js
	err = os.WriteFile(filepath.Join(g.AppLocation2Path.Files(""), "app.js"), []byte("app"), 0644)
	if err != nil {
		t.Error(err)
	}
	m.Errors = make([]string, 0)
	err = g.getEntrypoint("", &m)
	if err != nil {
		t.Error(err)
	}
	if len(m.Errors) != 0 {
		t.Errorf("expected no Meta.Error %v", m.Errors)
	}
	if m.VersionManifest.Entrypoint != "app.js" {
		t.Error("expected entrypoint to be app.js")
	}

	// now with both app.js and app.ts
	err = os.WriteFile(filepath.Join(g.AppLocation2Path.Files(""), "app.ts"), []byte("app"), 0644)
	if err != nil {
		t.Error(err)
	}
	m.VersionManifest.Entrypoint = ""
	m.Errors = make([]string, 0)
	err = g.getEntrypoint("", &m)
	if err != nil {
		t.Error(err)
	}
	if len(m.Errors) != 1 {
		t.Errorf("expected one Meta.Error")
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
	vers, appErr, err := g.getVersions(appID, *ver)
	if err != nil {
		t.Error(err)
	}
	if appErr != "" {
		t.Error(appErr)
	}
	if vers[0].appVersion.Version != domain.Version("0.2.1") {
		t.Error("sort order is wrong")
	}

	// dupe version
	appModel.EXPECT().GetVersionsForApp(appID).Return([]*domain.AppVersion{
		{Version: domain.Version("0.8.1")},
		{Version: domain.Version("0.5.0")},
	}, nil)
	_, appErr, err = g.getVersions(appID, *ver)
	if err != nil {
		t.Error(err)
	}
	if appErr == "" {
		t.Error("expected an error")
	}
}

func TestValidateSequence(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)

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
		appGetMeta := domain.AppGetMeta{
			VersionManifest: domain.AppVersionManifest{
				Version: domain.Version("0.5.0"),
				Schema:  1,
			},
		}
		appModel.EXPECT().GetVersionsForApp(appID).Return(c.appVersions, nil)
		err := g.validateVersionSequence(appID, &appGetMeta)
		if err != nil {
			t.Error(c.desc, err)
		}
		if len(appGetMeta.Errors) != c.numErr {
			t.Log(appGetMeta.Errors)
			t.Error(c.desc, "got unexpected errors")
		}
	}
}

func TestValidateMigrationSteps(t *testing.T) {
	g := &AppGetter{}

	cases := []struct {
		desc       string
		migrations []domain.MigrationStep
		schemas    []int
		isErr      bool
	}{
		{
			desc:       "empty array",
			migrations: []domain.MigrationStep{},
			schemas:    []int{},
			isErr:      false,
		}, {
			desc: "up",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}},
			schemas: nil,
			isErr:   true,
		}, {
			desc: "down",
			migrations: []domain.MigrationStep{
				{Direction: "down", Schema: 1}},
			schemas: nil,
			isErr:   true,
		}, {
			desc: "up down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1}},
			schemas: []int{1},
			isErr:   false,
		}, {
			desc: "up up down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2},
				{Direction: "up", Schema: 1}},
			schemas: nil,
			isErr:   true,
		}, {
			desc: "up1 up1 down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "up", Schema: 1}},
			schemas: nil,
			isErr:   true,
		}, {
			desc: "up down down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2},
				{Direction: "down", Schema: 1}},
			schemas: nil,
			isErr:   true,
		}, {
			desc: "up gap up down gap down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3}},
			schemas: nil,
			isErr:   true,
		}, {
			desc: "up up up down gap down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3},
				{Direction: "up", Schema: 2}},
			schemas: nil,
			isErr:   true,
		}, {
			desc: "up up up down down down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3},
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2}},
			schemas: []int{1, 2, 3},
			isErr:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			schemas, err := g.ValidateMigrationSteps(c.migrations)
			if (err != nil) != c.isErr {
				t.Errorf("mismatch between error and expected: %v %v", c.isErr, err)
			}
			if !reflect.DeepEqual(schemas, c.schemas) {
				t.Errorf("schemas not equal: %v, %v", schemas, c.schemas)
			}
		})
	}
}
