package appops

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestValidateMigrationSteps(t *testing.T) {
	cases := []struct {
		desc       string
		migrations []domain.MigrationStep
		isErr      bool
		isWarn     bool
	}{
		{
			desc:       "empty array",
			migrations: []domain.MigrationStep{},
			isErr:      false,
			isWarn:     false,
		}, {
			desc: "up",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}},
			isErr:  false,
			isWarn: false, // no warning because down from 1 is not needed
		}, {
			desc: "up3",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 3}},
			isErr:  false,
			isWarn: true,
		}, {
			desc: "down",
			migrations: []domain.MigrationStep{
				{Direction: "down", Schema: 1}},
			isErr:  true,
			isWarn: true,
		}, {
			desc: "up down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1}},
			isErr:  false,
			isWarn: false,
		}, {
			desc: "up1 up2 down2",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2},
				{Direction: "up", Schema: 1}},
			isErr:  false,
			isWarn: false, // no warning bc no need for down from 1
		}, {
			desc: "up2 up3 down3",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 3}, {Direction: "down", Schema: 3},
				{Direction: "up", Schema: 2}}, // need similar test but away from schema 1
			isErr:  false,
			isWarn: true,
		}, {
			desc: "up1 up1 down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "up", Schema: 1}},
			isErr:  true,
			isWarn: false,
		}, {
			desc: "up down down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2},
				{Direction: "down", Schema: 1}},
			isErr:  false,
			isWarn: true,
		}, {
			desc: "up gap up down gap down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3}},
			isErr:  false,
			isWarn: true,
		}, {
			desc: "up up up down gap down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3},
				{Direction: "up", Schema: 2}},
			isErr:  false,
			isWarn: true,
		}, {
			desc: "up up up down down down",
			migrations: []domain.MigrationStep{
				{Direction: "up", Schema: 1}, {Direction: "down", Schema: 1},
				{Direction: "down", Schema: 3}, {Direction: "up", Schema: 3},
				{Direction: "up", Schema: 2}, {Direction: "down", Schema: 2}},
			isErr:  false,
			isWarn: false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			meta := domain.AppGetMeta{
				Warnings:        make(map[string]string),
				VersionManifest: domain.AppVersionManifest{Migrations: c.migrations},
			}
			err := validateMigrationSteps(&meta)
			if err != nil {
				t.Error(err)
			}
			if (len(meta.Errors) == 0) == c.isErr {
				t.Errorf("mismatch between error and expected: %v %v", c.isErr, meta.Errors)
			}
			if (meta.Warnings["migrations"] == "") == c.isWarn {
				t.Errorf("mismatch between warnings and expected: %v %v", c.isWarn, meta.Warnings["migrations"])
			}
		})
	}
}

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

func TestBadURLRemoved(t *testing.T) {
	m := domain.AppGetMeta{
		Errors:   make([]string, 0),
		Warnings: make(map[string]string),
		VersionManifest: domain.AppVersionManifest{
			Website: "blah"},
	}
	validateSoftData(&m)
	if m.VersionManifest.Website != "" {
		t.Error("Expected Website to be blank")
	}
}
