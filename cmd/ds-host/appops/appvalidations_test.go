package appops

import (
	"fmt"
	"strings"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestValidateMigrationSequence(t *testing.T) {
	cases := []struct {
		desc       string
		migrations []domain.MigrationStep
		invalid    bool
		poor       bool
	}{
		{
			desc:       "empty array",
			migrations: []domain.MigrationStep{},
		}, {
			desc: "up",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}},
			poor: true,
		}, {
			desc: "up3",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 3}},
			poor: true,
		}, {
			desc: "down",
			migrations: []domain.MigrationStep{
				{Direction: "down", Schema: 1}},
			invalid: true,
		}, {
			desc: "up down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1}},
		}, {
			desc: "up1 up2 down2",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2},
				{Direction: "up", Schema: 1}},
			poor: true,
		}, {
			desc: "up2 up3 down3",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 3}, {Direction: "down", Schema: 3},
				{Direction: "up", Schema: 2}}, // need similar test but away from schema 1
			poor: true,
		}, {
			desc: "up1 up1 down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "up", Schema: 1}},
			invalid: true,
		}, {
			desc: "up down down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2},
				{Direction: "down", Schema: 1}},
			poor: true,
		}, {
			desc: "up gap up down gap down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3}},
			poor: true,
		}, {
			desc: "up up up down gap down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3},
				{Direction: "up", Schema: 2}},
			poor: true,
		}, {
			desc: "up up up down down down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3},
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2}},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			warns := validateMigrationSequence(c.migrations)
			isInvalid := false
			for _, w := range warns {
				if w.Problem == domain.ProblemInvalid {
					isInvalid = true
				}
			}
			if isInvalid != c.invalid {
				t.Errorf("mismatch: expected invalid: %v, got %v", c.invalid, isInvalid)
			}
			isPoor := false
			for _, w := range warns {
				if w.Problem == domain.ProblemPoorExperience {
					isPoor = true
				}
			}
			if isPoor != c.poor {
				t.Errorf("mismatch: expected poor experience: %v, got %v", c.poor, isPoor)
			}
		})
	}
}

func TestValidateVersion(t *testing.T) {
	cases := []struct {
		version domain.Version
		clean   domain.Version
		warn    bool
	}{
		{domain.Version(""), domain.Version(""), true},
	}

	for _, c := range cases {
		clean, warn := validateVersion(c.version)
		if clean != c.clean {
			t.Errorf("clean version not as expected: %v, %v", clean, c.clean)
		}
		if warn == (domain.ProcessWarning{}) && c.warn {
			t.Error("Expected warning")
		}
		if warn != (domain.ProcessWarning{}) && !c.warn {
			t.Errorf("Unexpected warning: %v", warn)
		}
	}
}

func TestBadURLRemoved(t *testing.T) {
	// you should check that all bad data is removed.
	manifest, _ := validateManifest(domain.AppVersionManifest{Website: "blah"})
	if manifest.Website != "" {
		t.Errorf("Expected website to be removed from manifest: %v", manifest.Website)
	}
}

func TestHasWarnings(t *testing.T) {
	warnings := []domain.ProcessWarning{
		{Field: "abc", Problem: domain.ProblemBig},
	}
	cases := []struct {
		field string
		has   bool
	}{
		{"abc", true},
		{"def", false},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case %v %v", i, c.field), func(t *testing.T) {
			got := hasWarnings(c.field, warnings)
			if got != c.has {
				t.Errorf("mismatch: %v, %v", got, c.has)
			}
		})
	}
}

func TestHasProblem(t *testing.T) {
	warnings := []domain.ProcessWarning{
		{Field: "abc", Problem: domain.ProblemBig},
		{Field: "abc", Problem: domain.ProblemInvalid},
		{Field: "def", Problem: domain.ProblemBig},
	}
	cases := []struct {
		field string
		has   bool
	}{
		{"abc", true},
		{"def", false},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case %v %v", i, c.field), func(t *testing.T) {
			got := hasProblem(c.field, domain.ProblemInvalid, warnings)
			if got != c.has {
				t.Errorf("mismatch: %v, %v", got, c.has)
			}
		})
	}
}

