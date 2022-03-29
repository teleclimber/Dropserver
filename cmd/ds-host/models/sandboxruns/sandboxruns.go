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

	m.stmt.update = p.Prep(`UPDATE sandbox_runs SET end = ?, cpu_time = ?, memory = ? WHERE sandbox_id = ?`)
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

func (m *SandboxRunsModel) Update(sandboxID int, cpuTime int, memory int) error {
	err := m.update(sandboxID, nil, cpuTime, memory)
	if err != nil {
		m.getLogger("Update()").Error(err)
	}
	return err
}

func (m *SandboxRunsModel) End(sandboxID int, end time.Time, cpuTime int, memory int) error {
	err := m.update(sandboxID, end, cpuTime, memory)
	if err != nil {
		m.getLogger("End()").Error(err)
	}
	return err
}

func (m *SandboxRunsModel) update(sandboxID int, end interface{}, cpuTime int, memory int) error {
	var id string
	err := m.stmt.checkID.QueryRowx(sandboxID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("sandbox id not in database")
		}
		return err
	}
	_, err = m.stmt.update.Exec(end, cpuTime, memory, sandboxID)
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

func (m *SandboxRunsModel) GetAppspace(ownerID domain.UserID, appspaceID domain.AppspaceID) ([]domain.SandboxRun, error) {
	ret := []domain.SandboxRun{}

	err := m.stmt.selectAppspace.Select(&ret, ownerID, appspaceID)
	if err != nil {
		m.getLogger("GetAppspace()").UserID(ownerID).AppspaceID(appspaceID).Error(err)
		return nil, err
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
