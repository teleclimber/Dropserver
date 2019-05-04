package appfiles

import (
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/internal/dserror"
)

func TestDecodeAppJsonError(t *testing.T) {
	// check that passing a json does return the struct as expceted
	// check that badly formed json returns the correct error code.
	r := strings.NewReader(`{ "name":"blah", "version":"0.0.1 }`)
	_, err := decodeAppJSON(r)
	if err == nil {
		t.Error("Error was nil")
	} else if err.Code() != dserror.AppConfigParseFailed {
		t.Error("wrong error code", err.Code())
	}
}

func TestDecodeAppJSON(t *testing.T) {
	r := strings.NewReader(`{ "name":"blah", "version":"0.0.1" }`)
	meta, err := decodeAppJSON(r)
	if err != nil {
		t.Error("got error for well formed json")
	} else if meta.AppName != "blah" || meta.AppVersion != "0.0.1" {
		t.Error("got incorrect values for meta")
	}
}

func TestValidateAppMeta(t *testing.T) {
	cases := []struct {
		json string
		err  bool
	}{
		{`{ "name":"blah", "version":"0.0.1" }`, false},
		{`{ "version":"0.0.1" }`, true},
		{`{ "name":"blah", "foo":"0.0.1" }`, true},
	}

	for _, c := range cases {
		r := strings.NewReader(c.json)
		meta, _ := decodeAppJSON(r)

		dsErr := validateAppMeta(meta)
		hasErr := dsErr != nil
		if hasErr != c.err {
			t.Error("error mismatch", meta, dsErr.ExtraMessage())
		}
	}
}
