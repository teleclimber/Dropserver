package migrate

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// The way migration steps are declared it is possible
// for string representation of steps to be accidentally used twice.
// This would wreck havoc on the migration process.
// This test checks for this error.
func TestDuplicateStrings(t *testing.T) {
	strs := map[string]bool{}
	for _, s := range MigrationSteps {
		_, ok := strs[s.name]
		if ok {
			t.Error("Duplicate string: " + s.name)
			break
		}
		strs[s.name] = true
	}
}

// Test that steps return an error on db error
func TestStepsCheckDBError(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	db := &domain.DB{
		Handle: handle}

	args := &stepArgs{
		db: db}

	for _, s := range MigrationSteps {
		args.dbErr = errors.New("db error")
		err := s.up(args)
		if err == nil {
			t.Error("step should have returned an error", s)
		}
	}
}

// Test migration steps against an in-memory DB
func TestAllSteps(t *testing.T) {
	// Beware of in-memory DBs: they vanish as soon as the connection closes!
	// We may be able to start a sqlx transaction to avoid problems with that?
	// See: https://github.com/jmoiron/sqlx/issues/164
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	db := &domain.DB{
		Handle: handle}

	args := &stepArgs{
		db: db}

	for _, s := range MigrationSteps {
		err := s.up(args)
		if err != nil {
			t.Error("Step returned an error", s, err)
		}
	}
}

func runMigrationUpTo(t *testing.T, args *stepArgs, to string) {
	for _, s := range MigrationSteps {
		err := s.up(args)
		if err != nil {
			t.Fatal("Step returned an error", s, err)
		}
		if s.name == to {
			break
		}
	}
	if args.dbErr != nil {
		t.Fatal(args.dbErr)
	}
}
