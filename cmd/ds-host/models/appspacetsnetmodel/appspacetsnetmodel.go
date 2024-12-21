package appspacetsnetmodel

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// AppspaceTSNetModel represents the model for appspace tsnet data
type AppspaceTSNetModel struct {
	DB *domain.DB

	AppspaceTSNetModelEvents interface {
		Send(domain.AppspaceTSNetModelEvent)
	} `checkinject:"required"`

	stmt struct {
		upsert         *sqlx.Stmt
		delete         *sqlx.Stmt
		getAppspaceID  *sqlx.Stmt
		selectConnects *sqlx.Stmt
		setConnect     *sqlx.Stmt
	}
}

func (m *AppspaceTSNetModel) PrepareStatements() {
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	m.stmt.getAppspaceID = p.Prep(`SELECT * FROM appspace_tsnet WHERE appspace_id = ?`)

	m.stmt.selectConnects = p.Prep(`SELECT * FROM appspace_tsnet WHERE connect = true`)

	m.stmt.upsert = p.Prep(`INSERT INTO appspace_tsnet
		("appspace_id", "backend_url", "hostname", "connect") VALUES (?, ?, ?, ?)
		ON CONFLICT(appspace_id) DO UPDATE
		SET backend_url = ?, hostname = ?, connect = ?`)

	m.stmt.setConnect = p.Prep(`UPDATE appspace_tsnet SET connect = ? WHERE appspace_id = ?`)

	// set hostname?

	m.stmt.delete = p.Prep(`DELETE FROM appspace_tsnet WHERE appspace_id = ?`)
}

func (m *AppspaceTSNetModel) Get(appspaceID domain.AppspaceID) (domain.AppspaceTSNet, error) {
	var ret domain.AppspaceTSNet

	err := m.stmt.getAppspaceID.Get(&ret, appspaceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ret, domain.ErrNoRowsInResultSet
		}
		m.getLogger("Get").AppspaceID(appspaceID).Error(err)
	}
	return ret, err
}

// GetAllConnect returns data for the appspaces that want a tsnet server
func (m *AppspaceTSNetModel) GetAllConnect() (tsnets []domain.AppspaceTSNet, err error) {
	err = m.stmt.selectConnects.Select(&tsnets)
	if err != nil {
		m.getLogger("GetAllConnect()").Error(err)
	}
	return
}

func (m *AppspaceTSNetModel) CreateOrUpdate(appspaceID domain.AppspaceID, backendURL string, hostname string, connect bool) error {
	_, err := m.stmt.upsert.Exec(appspaceID, backendURL, hostname, connect, backendURL, hostname, connect)
	if err != nil {
		m.getLogger("CreateTSNet() upsert").AppspaceID(appspaceID).Error(err)
	}
	m.sendModifiedEvent(appspaceID)
	return err
}

func (m *AppspaceTSNetModel) SetConnect(appspaceID domain.AppspaceID, connect bool) error {
	_, err := m.stmt.setConnect.Exec(connect, appspaceID)
	if err != nil {
		m.getLogger("SetConnect() update").AppspaceID(appspaceID).Error(err)
	}
	m.sendModifiedEvent(appspaceID)
	return err
}

func (m *AppspaceTSNetModel) Delete(appspaceID domain.AppspaceID) error {
	_, err := m.stmt.delete.Exec(appspaceID)
	if err != nil {
		m.getLogger("Delete() delete").AppspaceID(appspaceID).Error(err)
	}

	m.AppspaceTSNetModelEvents.Send(domain.AppspaceTSNetModelEvent{
		Deleted: true,
		AppspaceTSNet: domain.AppspaceTSNet{
			AppspaceID: appspaceID,
		},
	})

	return err
}

func (m *AppspaceTSNetModel) sendModifiedEvent(appspaceID domain.AppspaceID) {
	data, err := m.Get(appspaceID)
	if err != nil {
		return
	}
	m.AppspaceTSNetModelEvents.Send(domain.AppspaceTSNetModelEvent{
		Deleted: false,
		AppspaceTSNet: domain.AppspaceTSNet{
			AppspaceID: appspaceID,
			BackendURL: data.BackendURL,
			Hostname:   data.Hostname,
			Connect:    data.Connect,
		},
	})
}

func (m *AppspaceTSNetModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceTSNetModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
