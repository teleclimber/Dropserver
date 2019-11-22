package migrationjobmodel

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

// MigrationJobModel represents the model for app spaces
type MigrationJobModel struct {
	DB     *domain.DB
	Logger domain.LogCLientI

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

	var err error

	m.stmt.create, err = m.DB.Handle.Preparex(`INSERT INTO migrationjobs
		("job_id", "owner_id", "appspace_id", "to_version", "priority", "created") VALUES (NULL, ?, ?, ?, ?, datetime("now"))`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement INSERT INTO migrationjobs..."+err.Error())
		panic(err)
	}

	m.stmt.getJob, err = m.DB.Handle.Preparex(`SELECT * FROM migrationjobs WHERE job_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement getJob"+err.Error())
		panic(err)
	}

	m.stmt.getPendingAppspace, err = m.DB.Handle.Preparex(`SELECT * FROM migrationjobs WHERE appspace_id = ? AND started IS NULL`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement SELECT pending for appspace migrationjobs..."+err.Error())
		panic(err)
	}

	m.stmt.getPending, err = m.DB.Handle.Preparex(`SELECT * FROM migrationjobs WHERE started IS NULL
		ORDER BY priority DESC, created DESC`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement SELECT next migrationjobs..."+err.Error())
		panic(err)
	}

	m.stmt.setStarted, err = m.DB.Handle.Preparex(`UPDATE migrationjobs SET started = datetime("now") WHERE job_id = ? AND started IS NULL`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement UPDATE migrationjobs SET started..."+err.Error())
		panic(err)
	}

	m.stmt.setFinished, err = m.DB.Handle.Preparex(`UPDATE migrationjobs SET finished = datetime("now"), error = ? WHERE job_id = ? AND started IS NOT NULL AND finished IS NULL`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement UPDATE migrationjobs SET started..."+err.Error())
		panic(err)
	}

	m.stmt.deleteJob, err = m.DB.Handle.Preparex(`DELETE FROM migrationjobs WHERE job_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement DELETE FROM migrationjobs..."+err.Error())
		panic(err)
	}
}

// create job
// get job for appspaceid
// get a job to execute (marks as started?)
// mark as started
// mark as complete (or just delete it?)

// Create adds a job to the queue
// It replaces any pending job for same appspace
func (m *MigrationJobModel) Create(ownerID domain.UserID, appspaceID domain.AppspaceID, toVersion domain.Version, priority bool) (*domain.MigrationJob, domain.Error) {
	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		tx.Rollback()
		return nil, dserror.FromStandard(err)
	}

	var job domain.MigrationJob
	get := tx.Stmtx(m.stmt.getPendingAppspace)
	err = get.QueryRowx(appspaceID).StructScan(&job)
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return nil, dserror.FromStandard(err)
	}
	if err == nil { // means it got a row, right?
		del := tx.Stmtx(m.stmt.deleteJob)
		_, err := del.Exec(job.JobID)
		if err != nil {
			tx.Rollback()
			return nil, dserror.FromStandard(err)
		}
	}

	create := tx.Stmtx(m.stmt.create)

	r, err := create.Exec(ownerID, appspaceID, toVersion, priority)
	if err != nil {
		tx.Rollback()
		return nil, dserror.FromStandard(err)
	}

	jobID, err := r.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, dserror.FromStandard(err)
	}

	tx.Commit()

	return m.GetJob(domain.JobID(jobID))
}

// GetJob returns job from its job id.
// Errors if job not found.
func (m *MigrationJobModel) GetJob(jobID domain.JobID) (*domain.MigrationJob, domain.Error) {
	var ret domain.MigrationJob
	err := m.stmt.getJob.QueryRowx(jobID).StructScan(&ret)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dserror.New(dserror.NoRowsInResultSet)
		}
		return nil, dserror.FromStandard(err)
	}
	return &ret, nil
}

// GetForAppspace returns an appspace's job if there is one
// Returns nil, nil if no job is found
// Should it return finished jobs?
// Should it return jobs that have been started?
// func (m *MigrationJobModel) GetForAppspace(appspaceID domain.AppspaceID) (*domain.MigrationJob, domain.Error) {
// 	var job domain.MigrationJob

// 	err := m.stmt.selectAppspace.QueryRowx(appspaceID).StructScan(&job)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, dserror.FromStandard(err)
// 	}

// 	return &job, nil
// }

// GetPending returns an array of pending jobs
func (m *MigrationJobModel) GetPending() ([]*domain.MigrationJob, domain.Error) {
	ret := []*domain.MigrationJob{}

	err := m.stmt.getPending.Select(&ret)
	if err != nil && err != sql.ErrNoRows {
		return nil, dserror.FromStandard(err)
	}

	return ret, nil
}

// SetStarted attempts to set the started date to now,
// but returns ok=false if no rows were changed (in the case of deleted job)
func (m *MigrationJobModel) SetStarted(jobID domain.JobID) (bool, domain.Error) {
	// Just set started, though we have to ensure the job is still there too.
	// maybe we can check result to see if we've effectively changed one line
	// and craft the update so that it only works if started is null
	// return false, nil in case of no-change and caller can manage and start another one.
	r, err := m.stmt.setStarted.Exec(jobID)
	if err != nil {
		return false, dserror.FromStandard(err)
	}
	num, err := r.RowsAffected()
	if err != nil {
		return false, dserror.FromStandard(err)
	}
	if num != 1 {
		return false, nil
	}
	return true, nil
}

// SetFinished puts the current time in finished column, and an error string if there is one
func (m *MigrationJobModel) SetFinished(jobID domain.JobID, errStr nulltypes.NullString) domain.Error {
	r, err := m.stmt.setFinished.Exec(errStr, jobID)
	if err != nil {
		return dserror.FromStandard(err)
	}
	num, err := r.RowsAffected()
	if err != nil {
		return dserror.FromStandard(err)
	}
	if num != 1 {
		return dserror.New(dserror.NoRowsAffected)
	}
	return nil
}

// Delete removes a job, indicating it was completed or no longer desired.
// TODO: replace with purge or something.
// func (m *MigrationJobModel) Delete(appspaceID domain.AppspaceID) domain.Error {
// 	_, err := m.stmt.delete.Exec(appspaceID)
// 	if err != nil {
// 		return dserror.FromStandard(err)
// 	}
// 	return nil
// }
