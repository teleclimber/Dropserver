package userdisplayimagesmodel

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func makeModel(dataDir string) *UserDisplayImagesModel {
	return &UserDisplayImagesModel{
		Config: &domain.RuntimeConfig{
			DataDir: dataDir,
		},
	}
}

func TestUserDir(t *testing.T) {
	m := makeModel("/data")

	cases := []struct {
		userID domain.UserID
		expect string
	}{
		{1, "/data/user-display-images/user_1"},
		{42, "/data/user-display-images/user_42"},
		{0, "/data/user-display-images/user_0"},
		{999, "/data/user-display-images/user_999"},
	}

	for _, c := range cases {
		got := m.userDir(c.userID)
		if got != c.expect {
			t.Errorf("userDir(%d) = %q, want %q", c.userID, got, c.expect)
		}
	}
}

func TestFilePath(t *testing.T) {
	m := makeModel("/data")

	cases := []struct {
		userID   domain.UserID
		filename string
		expect   string
	}{
		{1, "abc123.jpg", "/data/user-display-images/user_1/abc123.jpg"},
		{42, "xyz789.jpg", "/data/user-display-images/user_42/xyz789.jpg"},
	}

	for _, c := range cases {
		got := m.FilePath(c.userID, c.filename)
		if got != c.expect {
			t.Errorf("FilePath(%d, %q) = %q, want %q", c.userID, c.filename, got, c.expect)
		}
	}
}

func TestFilePathDifferentDataDir(t *testing.T) {
	m := makeModel("/var/dropserver")

	got := m.FilePath(7, "img.jpg")
	expect := "/var/dropserver/user-display-images/user_7/img.jpg"
	if got != expect {
		t.Errorf("FilePath = %q, want %q", got, expect)
	}
}
