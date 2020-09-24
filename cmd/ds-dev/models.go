package main

import (
	"errors"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

// DevAppModel can return a predetermined app and app version
type DevAppModel struct {
	app domain.App
	ver domain.AppVersion
}

// Set the app and app version to return for all calls
func (m *DevAppModel) Set(app domain.App, ver domain.AppVersion) {
	m.app = app
	m.ver = ver
}

// GetFromID always returns the same app
func (m *DevAppModel) GetFromID(appID domain.AppID) (*domain.App, domain.Error) {
	return &m.app, nil
}

// GetVersion always returns the same version
func (m *DevAppModel) GetVersion(appID domain.AppID, version domain.Version) (*domain.AppVersion, domain.Error) {
	return &m.ver, nil
}

// DevAppspaceModel can return an appspace struct as needed
type DevAppspaceModel struct {
	AsPausedEvent interface {
		Send(domain.AppspaceID, bool)
	}
	appspace domain.Appspace
}

// Pause pauses the appspace
func (m *DevAppspaceModel) Pause(appspaceID domain.AppspaceID, pause bool) domain.Error {
	m.appspace.Paused = pause // wait does that work? we're not dealing with pointers here.

	m.AsPausedEvent.Send(appspaceID, pause)

	return nil
}

// Set the appspace to return for all calls
func (m *DevAppspaceModel) Set(appspace domain.Appspace) {
	m.appspace = appspace
}

// GetFromSubdomain always returns the same appspace
func (m *DevAppspaceModel) GetFromSubdomain(subdomain string) (*domain.Appspace, domain.Error) {
	return &m.appspace, nil
}

// GetFromID always returns the same appspace
func (m *DevAppspaceModel) GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, domain.Error) {
	return &m.appspace, nil
}

// SetVersion changes the active version of the application for tha tappspace
func (m *DevAppspaceModel) SetVersion(appspaceID domain.AppspaceID, version domain.Version) domain.Error {
	m.appspace.AppVersion = version // will bomb if no appspace is set
	return nil
}

////////
// MigrationJobModel handles writing jobs to the db
// type MigrationJobModel interface {
// 	Create(UserID, AppspaceID, Version, bool) (*MigrationJob, Error)
// 	GetJob(JobID) (*MigrationJob, Error)
// 	GetPending() ([]*MigrationJob, Error)
// 	SetStarted(JobID) (bool, Error)
// 	SetFinished(JobID, nulltypes.NullString) Error
// 	//GetForAppspace(AppspaceID) (*MigrationJob, Error)
// 	// Delete(AppspaceID) Error
// }

// DevMigrationJobModel tracks a single migration job at a time
type DevMigrationJobModel struct {
	job *domain.MigrationJob
}

// Create a migration job
func (m *DevMigrationJobModel) Create(ownerID domain.UserID, appspaceID domain.AppspaceID, toVersion domain.Version, priority bool) (*domain.MigrationJob, error) {
	if m.job.Started.Valid && !m.job.Finished.Valid {
		return nil, errors.New("DevMigrationJobModel can't create job while one is in use")
	}
	//m.jobs = append(m.jobs)
	return nil, nil
}

func (m *DevMigrationJobModel) GetJob(jobID domain.JobID) (*domain.MigrationJob, error) {
	// unused I think
	return nil, nil
}

func (m *DevMigrationJobModel) GetPending() ([]*domain.MigrationJob, error) {
	return nil, nil
}

func (m *DevMigrationJobModel) SetStarted(jobID domain.JobID) (bool, error) {
	return false, nil
}

func (m *DevMigrationJobModel) SetFinished(jobID domain.JobID, errStr nulltypes.NullString) error {
	return nil
}
