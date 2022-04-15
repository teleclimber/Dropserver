package sandboxruns

// Stores cookies in DB.
// This is so that they can be retrieved by user ID or by appspace ID
// Allows mass logouts of user or appspace.
// A fast in-memory cache will alleviate performance problems.

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// CookieModel stores and retrives cookies for you
type SandboxRunsModel struct {
	DB *domain.DB
	// need config to select db type?

	stmt struct {
		checkID        *sqlx.Stmt
		selectOwner    *sqlx.Stmt
		selectApp      *sqlx.Stmt
		selectAppspace *sqlx.Stmt
		insert         *sqlx.Stmt
		update         *sqlx.Stmt
		end            *sqlx.Stmt
		sumAppspace    *sqlx.Stmt
	}
}

// PrepareStatements for appspace model
func (m *SandboxRunsModel) PrepareStatements() {
	// Here is the place to get clever with statements if using multiple DBs.
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	m.stmt.checkID = p.Prep(`SELECT sandbox_id FROM sandbox_runs WHERE sandbox_id = ?`)
	m.stmt.selectOwner = p.Prep(`SELECT * FROM sandbox_runs WHERE owner_id = ?`)
	m.stmt.selectApp = p.Prep(`SELECT * FROM sandbox_runs WHERE owner_id = ? AND app_id = ?`)
	m.stmt.selectAppspace = p.Prep(`SELECT * FROM sandbox_runs WHERE owner_id = ? AND appspace_id = ?`)
	//m.stmt.selectApp = p.Prep(`SELECT * FROM sandbox_runs WHERE app_id = ?`)
	// Maybe a selectNonAppspace makes more sense? We'll see what the UI calls for.

	m.stmt.insert = p.Prep(`INSERT INTO sandbox_runs
		(instance, local_id, owner_id, app_id, version, appspace_id, operation, cgroup, start ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)

	m.stmt.update = p.Prep(`UPDATE sandbox_runs SET end = ?, tied_up_ms = ?, cpu_usec = ?, memory_bytes = ? WHERE sandbox_id = ?`)

	m.stmt.sumAppspace = p.Prep(`SELECT 
		IFNULL(SUM(tied_up_ms), 0) as tied_up_ms,
		IFNULL(SUM(cpu_usec), 0) as cpu_usec,
		IFNULL(SUM(tied_up_ms*memory_bytes), 0) as memory_byte_ms
		FROM sandbox_runs 
		WHERE owner_id = ? AND appspace_id = ?
		AND start >= ? AND start < ?`)
}

func (m *SandboxRunsModel) Create(run domain.SandboxRunIDs, start time.Time) (int, error) {
	result, err := m.stmt.insert.Exec(run.Instance, run.LocalID, run.OwnerID, run.AppID, run.Version, run.AppspaceID, run.Operation, run.CGroup, start)
	if err != nil {
		m.getLogger("Create() insert").Error(err)
		return 0, err
	}
	lastID, err := result.LastInsertId()
	if err != nil {
		m.getLogger("Create() getLastInsertID").Error(err)
		return 0, err
	}
	return int(lastID), nil
}

func (m *SandboxRunsModel) Update(sandboxID int, tiedUpMs int, cpuUsec int, memBytes int) error {
	err := m.update(sandboxID, nil, tiedUpMs, cpuUsec, memBytes)
	if err != nil {
		m.getLogger("Update()").Error(err)
	}
	return err
}

func (m *SandboxRunsModel) End(sandboxID int, end time.Time, tiedUpMs int, cpuUsec int, memBytes int) error {
	err := m.update(sandboxID, end, tiedUpMs, cpuUsec, memBytes)
	if err != nil {
		m.getLogger("End()").Error(err)
	}
	return err
}

func (m *SandboxRunsModel) update(sandboxID int, end interface{}, tiedUpMs int, cpuUsec int, memBytes int) error {
	var id string
	err := m.stmt.checkID.QueryRowx(sandboxID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("sandbox id not in database")
		}
		return err
	}
	_, err = m.stmt.update.Exec(end, tiedUpMs, cpuUsec, memBytes, sandboxID)
	if err != nil {
		return err
	}
	return nil
}
func (m *SandboxRunsModel) GetApp(ownerID domain.UserID, appID domain.AppID) ([]domain.SandboxRun, error) {
	ret := []domain.SandboxRun{}

	err := m.stmt.selectApp.Select(&ret, ownerID, appID)
	if err != nil {
		m.getLogger("GetApp()").UserID(ownerID).AppID(appID).Error(err)
		return nil, err
	}
	return ret, nil
}

// TODO this is far too open-ended. Need a time range.
func (m *SandboxRunsModel) GetAppspace(ownerID domain.UserID, appspaceID domain.AppspaceID) ([]domain.SandboxRun, error) {
	ret := []domain.SandboxRun{}

	err := m.stmt.selectAppspace.Select(&ret, ownerID, appspaceID)
	if err != nil {
		m.getLogger("GetAppspace()").UserID(ownerID).AppspaceID(appspaceID).Error(err)
		return nil, err
	}
	return ret, nil
}

// aggregate data getters:
// - for appspace
// - for user
// With arbitrary Time range.

func (m *SandboxRunsModel) AppsaceSums(ownerID domain.UserID, appspaceID domain.AppspaceID, from time.Time, to time.Time) (domain.SandboxRunSums, error) {
	var ret domain.SandboxRunSums

	err := m.stmt.sumAppspace.QueryRowx(ownerID, appspaceID, from, to).StructScan(&ret)
	if err != nil {
		m.getLogger("AppsaceSums()").UserID(ownerID).AppspaceID(appspaceID).Error(err)
		return ret, err
	}

	return ret, nil
}

func (m *SandboxRunsModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("SandboxRunsModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
