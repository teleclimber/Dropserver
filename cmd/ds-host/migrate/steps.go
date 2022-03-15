package migrate

import (
	"database/sql"
	"errors"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type stepArgs struct {
	db    *domain.DB
	dbErr error
}

func (sa *stepArgs) dbExec(q string, args ...interface{}) sql.Result {
	if sa.dbErr != nil {
		return nil
	}

	handle := sa.db.Handle

	ret, err := handle.Exec(q, args...)
	if err != nil {
		sa.dbErr = errors.New("Error Executing statement: " + q + " " + err.Error()) // use error wrapping here
	}

	return ret
}

// MigrationStep represents a single migration step for host DB
type MigrationStep struct {
	name string
	up   func(*stepArgs) error
	down func(*stepArgs) error
}

var MigrationSteps []MigrationStep = []MigrationStep{{
	name: "1905-fresh-install",
	up:   freshInstallUp,
	down: freshInstallDown},
}
