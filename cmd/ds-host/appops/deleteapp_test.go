package appops

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestDeleteAppVersion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	v := domain.Version("0.1.2")
	loc := "test-loc"

	asModel := testmocks.NewMockAppspaceModel(mockCtrl)
	asModel.EXPECT().GetForAppVersion(appID, v).Return([]*domain.Appspace{}, nil)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersion(appID, v).Return(domain.AppVersion{LocationKey: loc}, nil)
	appModel.EXPECT().DeleteVersion(appID, v).Return(nil)

	afModel := testmocks.NewMockAppFilesModel(mockCtrl)
	afModel.EXPECT().Delete(loc).Return(nil)

	appLogger := testmocks.NewMockAppLogger(mockCtrl)
	appLogger.EXPECT().Forget(loc)

	d := DeleteApp{
		AppFilesModel: afModel,
		AppModel:      appModel,
		AppspaceModel: asModel,
		AppLogger:     appLogger,
	}

	err := d.DeleteVersion(appID, v)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteVersionVersionInUse(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	v := domain.Version("0.1.2")

	asModel := testmocks.NewMockAppspaceModel(mockCtrl)
	asModel.EXPECT().GetForAppVersion(appID, v).Return([]*domain.Appspace{{AppID: appID, AppVersion: v}}, nil)

	d := DeleteApp{
		AppspaceModel: asModel,
	}

	err := d.DeleteVersion(appID, v)
	if err != domain.ErrAppVersionInUse {
		t.Error("Expect an app version in use error")
	}
}

func TestDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	v := domain.Version("0.1.2")
	loc := "test-loc"

	asModel := testmocks.NewMockAppspaceModel(mockCtrl)
	asModel.EXPECT().GetForApp(appID).Return([]*domain.Appspace{}, nil)

	appModel := testmocks.NewMockAppModel(mockCtrl)
	appModel.EXPECT().GetVersionsForApp(appID).Return([]*domain.AppVersion{{AppID: appID, Version: v, LocationKey: loc}}, nil)
	appModel.EXPECT().DeleteVersion(appID, v).Return(nil)
	appModel.EXPECT().Delete(appID).Return(nil)

	afModel := testmocks.NewMockAppFilesModel(mockCtrl)
	afModel.EXPECT().Delete(loc).Return(nil)

	appLogger := testmocks.NewMockAppLogger(mockCtrl)
	appLogger.EXPECT().Forget(loc)

	d := DeleteApp{
		AppspaceModel: asModel,
		AppFilesModel: afModel,
		AppModel:      appModel,
		AppLogger:     appLogger,
	}

	err := d.Delete(appID)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteVersionInUse(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	appID := domain.AppID(7)
	v := domain.Version("0.1.2")

	asModel := testmocks.NewMockAppspaceModel(mockCtrl)
	asModel.EXPECT().GetForApp(appID).Return([]*domain.Appspace{{AppID: appID, AppVersion: v}}, nil)

	d := DeleteApp{
		AppspaceModel: asModel,
	}

	err := d.Delete(appID)
	if err != domain.ErrAppVersionInUse {
		t.Error("Expect an app version in use error")
	}
}
