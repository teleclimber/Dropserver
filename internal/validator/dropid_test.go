package validator

import (
	"testing"
)

func TestKeySplitJoin(t *testing.T) {
	cases := []struct {
		key    string
		handle string
		domain string
		joined string
	}{
		{"abc.def/oli", "oli", "abc.def", "abc.def/oli"},
		{"abc.def", "", "abc.def", "abc.def/"}, // Note joined is different from key!
		{"abc.def/", "", "abc.def", "abc.def/"},
		{"abc.def/oli/er", "oli/er", "abc.def", "abc.def/oli/er"},
	}

	for _, c := range cases {
		h, d := SplitDropID(c.key)
		if h != c.handle || d != c.domain {
			t.Errorf("Mismatch: %v - %v ; %v %v", h, c.handle, d, c.domain)
		}
		joined := JoinDropID(h, d)
		if joined != c.joined {
			t.Errorf("Joined different: %v != %v ", joined, c.joined)
		}
	}
}
