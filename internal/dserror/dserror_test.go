package dserror

import (
	"errors"
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestNew(t *testing.T) {
	New(AppConfigNotFound) //not sending a extra should not cause panic
}

// Important thing to test with PublicString is whether internal error extras
// are hidden from public view.
func TestPublicString(t *testing.T) {
	cases := []struct {
		code   domain.ErrorCode
		extra  string
		public bool
	}{
		{InternalError, "abc", false},
		{AppConfigNotFound, "abc", true},
	}

	for _, c := range cases {
		err := New(c.code, c.extra)
		str := err.PublicString()
		isPublic := strings.Contains(str, c.extra)
		if isPublic != c.public {
			t.Error("mismatch on visibility: ", c, str)
		}
	}
}

func TestEncoded(t *testing.T) {
	cases := []struct{
		input string
		ret bool
	}{
		{"ds-error:123:abc", true},
		{"gobbidy-goop", false},
	}

	for _, c := range cases {
		err := errors.New(c.input)
		ret := Encoded(err)
		if ret != c.ret {
			t.Error("mismatch in return value", c.input, ret)
		}
	}
}

func TestFromStandard(t *testing.T) {
	cases := []struct {
		input string
		code  domain.ErrorCode
		extra string
	}{
		{"ds-error:123:abc", domain.ErrorCode(123), "abc"},
		{"ds-error:123:", domain.ErrorCode(123), ""},
		{"ds-error:1:abc", InternalError, "abc"},
		{"gobbidy-goop", InternalError, "gobbidy-goop"},
		{"ds-error::abc", InternalError, "ds-error::abc"},
	}

	for _, c := range cases {
		err := errors.New(c.input)
		dsErr := FromStandard(err)
		if dsErr.code != c.code {
			t.Error("code does not match", dsErr, c)
		}
		if dsErr.extraMessage != c.extra {
			t.Error("Extra message does not match", dsErr, c)
		}
	}
}
func TestToStandard(t *testing.T) {
	cases := []struct {
		inputCode  domain.ErrorCode
		inputExtra string
		output     string
	}{
		{InternalError, "foo", "ds-error:1:foo"},
		{AppConfigNotFound, "", "ds-error:3201:"},
	}

	for _, c := range cases {
		dsErr := New(c.inputCode, c.inputExtra)
		err := dsErr.ToStandard()
		if err.Error() != c.output {
			t.Error("mismatched output string", err.Error(), c.output)
		}
	}
}
