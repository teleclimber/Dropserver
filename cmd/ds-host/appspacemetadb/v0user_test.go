package appspacemetadb

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestValidateProxyID(t *testing.T) {
	m := V0UserModel{}
	err := m.validateProxyID("")
	if err == nil {
		t.Error("expected error from blank proxy ID")
	}
	err = m.validateProxyID("abcdefghijkl")
	if err == nil {
		t.Error("expected error from long proxy ID")
	}
	err = m.validateProxyID("abcdef123")
	if err != nil {
		t.Error("expected no error from valid proxy")
	}
}

func TestValidatePermissions(t *testing.T) {
	m := V0UserModel{}
	err := m.validatePermissions([]string{})
	if err != nil {
		t.Error("expected empty array to be valid")
	}
	err = m.validatePermissions([]string{""})
	if err == nil {
		t.Error("expected empty permission string to be invalid")
	}
}

func TestGetEmpty(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := v0UserGetTestModel(t, mockCtrl)

	user, err := m.Get("abc")
	if err != nil {
		t.Error(err)
	}
	if user.ProxyID != "" {
		t.Error("expected empty string for a proxy id")
	}
}

func TestCreateGet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := v0UserGetTestModel(t, mockCtrl)

	proxyID := domain.ProxyID("abc")
	displayName := "jimbob"
	permissions := []string{"read", "write"}
	err := m.Create(proxyID, displayName, permissions)
	if err != nil {
		t.Error(err)
	}

	u, err := m.Get(proxyID)
	if err != nil {
		t.Error(err)
	}
	if u.ProxyID != proxyID {
		t.Error("wrong proxy id")
	}
	if u.DisplayName != displayName {
		t.Error("wrong display name")
	}
	if len(u.Permissions) != 2 || u.Permissions[0] != "read" || u.Permissions[1] != "write" {
		t.Error("permissions wrong.")
	}
}

func TestCreateUpdate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := v0UserGetTestModel(t, mockCtrl)

	proxyID := domain.ProxyID("abc")
	displayName := "jimbob"
	permissions := []string{"read", "write"}

	err := m.Create(proxyID, "janedoe", []string{})
	if err != nil {
		t.Error(err)
	}

	err = m.Update(proxyID, displayName, permissions)
	if err != nil {
		t.Error(err)
	}

	u, err := m.Get(proxyID)
	if err != nil {
		t.Error(err)
	}
	if u.ProxyID != proxyID {
		t.Error("wrong proxy id")
	}
	if u.DisplayName != displayName {
		t.Error("wrong display name")
	}
	if len(u.Permissions) != 2 || u.Permissions[0] != "read" || u.Permissions[1] != "write" {
		t.Error("permissions wrong.")
	}
}

func TestDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := v0UserGetTestModel(t, mockCtrl)

	proxyID := domain.ProxyID("abc")
	displayName := "jimbob"
	permissions := []string{"read", "write"}

	err := m.Create(proxyID, displayName, permissions)
	if err != nil {
		t.Error(err)
	}

	err = m.Delete(proxyID)
	if err != nil {
		t.Error(err)
	}

	u, err := m.Get(proxyID)
	if err != nil {
		t.Error(err)
	}
	if u.ProxyID != "" {
		t.Error("expected blank proxy ID")
	}
}

func TestDuplicateProxyID(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := v0UserGetTestModel(t, mockCtrl)

	proxyID := domain.ProxyID("abc")
	displayName := "jimbob"
	permissions := []string{"read", "write"}

	err := m.Create(proxyID, displayName, permissions)
	if err != nil {
		t.Error(err)
	}

	err = m.Create(proxyID, "jane", []string{})
	if err == nil {
		t.Error("expected an error because duplicate proxy ID")
	}
}

func TestGetAll(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := v0UserGetTestModel(t, mockCtrl)

	u1 := domain.V0User{
		ProxyID:     domain.ProxyID("abc"),
		DisplayName: "abc",
		Permissions: []string{"abc"}}
	u2 := domain.V0User{
		ProxyID:     domain.ProxyID("def"),
		DisplayName: "def",
		Permissions: []string{"abc", "def"}}

	err := m.Create(u1.ProxyID, u1.DisplayName, u1.Permissions)
	if err != nil {
		t.Error(err)
	}
	err = m.Create(u2.ProxyID, u2.DisplayName, u2.Permissions)
	if err != nil {
		t.Error(err)
	}

	users, err := m.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(users) != 2 {
		t.Error("expected 2 users")
	}
	if users[0].DisplayName == "abc" {
		if deep.Equal(u1, users[0]) != nil || deep.Equal(u2, users[1]) != nil {
			t.Error("not equal")
		}
	} else {
		if deep.Equal(u1, users[1]) != nil || deep.Equal(u2, users[0]) != nil {
			t.Error("not equal")
		}
	}

}

// test twine message handlers

func v0UserGetTestModel(t *testing.T, mockCtrl *gomock.Controller) *V0UserModel {
	appspaceID := domain.AppspaceID(7)

	return &V0UserModel{
		AppspaceMetaDB: v0GetTestAppspaceMetaDB(t, mockCtrl, appspaceID),
		appspaceID:     appspaceID,
	}
}
