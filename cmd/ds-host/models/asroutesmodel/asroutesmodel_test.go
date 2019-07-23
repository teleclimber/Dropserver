package asroutesmodel

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

func TestApplyConfig(t *testing.T) {
	jr := domain.JSONRoute{
		Authorize: "foo",
		Method:    "PosT",
		Handler: domain.JSONRouteHandler{
			Type: "bar"}}

	route := domain.RoutePart{}

	dsErr := applyConfig(jr, &route)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if route.POST.Type != "bar" {
		t.Error("Config not applied", route)
	}
}

func TestApplyConfigExists(t *testing.T) {
	jr := domain.JSONRoute{
		Authorize: "foo",
		Method:    "Post",
		Handler: domain.JSONRouteHandler{
			Type: "bar"}}

	route := domain.RoutePart{
		POST: &domain.RouteConfig{
			Type: "static"}}

	dsErr := applyConfig(jr, &route)
	if dsErr == nil {
		t.Error("that should have been an error")
	}
}

//////////////////////////////
// test parseRoutes

func TestParseRoutes(t *testing.T) {
	jsonRoutes := []domain.JSONRoute{
		{Route: "/", Method: "Post", Authorize: "owner", Handler: domain.JSONRouteHandler{Type: "foo"}},
	}

	model := ASRoutesModel{}

	parsed, dsErr := model.parseRoutes(jsonRoutes)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if parsed.POST.Type != "foo" {
		t.Error("did not get expected structure", parsed)
	}
}

func TestParseMultipleRoutes(t *testing.T) {
	jsonRoutes := []domain.JSONRoute{
		{Route: "/", Method: "Get", Handler: domain.JSONRouteHandler{Type: "foo"}},
		{Route: "/aBc", Method: "Post", Handler: domain.JSONRouteHandler{Type: "bar"}},
	}

	model := ASRoutesModel{}

	parsed, dsErr := model.parseRoutes(jsonRoutes)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if parsed.GET.Type != "foo" {
		t.Error("did not get expected structure", parsed)
	}
	if parsed.Parts["abc"].POST.Type != "bar" {
		t.Error("did not get expected structure", parsed)
	}
}

func TestParseMultipleMethods(t *testing.T) {
	jsonRoutes := []domain.JSONRoute{
		{Route: "/abC/Def", Method: "Get", Handler: domain.JSONRouteHandler{Type: "foo"}},
		{Route: "/aBc/deF", Method: "Post", Handler: domain.JSONRouteHandler{Type: "bar"}},
	}

	model := ASRoutesModel{}

	parsed, dsErr := model.parseRoutes(jsonRoutes)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if parsed.Parts["abc"].Parts["def"].GET.Type != "foo" {
		t.Error("did not get expected structure", parsed)
	}
	if parsed.Parts["abc"].Parts["def"].POST.Type != "bar" {
		t.Error("did not get expected structure", parsed)
	}
}

func TestParseCommonSubRoute(t *testing.T) {
	jsonRoutes := []domain.JSONRoute{
		{Route: "/abc", Method: "Get", Handler: domain.JSONRouteHandler{Type: "foo"}},
		{Route: "/abc/def", Method: "Get", Handler: domain.JSONRouteHandler{Type: "bar"}},
	}

	model := ASRoutesModel{}

	parsed, dsErr := model.parseRoutes(jsonRoutes)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if parsed.Parts["abc"].GET.Type != "foo" {
		t.Error("did not get expected structure", parsed)
	}
	if parsed.Parts["abc"].Parts["def"].GET.Type != "bar" {
		t.Error("did not get expected structure", parsed)
	}
}

