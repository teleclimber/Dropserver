package record

import (
	"fmt"
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestAppID(t *testing.T) {
	l := NewDsLogger().AppID(domain.AppID(7))

	if !l.hasAppID {
		t.Error("should have app id")
	}
	if l.appID != domain.AppID(7) {
		t.Error("wrong app id")
	}

	l.AppID(domain.AppID(13))
	if l.appID != domain.AppID(13) {
		t.Error("wrong app id")
	}
	if !strings.Contains(l.note, "7") {
		t.Error("note should record fomer id")
	}
}

func TestAppVersion(t *testing.T) {
	l := NewDsLogger()

	if l.appVersion != domain.Version("") {
		t.Error("expected zero-value for version")
	}

	l.AppVersion(domain.Version("0.2.1"))
	l.AppVersion(domain.Version("0.2.9"))

	if l.appVersion != domain.Version("0.2.9") {
		t.Error("expected second value of version")
	}
	if !strings.Contains(l.note, "0.2.1") {
		t.Error("note should record fomer version")
	}
}

func TestAppspaceID(t *testing.T) {
	l := NewDsLogger().AppspaceID(domain.AppspaceID(7))

	if !l.hasAppspaceID {
		t.Error("should have appspace id")
	}
	if l.appspaceID != domain.AppspaceID(7) {
		t.Error("wrong appspace id")
	}

	l.AppspaceID(domain.AppspaceID(13))
	if l.appspaceID != domain.AppspaceID(13) {
		t.Error("wrong appspace id")
	}
	if !strings.Contains(l.note, "7") {
		t.Error("note should record fomer id")
	}
}

func TestUserID(t *testing.T) {
	l := NewDsLogger().UserID(domain.UserID(7))

	if !l.hasUserID {
		t.Error("should have user id")
	}
	if l.userID != domain.UserID(7) {
		t.Error("wrong user id")
	}

	l.UserID(domain.UserID(13))
	if l.userID != domain.UserID(13) {
		t.Error("wrong user id")
	}
	if !strings.Contains(l.note, "7") {
		t.Error("note should record fomer id")
	}
}

func TestContextStr(t *testing.T) {
	l := NewDsLogger()

	l.AppspaceID(domain.AppspaceID(7)).AppID(domain.AppID(13)).AppVersion(domain.Version("0.0.9")).UserID(domain.UserID(77)).AddNote("hello")
	str := l.contextStr()

	for _, s := range []string{"as:7", "a:13", "v:0.0.9", "u:77", "hello"} {
		if !strings.Contains(str, s) {
			t.Error(fmt.Sprintf("expected %v in context string: %v", s, str))
		}
	}
}

func TestClone(t *testing.T) {
	l := NewDsLogger().AppID(domain.AppID(7)).AddNote("hello")
	c := l.Clone()

	if c.appID != l.appID {
		t.Error("app ids should be the same")
	}

	l.AppID(domain.AppID(13))
	if c.appID == l.appID {
		t.Error("app ids should be different")
	}

	c = l.Clone()
	c.AddNote("world")
	if c.contextStr() == l.contextStr() {
		t.Error("contenxt strings should be different")
	}
}
