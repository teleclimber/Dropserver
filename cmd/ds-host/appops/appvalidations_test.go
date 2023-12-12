package appops

import (
	"fmt"
	"testing"

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
