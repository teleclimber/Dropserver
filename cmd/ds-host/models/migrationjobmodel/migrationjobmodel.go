package migrationjobmodel

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/nulltypes"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// MigrationJobModel represents the model for app spaces
type MigrationJobModel struct {
	DB *domain.DB

	stmt struct {
		create *sqlx.Stmt
		getJob *sqlx.Stmt
		//selectOwner    *sqlx.Stmt //later
		selectAppspace     *sqlx.Stmt
		getPendingAppspace *sqlx.Stmt
		getPending         *sqlx.Stmt
		setStarted         *sqlx.Stmt
		setFinished        *sqlx.Stmt
		deleteJob          *sqlx.Stmt
	}
}

// PrepareStatements for appspace model
func (m *MigrationJobModel) PrepareStatements() {
	// Here is the place to get clever with statemevts if using multiple DBs.
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	m.stmt.create = p.Prep(`INSERT INTO migrationjobs
		("job_id", "owner_id", "appspace_id", "to_version", "priority", "created") VALUES (NULL, ?, ?, ?, ?, datetime("now"))`)

	m.stmt.getJob = p.Prep(`SELECT * FROM migrationjobs WHERE job_id = ?`)

	m.stmt.getPendingAppspace = p.Prep(`SELECT * FROM migrationjobs WHERE appspace_id = ? AND started IS NULL`)

	m.stmt.getPending = p.Prep(`SELECT * FROM migrationjobs WHERE started IS NULL
		ORDER BY priority DESC, created DESC`)

	m.stmt.setStarted = p.Prep(`UPDATE migrationjobs SET started = datetime("now") WHERE job_id = ? AND started IS NULL`)

	m.stmt.setFinished = p.Prep(`UPDATE migrationjobs SET finished = datetime("now"), error = ? WHERE job_id = ? AND started IS NOT NULL AND finished IS NULL`)

	m.stmt.deleteJob = p.Prep(`DELETE FROM migrationjobs WHERE job_id = ?`)
}

// create job
// get job for appspaceid
// get a job to execute (marks as started?)
// mark as started
// mark as complete (or just delete it?)

// Create adds a job to the queue
// It replaces any pending job for same appspace
func (m *MigrationJobModel) Create(ownerID domain.UserID, appspaceID domain.AppspaceID, toVersion domain.Version, priority bool) (*domain.MigrationJob, error) {
	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		m.getLogger("Create(), Beginx()").Error(err)
		tx.Rollback()
		return nil, err
	}

	var job domain.MigrationJob
	get := tx.Stmtx(m.stmt.getPendingAppspace)
	err = get.QueryRowx(appspaceID).StructScan(&job)
	if err != nil && err != sql.ErrNoRows {
		m.getLogger("Create(), QueryRowx()").Error(err)
		tx.Rollback()
		return nil, err
	}
	if err == nil { // means it got a row, right?
		del := tx.Stmtx(m.stmt.deleteJob)
		_, err := del.Exec(job.JobID)
		if err != nil {
			m.getLogger("Create(), del.Exec()").Error(err)
			tx.Rollback()
			return nil, err
		}
	}

	create := tx.Stmtx(m.stmt.create)

	r, err := create.Exec(ownerID, appspaceID, toVersion, priority)
	if err != nil {
		m.getLogger("Create(), create.Exec()").Error(err)
		tx.Rollback()
		return nil, err
	}

	jobID, err := r.LastInsertId()
	if err != nil {
		m.getLogger("Create(), LastInsertId()").Error(err)
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		m.getLogger("Create(), tx.Commit()").Error(err)
		return nil, err
	}

	return m.GetJob(domain.JobID(jobID))
}

// GetJob returns job from its job id.
// Errors if job not found.
func (m *MigrationJobModel) GetJob(jobID domain.JobID) (*domain.MigrationJob, error) {
	var ret domain.MigrationJob
	err := m.stmt.getJob.QueryRowx(jobID).StructScan(&ret)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNoRowsInResultSet
		}
		m.getLogger("GetJob()").Error(err)
		return nil, err
	}
	return &ret, nil
}

// GetForAppspace returns an appspace's job if there is one
// Returns nil, nil if no job is found
// Should it return finished jobs?
// Should it return jobs that have been started?
// func (m *MigrationJobModel) GetForAppspace(appspaceID domain.AppspaceID) (*domain.MigrationJob, error) {
// 	var job domain.MigrationJob

// 	err := m.stmt.selectAppspace.QueryRowx(appspaceID).StructScan(&job)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	return &job, nil
// }

// GetPending returns an array of pending jobs
func (m *MigrationJobModel) GetPending() ([]*domain.MigrationJob, error) {
	ret := []*domain.MigrationJob{}

	err := m.stmt.getPending.Select(&ret)
	if err != nil && err != sql.ErrNoRows {
		m.getLogger("GetPending()").Error(err)
		return nil, err
	}

	return ret, nil
}

// SetStarted attempts to set the started date to now,
// but returns ok=false if no rows were changed (in the case of deleted job)
func (m *MigrationJobModel) SetStarted(jobID domain.JobID) (bool, error) {
	// Just set started, though we have to ensure the job is still there too.
	// maybe we can check result to see if we've effectively changed one line
	// and craft the update so that it only works if started is null
	// return false, nil in case of no-change and caller can manage and start another one.
	r, err := m.stmt.setStarted.Exec(jobID)
	if err != nil {
		m.getLogger("SetStarted(), setStarted.Exec()").Error(err)
		return false, err
	}
	num, err := r.RowsAffected()
	if err != nil {
		m.getLogger("SetStarted(), r.RowsAffected").Error(err)
		return false, err
	}
	if num != 1 {
		return false, nil
	}
	return true, nil
}

// SetFinished puts the current time in finished column, and an error string if there is one
func (m *MigrationJobModel) SetFinished(jobID domain.JobID, errStr nulltypes.NullString) error {
	r, err := m.stmt.setFinished.Exec(errStr, jobID)
	if err != nil {
		m.getLogger("SetFinished(), setFinished.Exec()").Error(err)
		return err
	}
	num, err := r.RowsAffected()
	if err != nil {
		m.getLogger("SetFinished(), RowsAffected").Error(err)
		return err
	}
	if num != 1 {
		return domain.ErrNoRowsAffected
	}
	return nil
}

// Delete removes a job, indicating it was completed or no longer desired.
// TODO: replace with purge or something.
// func (m *MigrationJobModel) Delete(appspaceID domain.AppspaceID) error {
// 	_, err := m.stmt.delete.Exec(appspaceID)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func (m *MigrationJobModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("MigrationJobModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
