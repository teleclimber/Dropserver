package appspaceops

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestFindByAuth(t *testing.T) {
	_, found := findByAuth("abc", "123", []domain.AppspaceUser{})
	if found {
		t.Error("expected found to be false")
	}

	users := []domain.AppspaceUser{
		{
			AppspaceID: 7,
			ProxyID:    "proxy123",
			Auths: []domain.AppspaceUserAuth{
				{
					Type:       "dropid",
					Identifier: "dropid456",
				},
			},
		},
	}
	_, found = findByAuth("dropid", "bogus", users)
	if found {
		t.Error("expected found to be false")
	}
	user, found := findByAuth("dropid", "dropid456", users)
	if !found {
		t.Error("expected found to be true")
	}
	if user.ProxyID != "proxy123" {
		t.Error("got wrong user", user)
	}
}

func TestGetDisplayNameFromTSNetUser(t *testing.T) {
	result := getDisplayNameFromTSNetUser(domain.TSNetPeerUser{
		DisplayName: "The Display Name",
		LoginName:   "The Login Name"})
	if result != "The Display Name" {
		t.Errorf("Got %s", result)
	}
	result = getDisplayNameFromTSNetUser(domain.TSNetPeerUser{
		DisplayName: "",
		LoginName:   "The Login Name"})
	if result != "The Login Name" {
		t.Errorf("Got %s", result)
	}
	result = getDisplayNameFromTSNetUser(domain.TSNetPeerUser{
		DisplayName: "A very Long Display Name to get past 30 characters",
		LoginName:   "The Login Name"})
	if result != "A very Long Display Name to ge" {
		t.Errorf("Got %s", result)
	}
	result = getDisplayNameFromTSNetUser(domain.TSNetPeerUser{
		DisplayName: "",
		LoginName:   ""})
	if result != "" {
		t.Errorf("Got %s", result)
	}
}