///////////////////////////////
func TestFetchAppConfig(t *testing.T) {
	// ensure fetch doesn't trip up creating the nested map:

	ar := map[domain.AppID]map[domain.Version]*domain.RoutePart{}
	RunTestFetchAppConfig(t, ar)

	ar = make(map[domain.AppID]map[domain.Version]*domain.RoutePart)
	RunTestFetchAppConfig(t, ar)

	ar = make(map[domain.AppID]map[domain.Version]*domain.RoutePart)
	ar[domain.AppID(1)] = make(map[domain.Version]*domain.RoutePart)
	RunTestFetchAppConfig(t, ar)
}
func RunTestFetchAppConfig(t *testing.T, appRoutes map[domain.AppID]map[domain.Version]*domain.RoutePart) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reply := domain.AppFilesMetadata{
		Routes: []domain.JSONRoute{
			{Route: "/", Method: "Get", Handler: domain.JSONRouteHandler{Type: "foo"}},
			{Route: "/abc/def", Method: "Get", Handler: domain.JSONRouteHandler{Type: "bar"}},
		}}
	appFilesModel := domain.NewMockAppFilesModel(mockCtrl)
	appFilesModel.EXPECT().ReadMeta(gomock.Any()).Return(&reply, nil)

	model := ASRoutesModel{
		AppFilesModel: appFilesModel,
		appRoutes:     appRoutes}

	appID := domain.AppID(1)
	ver := domain.Version("0.0.1")
	model.fetchAppConfig(domain.AppVersion{AppID: appID, Version: ver})

	if model.appRoutes[appID][ver].GET.Type != "foo" {
		t.Error("appRoutes not as expected", model.appRoutes)
	}

}

func TestGetAppRoutes(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	reply := domain.AppFilesMetadata{
		Routes: []domain.JSONRoute{
			{Route: "/", Method: "Get", Handler: domain.JSONRouteHandler{Type: "foo"}},
			{Route: "/abc/def", Method: "Get", Handler: domain.JSONRouteHandler{Type: "bar"}},
		}}
	appFilesModel := domain.NewMockAppFilesModel(mockCtrl)
	appFilesModel.EXPECT().ReadMeta(gomock.Any()).Return(&reply, nil)

	model := ASRoutesModel{
		AppFilesModel: appFilesModel}

	appID := domain.AppID(1)
	ver := domain.Version("0.0.1")
	rp, dsErr := model.getAppRoutes(domain.AppVersion{AppID: appID, Version: ver})
	if dsErr != nil {
		t.Error(dsErr)
	}

	if rp.GET.Type != "foo" {
		t.Error("appRoutes not as expected", rp)
	}
}

/////////////////////
func TestGetRouteConfig(t *testing.T) {
	appID := domain.AppID(1)
	ver := domain.Version("0.0.1")

	ar := make(map[domain.AppID]map[domain.Version]*domain.RoutePart)
	ar[appID] = make(map[domain.Version]*domain.RoutePart)

	m := ASRoutesModel{
		appRoutes: ar}

	appVersion := domain.AppVersion{AppID: appID, Version: ver}

	staticJsr := []domain.JSONRoute{
		{Route: "/static/", Method: "get", Authorize: "owner", Handler: domain.JSONRouteHandler{
			Type: "foo"}},
	}

	rootJsr := []domain.JSONRoute{
		{Route: "/", Method: "get", Authorize: "owner", Handler: domain.JSONRouteHandler{
			Type: "foo"}},
	}

	cases := []struct {
		jsr     []domain.JSONRoute
		pathStr string
		found   bool
	}{
		{staticJsr, "/staTic", true},
		{staticJsr, "/", false},
		{staticJsr, "/staTic/aBc/dEf", true},
		{rootJsr, "/", true},
		{rootJsr, "/foO", true}, // kind of unfortunate, really
	}

	for _, c := range cases {
		rp, dsErr := m.parseRoutes(c.jsr)
		if dsErr != nil {
			t.Error(dsErr)
		}
		m.appRoutes[appID][ver] = rp

		rc, dsErr := m.GetRouteConfig(appVersion, "gEt", c.pathStr)
		if dsErr != nil {
			if dsErr.Code() == dserror.AppspaceRouteNotFound {
				if c.found {
					t.Error("should have found route", c)
				}
			} else {
				t.Error("should not have errored", c, dsErr)
			}
		} else if c.found {
			if rc.Type != "foo" {
				t.Error("did not get expected route config", rc)
			}
		}
	}
}
