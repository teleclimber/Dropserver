package appspacemetadb

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

func TestV0validateAuth(t *testing.T) {
	// OK there ust isn't anything to do here yet
}

func TestV0validateHandler(t *testing.T) {
	handler := domain.AppspaceRouteHandler{
		Type: "file",
		Path: "/some-path/yo"}

	err := v0validateHandler(handler)
	if err == nil {
		t.Error("expected error from bad path")
	}

	handler.Path = "@app/yo"
	err = v0validateHandler(handler)
	if err != nil {
		t.Error(err)
	}
}

func TestV0RouteCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	auth := domain.AppspaceRouteAuth{Type: "owner"}
	handler := domain.AppspaceRouteHandler{Type: "file", Path: "@app/abc/"}

	r := v0RoutesGetModel(t, mockCtrl)

	dsErr := r.Create([]string{"get", "post"}, "/abc/", auth, handler)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	dsErr = r.Create([]string{"post"}, "/abc", auth, handler)
	if dsErr == nil {
		t.Fatal("Expected error on duplicate route")
	}
	if dsErr.Code() != dserror.AppspaceRouteExists {
		t.Fatal("Expected error to be route exists.")
	}

	dsErr = r.Create([]string{"patch"}, "/abc", auth, handler)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	dsErr = r.Create([]string{"get", "post"}, "/abc/def", auth, handler)
	if dsErr != nil {
		t.Fatal(dsErr)
	}
}

func TestV0RouteGet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	r := v0RoutesGetModel(t, mockCtrl)

	rr, dsErr := r.Get([]string{"get"}, "/abc")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if len(*rr) != 0 {
		t.Error("expected no rows")
	}

	dsErr = r.Create([]string{"get"}, "/abc/", domain.AppspaceRouteAuth{Type: "public"}, domain.AppspaceRouteHandler{Type: "function", File: "@app/abc"})
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	dsErr = r.Create([]string{"post", "patch"}, "/abc/", domain.AppspaceRouteAuth{Type: "owner"}, domain.AppspaceRouteHandler{Type: "function", File: "@app/abc"})
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	rr, dsErr = r.Get([]string{"get"}, "/abc")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if len(*rr) != 1 {
		t.Error("expected 1 row")
	}
	if (*rr)[0].Auth.Type != "public" {
		t.Error("didn't get the row data we expected")
	}
	if (*rr)[0].Handler.Type != "function" {
		t.Error("didn't get the row data we expected")
	}

	rr, dsErr = r.Get([]string{"get", "post"}, "/abc")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if len(*rr) != 2 {
		t.Error("expected 2 row")
	}
}

func TestV0Delete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	r := v0RoutesGetModel(t, mockCtrl)

	dsErr := r.Create([]string{"get", "post", "patch"}, "/abc/", domain.AppspaceRouteAuth{Type: "owner"}, domain.AppspaceRouteHandler{Type: "function", File: "@app/abc"})
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	dsErr = r.Delete([]string{"get"}, "/abc")
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	rr, dsErr := r.Get([]string{"get"}, "/abc")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if len(*rr) != 0 {
		t.Error("expected no row")
	}
}

func TestV0RouteMatch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	handler := domain.AppspaceRouteHandler{Type: "function", File: "@app/abc"}

	appspaceID := domain.AppspaceID(7)

	db := v0RoutesGetTestDBHandle(t)

	dbc := domain.NewMockDbConn(mockCtrl)
	dbc.EXPECT().GetHandle().Return(db).AnyTimes()
	appspaceMetaDb := domain.NewMockAppspaceMetaDB(mockCtrl)
	appspaceMetaDb.EXPECT().GetConn(appspaceID).Return(dbc).AnyTimes()
	r := &RouteModelV0{
		AppspaceMetaDB: appspaceMetaDb,
		appspaceID:     appspaceID,
	}

	dsErr := r.Create([]string{"get", "post"}, "/abc/", domain.AppspaceRouteAuth{Type: "owner"}, handler)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	dsErr = r.Create([]string{"get"}, "/abc/def/", domain.AppspaceRouteAuth{Type: "public"}, handler)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	dsErr = r.Create([]string{"get"}, "/uvw/somefile.txt", domain.AppspaceRouteAuth{Type: "public"}, handler)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	route, dsErr := r.Match("get", "/xyz/")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if route != nil {
		t.Error("expected no route found")
	}

	route, dsErr = r.Match("get", "/abc/def/")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if route == nil {
		t.Fatal("expected a route found")
	}
	if route.Path != "/abc/def" {
		t.Error("Got the wrong route?")
	}
	if route.Auth.Type != "public" {
		t.Error("got the wrong route data")
	}

	route, dsErr = r.Match("post", "/abc/def/")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if route == nil {
		t.Fatal("expected a route found")
	}
	if route.Path != "/abc" {
		t.Error("Got the wrong route?")
	}
	if route.Auth.Type != "owner" {
		t.Error("got the wrong route data")
	}

	route, dsErr = r.Match("get", "/uvw/")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if route != nil {
		t.Error("expecte /uvw/ route to be nil")
	}

	route, dsErr = r.Match("get", "/uvw/somefile.txt")
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if route == nil {
		t.Error("expected non-nil route")
	}
}

// test normalize url
// test normalize method

func TestV0GetMethodsFromBits(t *testing.T) {
	cases := []struct {
		bits    uint16
		methods []string
	}{
		{0, []string{}},
		{1, []string{"get"}},
		{260, []string{"patch", "post"}},
		{511, []string{"connect", "delete", "get", "head", "options", "patch", "post", "put", "trace"}},
	}

	for _, c := range cases {
		t.Run(strconv.Itoa(int(c.bits)), func(t *testing.T) {
			methods := v0GetMethodsFromBits(c.bits)
			if !reflect.DeepEqual(methods, c.methods) {
				t.Errorf("Expected %v, got %v", c.methods, methods)
			}
		})
	}
}

//////////////////////////////////////////////////

func v0RoutesGetModel(t *testing.T, mockCtrl *gomock.Controller) *RouteModelV0 {
	appspaceID := domain.AppspaceID(7)

	db := v0RoutesGetTestDBHandle(t)

	dbc := domain.NewMockDbConn(mockCtrl)
	dbc.EXPECT().GetHandle().Return(db).AnyTimes()
	appspaceMetaDb := domain.NewMockAppspaceMetaDB(mockCtrl)
	appspaceMetaDb.EXPECT().GetConn(appspaceID).Return(dbc).AnyTimes()
	r := &RouteModelV0{
		AppspaceMetaDB: appspaceMetaDb,
		appspaceID:     appspaceID,
	}

	return r
}

func v0RoutesGetTestDBHandle(t *testing.T) *sqlx.DB {
	// Beware of in-memory DBs: they vanish as soon as the connection closes!
	// We may be able to start a sqlx transaction to avoid problems with that?
	// See: https://github.com/jmoiron/sqlx/issues/164
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	v0h := &v0handle{handle: handle}

	v0h.migrateUpToV0()

	dsErr := v0h.checkErr()
	if dsErr != nil {
		t.Error(dsErr)
	}

	return handle
}
