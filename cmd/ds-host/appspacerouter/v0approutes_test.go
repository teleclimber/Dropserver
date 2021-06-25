package appspacerouter

import (
	"testing"

	pathToRegexp "github.com/soongo/path-to-regexp"
)

func TestP2R(t *testing.T) {
	toEnd := false
	r, err := pathToRegexp.PathToRegexp("/abc", nil, &pathToRegexp.Options{End: &toEnd})
	if err != nil {
		t.Error(err)
	}

	m, err := r.FindStringMatch("/abc/def")
	if err != nil {
		t.Error(err)
	}
	if m == nil {
		t.Error("expected a match")
	}
	if len(m.Groups()) != 1 {
		t.Error("expected 1 group")
	}

	m, err = r.FindStringMatch("/uvw/def")
	if err != nil {
		t.Error(err)
	}
	if m != nil {
		t.Error("expected no match")
	}

	m, err = r.FindStringMatch("/ab")
	if err != nil {
		t.Error(err)
	}
	if m != nil {
		t.Error("expected no match")
	}
}

func TestP2RTokens(t *testing.T) {
	var tokens []pathToRegexp.Token
	r, err := pathToRegexp.PathToRegexp("/abc/:id", &tokens, nil)
	if err != nil {
		t.Error(err)
	}
	if len(tokens) != 1 {
		t.Error("expected 1 token")
	}

	m, err := r.FindStringMatch("/abc")
	if err != nil {
		t.Error(err)
	}
	if m != nil {
		t.Error("expected no match")
	}

	m, err = r.FindStringMatch("/abc/")
	if err != nil {
		t.Error(err)
	}
	if m != nil {
		t.Error("expected no match")
	}

	m, err = r.FindStringMatch("/abc/x")
	if err != nil {
		t.Error(err)
	}
	if m == nil {
		t.Error("expected a match")
	}
	groups := m.Groups()
	if len(groups) != 2 {
		t.Error("expected 2 group")
	}
	group := groups[1]
	if group.String() != "x" {
		t.Error("expected x " + group.String())
	}

}

func TestP2RMatch(t *testing.T) {
	abcIDMatch, err := pathToRegexp.Match("/abc/:id", nil)
	if err != nil {
		t.Error(err)
	}

	m, err := abcIDMatch("/abc/x")
	if err != nil {
		t.Error(err)
	}
	if m == nil {
		t.Error("expected a match")
		return
	}
	// if m.Path != "/abc/:id" {
	// 	t.Error("expected the the router path " + m.Path)
	// }
	id, ok := m.Params["id"]
	if !ok {
		t.Error("aw no id in params")
	}
	if id != "x" {
		t.Error("was hoping for id to be x")
	}

}

///////////////
// Need actual tests for V0approutes
