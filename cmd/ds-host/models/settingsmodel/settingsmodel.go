package settingsmodel

import (
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/dserror"
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

type prepper struct {
	handle *sqlx.DB
	err    error
}

func (p *prepper) exec(query string) *sqlx.Stmt {
	if p.err != nil {
		return nil
	}

	stmt, err := p.handle.Preparex(query)
	if err != nil {
		p.err = errors.New("Error preparing statmement " + query + " " + err.Error())
		return nil
	}

	return stmt
}

// PrepareStatements prepares the statements
func (m *SettingsModel) PrepareStatements() {
	p := prepper{handle: m.DB.Handle}

	// We are using the params table, which is also used by the db migration system to stash the current schema

	m.stmt.getAll = p.exec(`SELECT registration_open FROM settings WHERE id = 1`)
	m.stmt.set = p.exec(`UPDATE settings SET registration_open = ? WHERE id = 1`)
	m.stmt.setReg = p.exec(`UPDATE settings SET registration_open = ? WHERE id = 1`)

	if p.err != nil {
		panic(p.err)
	}
}

// Get returns the value for a specific key
func (m *SettingsModel) Get() (*domain.Settings, domain.Error) {
	var settings domain.Settings

	err := m.stmt.getAll.QueryRowx().StructScan(&settings)
	if err != nil {
		m.getLogger("Get()").Error(err)
		return nil, dserror.FromStandard(err)
	}

	return &settings, nil
}

// Set sets the registration open column in settings row
func (m *SettingsModel) Set(settings *domain.Settings) domain.Error {
	_, err := m.stmt.setReg.Exec(settings.RegistrationOpen)
	if err != nil {
		m.getLogger("Set()").Error(err)
		return dserror.FromStandard(err)
	}

	return nil
}

// SetRegistrationOpen sets the registration open column in settings row
func (m *SettingsModel) SetRegistrationOpen(open bool) domain.Error {
	_, err := m.stmt.setReg.Exec(open)
	if err != nil {
		m.getLogger("SetRegistrationOpen()").Error(err)
		return dserror.FromStandard(err)
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
