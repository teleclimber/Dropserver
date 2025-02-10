package userroutes

import (
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestValidateAuthStrings(t *testing.T) {
	_, err := validateAuthStrings("foo", "bar")
	if err == nil {
		t.Error("expected error")
	}

	_, err = validateAuthStrings("dropid", "site.com/abc")
	if err != nil {
		t.Error(err)
	}
	_, err = validateAuthStrings("tsnetid", "abc@site.com")
	if err != nil {
		t.Error(err)
	}
}

func TestGetEditAuth(t *testing.T) {
	_, err := getEditAuth(PostAuth{}, true)
	if err != errNoOp {
		t.Error("expected err No Op")
	}

	_, err = getEditAuth(PostAuth{Op: "remove"}, false)
	if err == nil {
		t.Error("expected an error")
	} else if !strings.Contains(err.Error(), "allowed") {
		t.Errorf("expected error about not allowed, got %v", err)
	}

	_, err = getEditAuth(PostAuth{Op: "gibberish"}, true)
	if err == nil {
		t.Error("expected an error")
	} else if !strings.Contains(err.Error(), "unknown operation") {
		t.Errorf("expected unknown operation error, got %v", err)
	}

	auth, err := getEditAuth(PostAuth{Op: "add", Type: "dropid", Identifier: "siTe.cOm/aBc"}, true)
	if err != nil {
		t.Error(err)
	}
	if auth.Operation != domain.EditOperationAdd || auth.Type != "dropid" || auth.Identifier != "site.com/abc" {
		t.Errorf("Got wrong auth: %s, %s, %s", auth.Operation, auth.Type, auth.Identifier)
	}
}
