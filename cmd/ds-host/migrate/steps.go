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
	name                 string
	up                   func(*stepArgs) error
	down                 func(*stepArgs) error
	appspaceMetaDBSchema int
}

var MigrationSteps []MigrationStep = []MigrationStep{{
	name:                 "1905-fresh-install",
	up:                   freshInstallUp,
	down:                 freshInstallDown,
	appspaceMetaDBSchema: 0,
}, {
	name:                 "2203-sandboxusage",
	up:                   sandboxUsageUp,
	down:                 sandboxUsageDown,
	appspaceMetaDBSchema: 0,
}, {
	name:                 "2305-packagedapps",
	up:                   packagedAppsUp,
	down:                 packagedAppsDown,
	appspaceMetaDBSchema: 0,
}, {
	name:                 "2311-appurls",
	up:                   appsFromURLsUp,
	down:                 appsFromURLsDown,
	appspaceMetaDBSchema: 0,
}, {
	name:                 "2408-tailscale",
	up:                   tailscaleIntegrationUp,
	down:                 tailscaleIntegrationDown,
	appspaceMetaDBSchema: 1,
},
}
