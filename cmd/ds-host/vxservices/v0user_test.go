package vxservices

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// test twine message handlers

func v0UserGetTestModel(t *testing.T, mockCtrl *gomock.Controller) *V0UserModel {
	appspaceID := domain.AppspaceID(7)

	return &V0UserModel{
		appspaceID: appspaceID,
	}
}
