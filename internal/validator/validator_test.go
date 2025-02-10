package validator

import (
	"testing"
)

func TestPassword(t *testing.T) {

	cases := []struct {
		pw  string
		err bool
	}{
		{"", true},
		{"abc", true},
		{"abcabcabcabc", false},
		{"             ", false},
	}

	for _, c := range cases {
		err := Password(c.pw)
		if !c.err && err != nil {
			t.Error("should not have gotten error", err)
		} else if c.err && err == nil {
			t.Error("should have gotten error")
		}
	}
}

func TestEmail(t *testing.T) {
	cases := []struct {
		email string
		err   bool
	}{
		{"", true},
		{"abc", true},
		{"abcabcabcabc", true},
		{"             ", true},
		{"a@b.c", false},
	}

	for _, c := range cases {
		err := Email(c.email)
		if !c.err && err != nil {
			t.Error("should not have gotten error", err)
		} else if c.err && err == nil {
			t.Error("should have gotten error")
		}
	}
}

func TestHttpURL(t *testing.T) {
	cases := []struct {
		url string
		err bool
	}{
		{"", true},
		{"abc", true},
		{"abcabcabcabc", true},
		{"             ", true},
		{"a@b.c", true},
		{"http:", true},
		{"http://", true},
		{"https:", true},
		{"https://", true},
		{"https://blah", false},
	}

	for _, c := range cases {
		err := HttpURL(c.url)
		if !c.err && err != nil {
			t.Error("should not have gotten error", err)
		} else if c.err && err == nil {
			t.Error("should have gotten error")
		}
	}
}

func TestDomainName(t *testing.T) {
	cases := []struct {
		domain string
		err    bool
	}{
		{"abc", true},
		{"abc.def", false},
		{"abc.DEF", false},
		{"abc.def.ghi", false},
		{"abc.déf.ghi", true},
		{"0abc.def", false},
		{"a-b-c.def", false},
		{"-abc.def", true},
		{"a_b_c.def", true},
		{"_abc.def", true},
	}

	for _, c := range cases {
		t.Run(c.domain, func(t *testing.T) {
			err := DomainName(c.domain)
			if !c.err && err != nil {
				t.Error("should not have gotten error", err)
			} else if c.err && err == nil {
				t.Error("should have gotten error")
			}
		})
	}
}

func TestLocationKey(t *testing.T) {
	cases := []struct {
		loc string
		err bool
	}{
		{"", true},
		{"as531411051", false},
		{"/a/abs/path/", true},
		{"../relative", true},
		{"tr/../i/ck/", true},
	}

	for _, c := range cases {
		t.Run(c.loc, func(t *testing.T) {
			err := LocationKey(c.loc)
			if !c.err && err != nil {
				t.Error("should not have gotten error", err)
			} else if c.err && err == nil {
				t.Error("should have gotten error")
			}
		})
	}
}

func TestDBName(t *testing.T) {
	cases := []struct {
		db  string
		err bool
	}{
		{"abc", false},
		{"abcabcabcabc", false},
		{"", true},
		{"      ", true},
		{"abc def", true},
		{"abc-def", true},
		{"abc*def", true},
		{"abc/def", true},
		{"abc.def", true},
		{"..def", true},
		//\/:*?"<>|
		{"abc:def", true},
		{"abc?def", true},
		{"abc\"def", true},
		{"abc<def", true},
		{"abc>def", true},
		{"abc|def", true},
	}

	for _, c := range cases {
		err := DBName(c.db)
		if !c.err && err != nil {
			t.Error("should not have gotten error", err)
		} else if c.err && err == nil {
			t.Error("should have gotten error")
		}
	}
}

func TestDropID(t *testing.T) {
	cases := []struct {
		b   string
		err bool
	}{
		{"", false},
		{"abc", false},
		{"abc/def", true},
		{"abcdé", true},
	}

	for _, c := range cases {
		err := DropIDHandle(c.b)
		if !c.err && err != nil {
			t.Error("should not have gotten error", err)
		} else if c.err && err == nil {
			t.Error("should have gotten error")
		}
	}
}

func TestTSNetIDFull(t *testing.T) {
	cases := []struct {
		id  string
		err bool
	}{
		{"123@tailscale.com", false},
		{"123@example.com", false},
		{"123@example", true},
	}

	for _, c := range cases {
		err := TSNetIDFull(c.id)
		if !c.err && err != nil {
			t.Error("should not have gotten error", err)
		} else if c.err && err == nil {
			t.Error("should have gotten error")
		}
	}
}

func TestAppspaceAvatarFilename(t *testing.T) {
	cases := []struct {
		b   string
		err bool
	}{
		{"abc-def.jpg", false},
		{"../abc-def.jpg", true},
		{"abc-d/../../ef.jpg", true},
		{"../../abc-def.jpg", true},
		{"..-..jpg", true},
	}

	for _, c := range cases {
		err := AppspaceAvatarFilename(c.b)
		if !c.err && err != nil {
			t.Error("should not have gotten error", err)
		} else if c.err && err == nil {
			t.Error("should have gotten error")
		}
	}
}

func TestAppspaceBackupFile(t *testing.T) {
	cases := []struct {
		b   string
		err bool
	}{
		{"1234-56-78_1234.zip", false},
		{"1234-56-78_1234_5.zip", false},
		{"12EB-56-78_1234.zip", true},
		{"1234-56-78_1234_.zip", true},
		{"1234-56-78_1234_11.zip", true},
		{".zip", true},
		{"", true},
	}

	for _, c := range cases {
		err := AppspaceBackupFile(c.b)
		if !c.err && err != nil {
			t.Error("should not have gotten error", err)
		} else if c.err && err == nil {
			t.Error("should have gotten error")
		}
	}
}
