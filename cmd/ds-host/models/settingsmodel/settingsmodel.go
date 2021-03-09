package settingsmodel

import (
	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// SettingsModel represents the model for settings
type SettingsModel struct {
	DB *domain.DB

	stmt struct {
		getAll *sqlx.Stmt
		set    *sqlx.Stmt
		setReg *sqlx.Stmt
	}
}

// PrepareStatements prepares the statements
func (m *SettingsModel) PrepareStatements() {
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	// We are using the params table, which is also used by the db migration system to stash the current schema

	m.stmt.getAll = p.Prep(`SELECT registration_open FROM settings WHERE id = 1`)
	m.stmt.set = p.Prep(`UPDATE settings SET registration_open = ? WHERE id = 1`)
	m.stmt.setReg = p.Prep(`UPDATE settings SET registration_open = ? WHERE id = 1`)
}

// Get returns the value for a specific key
func (m *SettingsModel) Get() (domain.Settings, error) {
	var settings domain.Settings

	err := m.stmt.getAll.QueryRowx().StructScan(&settings)
	if err != nil {
		m.getLogger("Get()").Error(err)
		return settings, err
	}

	return settings, nil
}

// Set sets the registration open column in settings row
func (m *SettingsModel) Set(settings domain.Settings) error {
	_, err := m.stmt.setReg.Exec(settings.RegistrationOpen)
	if err != nil {
		m.getLogger("Set()").Error(err)
		return err
	}

	return nil
}

// wait why do we need both ^v??

// SetRegistrationOpen sets the registration open column in settings row
func (m *SettingsModel) SetRegistrationOpen(open bool) error {
	_, err := m.stmt.setReg.Exec(open)
	if err != nil {
		m.getLogger("SetRegistrationOpen()").Error(err)
		return err
	}

	return nil
}

func (m *SettingsModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("SettingsModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
