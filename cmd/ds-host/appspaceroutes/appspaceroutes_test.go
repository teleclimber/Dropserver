package appspaceroutes

import (
	"testing"
)

func TestGetAppspaceName(t *testing.T) {
	cases := []struct {
		input    string
		appspace string
		ok       bool
	}{
		{"dropserver.develop", "", false},
		{"dropserver.xyz", "", false},
		{"dropserver", "", false},
		{"du-report.dropserver.develop", "du-report", true},
		{"foo.du-report.dropserver.develop", "du-report", true},
		{"foo.du-report.dropserver.develop:3000", "du-report", true},
	}

	// TODO: cases should take configuration into consideration
	// ..config needs to be set up and injectable.

	for _, c := range cases {
		appspace, ok := getAppspaceName(c.input)
		if c.ok != ok {
			t.Errorf("%s: expected OK %t, got %t", c.input, c.ok, ok)
		}
		if c.appspace != appspace {
			t.Errorf("%s: expected appspace '%s', got '%s'", c.input, c.appspace, appspace)
		}
	}
}

// TODO: test ServeHTTP
// - gets appspace, fails if not there
// - recognizes /dropserver/ as path and forwards accordingly
// - gets app and fails if none
// - ...
