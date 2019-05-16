package migrate

import (
	"database/sql"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// this could potentially be a linked list or simething,
// ..and could be created in procedurlal code, which could catch duplicated

// type MigrationStepI interface {
// 	Up(*Migrator) domain.Error
// 	Down(*Migrator) domain.Error
// }

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
		sa.dbErr = err
	}

	return ret
}

// MigrationStep represents a single step
// It can be up or down
type migrationStep struct {
	up   func(*stepArgs) domain.Error
	down func(*stepArgs) domain.Error
}

// Do we really want to pass Migrator?
// -> Better to pass something more cut-down.
// Like DB handle,
// and a filesystem abstraction for when that becomes relevant.

// OrderedSteps lists the steps as strings in order
// The last step is understood to be the code's required schema
var OrderedSteps = []string{
	"1905-fresh-install",
}

// StringSteps gives you a migrationStep for a string
var StringSteps = map[string]migrationStep{
	"1905-fresh-install": freshInstall,
}


