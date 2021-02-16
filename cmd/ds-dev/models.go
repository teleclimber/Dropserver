package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

// DevAppModel can return a predetermined app and app version
type DevAppModel struct {
	App         domain.App
	Ver         domain.AppVersion
	AppspaceVer struct {
		Version domain.Version
		Schema  int
	}
	ToVer struct {
		Version domain.Version
		Schema  int
	}

	// need to expand on this.
	// - devVersion matches the code we are currently watching. Change to match when code (application.json, migrations) change
	// - appspaceVersion corresponds to what the appspace would be running on. Has a schema that matches appspace
	// - toVersion corresponds to the version of code that would run after a migration. Its schema matches the to schema
	// The only difference in these is the version string and schema int. So maybe leverage that.
	// alt may not be the right way to do this. Maybe need appspaceVersion struct{ Version, Schema }, toVersion struct{ Version, Schema }
}

// // Set the app and app version to return for all calls
// func (m *DevAppModel) Set(app domain.App, ver domain.AppVersion) {
// 	m.app = app
// 	m.ver = ver
// }

// // SetAppspaceVersion sets an alternative schema for a version that corresponds to the the code the appspace was last running
// func (m *DevAppModel) SetAppspaceVersion(version domain.Version, schema int) {
// 	m.appspaceVer.version = version
// 	m.appspaceVer.schema = schema
// }

// // SetToVersion sets an alternative schema for a version that corresponds to the the code the appspace will run at after a migration
// func (m *DevAppModel) SetToVersion(version domain.Version, schema int) {
// 	m.toVer.version = version
// 	m.toVer.schema = schema
// }

// GetFromID always returns the same app
func (m *DevAppModel) GetFromID(appID domain.AppID) (*domain.App, error) {
	return &m.App, nil
}

// GetVersion always returns the same version
func (m *DevAppModel) GetVersion(appID domain.AppID, version domain.Version) (*domain.AppVersion, error) {
	if version == m.Ver.Version {
		return &m.Ver, nil
	}

	ret := m.Ver
	ret.Version = version
	if version == m.AppspaceVer.Version {
		ret.Schema = m.AppspaceVer.Schema
		return &ret, nil
	}
	if version == m.ToVer.Version {
		ret.Schema = m.ToVer.Schema
		return &ret, nil
	}

	return nil, sql.ErrNoRows
}

// DevAppspaceModel can return an appspace struct as needed
type DevAppspaceModel struct {
	AsPausedEvent interface {
		Send(domain.AppspaceID, bool)
	}
	Appspace domain.Appspace
}

// Pause pauses the appspace
func (m *DevAppspaceModel) Pause(appspaceID domain.AppspaceID, pause bool) error {
	m.Appspace.Paused = pause // wait does that work? we're not dealing with pointers here.

	m.AsPausedEvent.Send(appspaceID, pause)

	return nil
}

// GetFromSubdomain always returns the same appspace
func (m *DevAppspaceModel) GetFromSubdomain(subdomain string) (*domain.Appspace, error) {
	return &m.Appspace, nil
}

// GetFromID always returns the same appspace
func (m *DevAppspaceModel) GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, error) {
	return &m.Appspace, nil
}

// SetVersion changes the active version of the application for tha tappspace
func (m *DevAppspaceModel) SetVersion(appspaceID domain.AppspaceID, version domain.Version) error {
	//m.Appspace.AppVersion = version // will bomb if no appspace is set
	// no-op! this gets called after a migration, but in ds-dev we don't want it to have an effect.
	return nil
}

//
type DevAppspaceContactModel struct {
	hasOwnerProxy bool
	ownerProxy    domain.ProxyID
}

// SetOwnerProxyID is Dev-only
func (m *DevAppspaceContactModel) SetOwnerProxyID(proxyID domain.ProxyID) {
	m.hasOwnerProxy = true
	m.ownerProxy = proxyID
}
func (m *DevAppspaceContactModel) GetOwnerProxyID() domain.ProxyID {
	if m.hasOwnerProxy {
		return m.ownerProxy
	}
	return domain.ProxyID("")
}

// Proxy is owner returns true if proxy_id is the appspace owner
func (m *DevAppspaceContactModel) ProxyIsOwner(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (bool, error) {
	return m.hasOwnerProxy && m.ownerProxy == proxyID, nil
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
	MigrationJobEvents interface {
		Send(domain.MigrationJob)
	}

	nextJobID int
	job       *domain.MigrationJob
}

// Create a migration job
func (m *DevMigrationJobModel) Create(ownerID domain.UserID, appspaceID domain.AppspaceID, toVersion domain.Version, priority bool) (*domain.MigrationJob, error) {
	if m.job != nil && m.job.Started.Valid && !m.job.Finished.Valid {
		return nil, errors.New("DevMigrationJobModel can't create job while one is in use")
	}
	m.nextJobID++
	m.job = &domain.MigrationJob{
		JobID:      domain.JobID(m.nextJobID),
		AppspaceID: appspaceID,
		Created:    time.Now(),
		OwnerID:    ownerID,
		ToVersion:  toVersion,
		Priority:   priority,
	}
	go m.sendJobAsEvent()
	return nil, nil
}

func (m *DevMigrationJobModel) GetJob(jobID domain.JobID) (*domain.MigrationJob, error) {
	// unused I think
	return m.job, nil
}

func (m *DevMigrationJobModel) GetPending() ([]*domain.MigrationJob, error) {
	if m.job != nil {
		ret := make([]*domain.MigrationJob, 1)
		ret[0] = m.job
		return ret, nil
	}
	return nil, nil
}

func (m *DevMigrationJobModel) GetRunning() ([]domain.MigrationJob, error) {
	ret := []domain.MigrationJob{}
	if m.job != nil && m.job.Started.Valid && !m.job.Finished.Valid {
		ret = append(ret, *m.job)
	}
	return ret, nil
}

func (m *DevMigrationJobModel) SetStarted(jobID domain.JobID) (bool, error) {
	if m.job != nil {
		if m.job.Started.Valid {
			return false, nil
		}
		m.job.Started = nulltypes.NewTime(time.Now(), true)
		go m.sendJobAsEvent()
		return true, nil
	}
	return false, nil
}

func (m *DevMigrationJobModel) SetFinished(jobID domain.JobID, errStr nulltypes.NullString) error {
	if m.job != nil {
		if m.job.Finished.Valid {
			return errors.New("already finished")
		}
		m.job.Finished = nulltypes.NewTime(time.Now(), true)
		m.job.Error = errStr
		fmt.Println("set job to finished and sending")
		go m.sendJobAsEvent()
		return nil
	}
	return errors.New("no job??")
}

func (m *DevMigrationJobModel) sendJobAsEvent() {
	m.MigrationJobEvents.Send(*m.job)
}
