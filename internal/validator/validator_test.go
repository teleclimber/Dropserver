package validator

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

func TestPassword(t *testing.T) {
	v := &Validator{}
	v.Init()

	valErr := dserror.New(dserror.InputValidationError)
	cases := []struct {
		pw    string
		dsErr domain.Error
	}{
		{"", valErr},
		{"abc", valErr},
		{"abcabcabcabc", nil},
		{"             ", nil},
	}

	for _, c := range cases {
		dsErr := v.Password(c.pw)
		if c.dsErr == nil && dsErr != nil {
			t.Error("should not have gotten error", dsErr)
		} else if c.dsErr != nil && dsErr == nil {
			t.Error("should have gotten error")
		} else if c.dsErr != dsErr {
			t.Error("wrong error", dsErr)
		}
	}
}

func TestEmail(t *testing.T) {
	v := &Validator{}
	v.Init()

	valErr := dserror.New(dserror.InputValidationError)
	cases := []struct {
		email string
		dsErr domain.Error
	}{
		{"", valErr},
		{"abc", valErr},
		{"abcabcabcabc", valErr},
		{"             ", valErr},
		{"a@b.c", nil},
	}

	for _, c := range cases {
		dsErr := v.Email(c.email)
		if c.dsErr == nil && dsErr != nil {
			t.Error("should not have gotten error", dsErr)
		} else if c.dsErr != nil && dsErr == nil {
			t.Error("should have gotten error")
		} else if c.dsErr != dsErr {
			t.Error("wrong error", dsErr)
		}
	}
}

func TestDBName(t *testing.T) {
	v := &Validator{}
	v.Init()

	valErr := dserror.New(dserror.InputValidationError)
	cases := []struct {
		db    string
		dsErr domain.Error
	}{
		{"abc", nil},
		{"abcabcabcabc", nil},
		{"", valErr},
		{"      ", valErr},
		{"abc def", valErr},
		{"abc-def", valErr},
		{"abc*def", valErr},
		{"abc/def", valErr},
		{"abc.def", valErr},
		{"..def", valErr},
		//\/:*?"<>|
		{"abc:def", valErr},
		{"abc?def", valErr},
		{"abc\"def", valErr},
		{"abc<def", valErr},
		{"abc>def", valErr},
		{"abc|def", valErr},
	}

	for _, c := range cases {
		dsErr := v.DBName(c.db)
		if c.dsErr == nil && dsErr != nil {
			t.Error("should not have gotten error", dsErr)
		} else if c.dsErr != nil && dsErr == nil {
			t.Error("should have gotten error")
		} else if c.dsErr != dsErr {
			t.Error("wrong error", dsErr)
		}
	}
}