func TestValidateSequence(t *testing.T) {
	cases := []struct {
		desc        string
		appVersions []appVersionSemver
		warn        bool
	}{
		{
			desc: "incrementing schema",
			appVersions: []appVersionSemver{
				makeAppVersionSemver("0.8.1", 2),
				makeAppVersionSemver("0.2.1", 0),
			},
		}, {
			desc: "same schema",
			appVersions: []appVersionSemver{
				makeAppVersionSemver("0.8.1", 1),
				makeAppVersionSemver("0.2.1", 1),
			},
		}, {
			desc: "next has lower schema",
			appVersions: []appVersionSemver{
				makeAppVersionSemver("0.8.1", 0),
				makeAppVersionSemver("0.2.1", 1),
			},
			warn: true,
		}, {
			desc: "prev has higher schema",
			appVersions: []appVersionSemver{
				makeAppVersionSemver("0.8.1", 1),
				makeAppVersionSemver("0.2.1", 2),
			},
			warn: true,
		}, {
			desc: "prev only increment schema",
			appVersions: []appVersionSemver{
				makeAppVersionSemver("0.2.1", 0),
			},
		}, {
			desc: "next only increment schema",
			appVersions: []appVersionSemver{
				makeAppVersionSemver("0.8.1", 2),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			_, _, warns := validateVersionSequence(domain.Version("0.5.0"), 1, c.appVersions)
			if len(warns) == 0 && c.warn {
				t.Error("got no warnings, expected some")
			} else if len(warns) != 0 && !c.warn {
				t.Log(warns)
				t.Error("got warnings,none expected")
			}
		})
	}
}

func TestValidateSequenceAlreadyExists(t *testing.T) {
	_, _, warns := validateVersionSequence(domain.Version("0.5.0"), 1, []appVersionSemver{makeAppVersionSemver("0.5.0", 2)})
	if len(warns) != 1 {
		t.Error("expected a warning")
	}
}

func TestValidateSequencePrevNext(t *testing.T) {
	v2 := "0.2.0"
	v3 := "0.3.0"
	v7 := "0.7.0"
	v8 := "0.8.0"

	cases := []struct {
		desc        string
		appVersions []appVersionSemver
		warn        bool
		prev        domain.Version
		next        domain.Version
	}{
		{
			desc: "four versions",
			appVersions: []appVersionSemver{
				makeAppVersionSemver(v2, 1),
				makeAppVersionSemver(v8, 1),
				makeAppVersionSemver(v7, 1),
				makeAppVersionSemver(v3, 1),
			},
			prev: domain.Version(v3),
			next: domain.Version(v7),
		}, {
			desc:        "no versions",
			appVersions: []appVersionSemver{},
		}, {
			desc: "prev only",
			appVersions: []appVersionSemver{
				makeAppVersionSemver(v2, 1),
				makeAppVersionSemver(v3, 1),
			},
			prev: domain.Version(v3),
		}, {
			desc: "next only",
			appVersions: []appVersionSemver{
				makeAppVersionSemver(v8, 1),
				makeAppVersionSemver(v7, 1),
			},
			next: domain.Version(v7),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			p, n, _ := validateVersionSequence(domain.Version("0.5.0"), 1, c.appVersions)
			if p != c.prev {
				t.Errorf("mismatched prev: %v %v", p, c.prev)
			}
			if n != c.next {
				t.Errorf("mismatched next: %v %v", n, c.next)
			}
		})
	}
}

func TestGetValidChangelog(t *testing.T) {
	cases := []struct {
		in  string
		ver string
		out string
	}{{
		"blah\n0.0.1\nx\n1.2.3 \nabc",
		"1.2.3",
		"abc",
	}, {
		"1.2.3 \nabc \n2.3.4\nblah",
		"1.2.3",
		"abc",
	}, {
		"1.2.3 \n\nabc\n\ndef\nghi \n \n2.3.4\nblah",
		"1.2.3",
		"abc\n\ndef\nghi",
	}}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			r := strings.NewReader(c.in)
			sVer, _ := semver.ParseTolerant(c.ver)
			result, err := getValidChangelog(r, sVer)
			if err != nil {
				t.Error(err)
			}
			if result != c.out {
				t.Errorf("expected %v, got -%v-", c.out, result)
			}
		})
	}
}

func makeAppVersionSemver(version string, schema int) appVersionSemver {
	return appVersionSemver{
		domain.AppVersion{
			Version: domain.Version(version),
			Schema:  schema,
		},
		semver.MustParse(version),
	}
}
