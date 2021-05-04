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
		{"a_b_c.def", false},
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
