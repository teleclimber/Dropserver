package server

import (
	"testing"
)

func TestGetSubdomains(t *testing.T) {
	cases := []struct {
		input      string
		rootPieces []string
		subdomains []string
		ok         bool
	}{
		// correct domain, no subdomains:
		{"dropserver.develop", dsDevPieces(), []string{}, true},
		// incorect domain
		{"dropserver.xyz", dsDevPieces(), []string{}, false},
		{"dropserver", dsDevPieces(), []string{}, false},
		// incomplete domain
		{"develop", dsDevPieces(), []string{}, false},
		// correct domain and one subdomain
		{"du-report.dropserver.develop", dsDevPieces(), []string{"du-report"}, true},
		// throw in some capital letters:
		{"du-Report.dropserVer.develoP", dsDevPieces(), []string{"du-report"}, true},
		// correct and two subdomains
		{"foo.du-report.dropserver.develop", dsDevPieces(), []string{"foo", "du-report"}, true},
		// correct and subdomains and throw a :port in
		{"foo.du-report.dropserver.develop:3000", dsDevPieces(), []string{"foo", "du-report"}, true},
		// Single-level domain (like for local dev)
		{"dropserver", []string{"dropserver"}, []string{}, true},
		// three level root domain, no subdomain
		{"dropserver.co.uk", []string{"uk", "co", "dropserver"}, []string{}, true},
		// Three levels, incomplete
		{"co.uk", []string{"uk", "co", "dropserver"}, []string{}, false},
		// Three levels one subdomain
		{"abc.dropserver.co.uk", []string{"uk", "co", "dropserver"}, []string{"abc"}, true},
	}

	for _, c := range cases {
		subdomains, ok := getSubdomains(c.input, c.rootPieces)
		if c.ok != ok {
			t.Errorf("%s: expected OK %t, got %t", c.input, c.ok, ok)
		}
		if !stringSlicesEqual(c.subdomains, subdomains) {
			t.Error(c.input, "expected subdomains / got:", c.subdomains, subdomains)
		}
	}
}
func dsDevPieces() []string {
	return []string{"develop", "dropserver"}
}

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
// from https://yourbasic.org/golang/compare-slices/
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestReverse(t *testing.T) {
	cases := []struct {
		in  []string
		out []string
	}{
		{[]string{"a"}, []string{"a"}},
		{[]string{"a", "b"}, []string{"b", "a"}},
		{[]string{"a", "b", "c"}, []string{"c", "b", "a"}},
	}

	for _, c := range cases {
		reverse(c.in)
		if !stringSlicesEqual(c.in, c.out) {
			t.Error(c.in, "expected / got", c.out, c.in)
		}
	}
}

// TODO need to test ServeHTTP!
