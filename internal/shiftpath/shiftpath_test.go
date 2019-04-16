package shiftpath

import (
	"testing"
)

func TestShiftPath(t *testing.T) {

	cases := []struct {
		input string
		head  string
		tail  string
	}{
		{"", "", "/"},
		{"/", "", "/"},
		{"abc/def", "abc", "/def"},
		{"/abc/def", "abc", "/def"},
		{"/abc/def/", "abc", "/def"},
		{"abc/def/", "abc", "/def"},
	}

	for _, c := range cases {
		head, tail := ShiftPath(c.input)
		if head != c.head || tail != c.tail {
			t.Error(c.input + ": incorrect. Exp: " + c.head + " ~ " + c.tail + "\nGot: " + head + " ~ " + tail)
		}
	}
}
