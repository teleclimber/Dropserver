package runtimeconfig

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestHas(t *testing.T) {
	k := &SetupKey{
		cached: true,
		key:    "abc"}

	has, _ := k.Has()
	if !has {
		t.Error("expected Has to be true")
	}

	k.key = ""
	has, _ = k.Has()
	if has {
		t.Error("expected Has to be false")
	}
}

func TestLoadKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := testmocks.NewMockDBManager(mockCtrl)
	m.EXPECT().GetSetupKey().Return("abc", nil)
	u := testmocks.NewMockUserModel(mockCtrl)
	u.EXPECT().GetAllAdmins().Return([]domain.UserID{}, nil)

	k := &SetupKey{
		UserModel: u,
		DBManager: m,
	}

	err := k.loadKey()
	if err != nil {
		t.Error(err)
	}
	if !k.cached {
		t.Error("expected cached to be true")
	}
	if k.key != "abc" {
		t.Error("expected key to be abc")
	}
}

func TestLoadNoKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := testmocks.NewMockDBManager(mockCtrl)
	m.EXPECT().GetSetupKey().Return("", nil)

	k := &SetupKey{
		DBManager: m,
	}

	err := k.loadKey()
	if err != nil {
		t.Error(err)
	}
	if !k.cached {
		t.Error("expected cached to be true")
	}
	if k.key != "" {
		t.Error("expected key to be empty string")
	}
}

func TestLoadKeyWithAdmins(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := testmocks.NewMockDBManager(mockCtrl)
	m.EXPECT().GetSetupKey().Return("abc", nil)
	u := testmocks.NewMockUserModel(mockCtrl)
	u.EXPECT().GetAllAdmins().Return([]domain.UserID{domain.UserID(123)}, nil)

	k := &SetupKey{
		UserModel: u,
		DBManager: m,
	}

	err := k.loadKey()
	if err != nil {
		t.Error(err)
	}
	if !k.cached {
		t.Error("expected cached to be true")
	}
	if k.key != "" {
		t.Error("expected key to be empty string")
	}
}

func TestGetSecretUrl(t *testing.T) {
	c := &domain.RuntimeConfig{}
	c.Exec.UserRoutesDomain = "xyz.uvw"
	c.PortString = ":123"

	k := &SetupKey{
		Config: c,
		cached: true,
		key:    "abc"}

	assertEqStr(t, "https://xyz.uvw:123/abc", k.getSecretUrl())

	c.PortString = ""
	c.Server.NoTLS = true

	assertEqStr(t, "http://xyz.uvw/abc", k.getSecretUrl())
}
