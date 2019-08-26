package userroutes

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestGetAppspaceFromPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	uid := domain.UserID(7)
	routeData := &domain.AppspaceRouteData{
		URLTail: "/123",
		Cookie: &domain.Cookie{
			UserID: uid}}

	asm := domain.NewMockAppspaceModel(mockCtrl)
	asm.EXPECT().GetFromID(domain.AppspaceID(123)).Return(&domain.Appspace{OwnerID: uid}, nil)

	a := AppspaceRoutes{
		AppspaceModel: asm,
	}

	appspace, dsErr := a.getAppspaceFromPath(routeData)
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if appspace == nil {
		t.Fatal("appspace should not be null")
	}
}

func TestGetAppspaceFromPathNil(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	routeData := &domain.AppspaceRouteData{
		URLTail: "/"}

	a := AppspaceRoutes{}

	appspace, dsErr := a.getAppspaceFromPath(routeData)
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if appspace != nil {
		t.Fatal("appspace should have been nil")
	}
}

func TestGetAppspaceFromPathBadReq(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	routeData := &domain.AppspaceRouteData{
		URLTail: "/123def"}

	a := AppspaceRoutes{}

	appspace, dsErr := a.getAppspaceFromPath(routeData)
	if dsErr == nil {
		t.Error("bad req should produce an error")
	}
	if appspace != nil {
		t.Fatal("appspace should have been nil")
	}
}

func TestGetAppspaceFromPathUnauthorized(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	routeData := &domain.AppspaceRouteData{
		URLTail: "/123",
		Cookie: &domain.Cookie{
			UserID: domain.UserID(7)}}

	asm := domain.NewMockAppspaceModel(mockCtrl)
	asm.EXPECT().GetFromID(domain.AppspaceID(123)).Return(&domain.Appspace{OwnerID: domain.UserID(13)}, nil)

	a := AppspaceRoutes{
		AppspaceModel: asm,
	}

	appspace, dsErr := a.getAppspaceFromPath(routeData)
	if dsErr == nil {
		t.Error("there should have been an error")
	}
	if appspace != nil {
		t.Fatal("appspace should not be nil")
	}
}
