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
		getAll          *sqlx.Stmt // not sure about this one...
		set             *sqlx.Stmt
		setReg          *sqlx.Stmt
		getTSNet        *sqlx.Stmt
		setTSNet        *sqlx.Stmt
		setTSNetConnect *sqlx.Stmt
	}
}

// PrepareStatements prepares the statements
func (m *SettingsModel) PrepareStatements() {
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	// We are using the params table, which is also used by the db migration system to stash the current schema

	m.stmt.getAll = p.Prep(`SELECT registration_open FROM settings WHERE id = 1`)
	m.stmt.setReg = p.Prep(`UPDATE settings SET registration_open = ? WHERE id = 1`)

	m.stmt.getTSNet = p.Prep(`SELECT tsnet_control_url, tsnet_hostname, tsnet_connect FROM settings WHERE id = 1`)
	m.stmt.setTSNet = p.Prep(`UPDATE settings SET tsnet_control_url = ?, tsnet_hostname = ?, tsnet_connect = ? WHERE id = 1`)
	m.stmt.setTSNetConnect = p.Prep(`UPDATE settings SET tsnet_connect = ? WHERE id = 1`)
}

// Get returns the settings except for tsnet
func (m *SettingsModel) Get() (domain.Settings, error) { //rename to GetReg? or make it return everything? Unclear.
	var settings domain.Settings

	err := m.stmt.getAll.QueryRowx().StructScan(&settings)
	if err != nil {
		m.getLogger("Get()").Error(err)
		return settings, err
	}

	return settings, nil
}

// Set sets the registration open column in settings row
// func (m *SettingsModel) Set(settings domain.Settings) error { // Eliminate this in favor of granular sets
// 	_, err := m.stmt.setReg.Exec(settings.RegistrationOpen)
// 	if err != nil {
// 		m.getLogger("Set()").Error(err)
// 		return err
// 	}

// 	return nil
// }

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

// TSNetConfig is domain.TSNetCommon
// but with different db struct tags
type TSNetConfig struct {
	ControlURL string `db:"tsnet_control_url"`
	Hostname   string `db:"tsnet_hostname"`
	Connect    bool   `db:"tsnet_connect"`
}

func (m *SettingsModel) GetTSNet() (domain.TSNetCommon, error) {
	var settings TSNetConfig
	err := m.stmt.getTSNet.QueryRowx().StructScan(&settings)
	if err != nil {
		m.getLogger("GetTSNet()").Error(err)
		return domain.TSNetCommon{}, err
	}
	return domain.TSNetCommon{
		ControlURL: settings.ControlURL,
		Hostname:   settings.Hostname,
		Connect:    settings.Connect,
	}, nil
}

func (m *SettingsModel) SetTSNet(config domain.TSNetCommon) error {
	_, err := m.stmt.setTSNet.Exec(config.ControlURL, config.Hostname, config.Connect)
	if err != nil {
		m.getLogger("SetTSNet()").Error(err)
		return err
	}
	return nil
}

func (m *SettingsModel) SetTSNetConnect(connect bool) error {
	_, err := m.stmt.setTSNetConnect.Exec(connect)
	if err != nil {
		m.getLogger("SetTSNetConnect()").Error(err)
		return err
	}
	return nil
}

func (m *SettingsModel) DeleteTSNet() error {
	_, err := m.stmt.setTSNet.Exec("", "", false)
	if err != nil {
		m.getLogger("DeleteTSNet()").Error(err)
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
