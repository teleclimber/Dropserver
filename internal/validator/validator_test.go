package validator

import (
	"testing"
)

func TestValidateAlphaNumDashRegex(t *testing.T) {
	cases := []struct {
		str string
		res bool
	}{
		{"", true},
		{"1", true},
		{"-", true},
		{"a-", true},
		{"abc123", true},
		{"abc-123", true},
		{"123-abc", true},
		{"-abc-123", true},
		{"---", true},
		{"%", false},
	}

	for _, c := range cases {
		res := alphaNumDashRegex.MatchString(c.str)
		if c.res != res {
			t.Errorf("mismatch for %s, got %v expected %v", c.str, res, c.res)
		}
	}
}

func TestValidateStartAlpha(t *testing.T) {
	cases := []struct {
		str string
		res bool
	}{
		{"", false},
		{"a123", true},
		{"1abc", false},
		{"123-abc", false},
		{"%", false},
	}

	for _, c := range cases {
		res := startAlphaRegex.MatchString(c.str)
		if c.res != res {
			t.Errorf("mismatch for %s, got %v expected %v", c.str, res, c.res)
		}
	}
}

func TestValidateStartAlphaNum(t *testing.T) {
	cases := []struct {
		str string
		res bool
	}{
		{"", false},
		{"a123", true},
		{"1abc", true},
		{"123-abc", true},
		{"%", false},
	}

	for _, c := range cases {
		res := startAlphaNumRegex.MatchString(c.str)
		if c.res != res {
			t.Errorf("mismatch for %s, got %v expected %v", c.str, res, c.res)
		}
	}
}
func TestValidateEndAlphaNum(t *testing.T) {
	cases := []struct {
		str string
		res bool
	}{
		{"", false},
		{"a", true},
		{"1", true},
		{"a123", true},
		{"1abc", true},
		{"123-abc", true},
		{"%", false},
	}

	for _, c := range cases {
		res := endAlphaNumRegex.MatchString(c.str)
		if c.res != res {
			t.Errorf("mismatch for %s, got %v expected %v", c.str, res, c.res)
		}
	}
}

type testControlURL struct {
	ControlURL string `validate:"tsnetcontrolurl"`
}

func TestValidateTSNetControlURLStruct(t *testing.T) {
	cases := []struct {
		str   string
		isErr bool
	}{
		{"", false},
		{"a", false},
		{"foo.bar", false},
	}
	for _, c := range cases {
		err := goVal.Struct(testControlURL{c.str})
		if (err != nil && !c.isErr) || (err == nil && c.isErr) {
			t.Errorf("mismatch for %s, got %v expected %v", c.str, err, c.isErr)
		}
	}
}

type testTSNetMachineName struct {
	MachineName string `validate:"tsnetmachinename"`
}

func TestValidateTSNetMachineNameStruct(t *testing.T) {
	cases := []struct {
		str   string
		isErr bool
	}{
		{"", true},
		{"1", false},
		{"a", false},
		{"abc-123", false},
		{"-abc-123", true},
		{"abc-123-", true},
		{"abc-def-1-abc-def-2-abc-def-3-abc-def-4-abc-def-5-abc-def-6-over-63", true},
		{"%", true},
	}

	for _, c := range cases {
		err := goVal.Struct(testTSNetMachineName{c.str})
		if (err != nil && !c.isErr) || (err == nil && c.isErr) {
			t.Errorf("mismatch for %s, got %v expected %v", c.str, err, c.isErr)
		}
	}
}

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
