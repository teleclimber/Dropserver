package appops

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestValidateAppManifest(t *testing.T) {
	cases := []struct {
		manifest domain.AppVersionManifest
		numErr   int
	}{
		{domain.AppVersionManifest{Name: "blah", Version: "0.0.1"}, 0},
		{domain.AppVersionManifest{Version: "0.0.1"}, 1},
		{domain.AppVersionManifest{Name: "blah"}, 1},
	}

	for _, c := range cases {
		m := domain.AppGetMeta{
			Errors:          make([]string, 0),
			VersionManifest: c.manifest,
		}
		err := validateVersion(&m)
		if err != nil {
			t.Error(err)
		}
		if len(m.Errors) != c.numErr {
			t.Log(m.Errors)
			t.Error("Error count mismatch")
		}
	}
}
