package remoteappspacemodel

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// AppspaceModel represents the model for app spaces
type RemoteAppspaceModel struct {
	DB *domain.DB

	stmt struct {
		selectDomain *sqlx.Stmt
		//selectOwner  *sqlx.Stmt	// later
		selectUser *sqlx.Stmt
		insert     *sqlx.Stmt
		delete     *sqlx.Stmt
	}
}

// PrepareStatements for appspace model
func (m *RemoteAppspaceModel) PrepareStatements() {
	// Here is the place to get clever with statements if using multiple DBs.
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	//get from domain
	m.stmt.selectDomain = p.Prep(`SELECT * FROM remote_appspaces WHERE user_id = ? AND domain_name = ?`)

	// get all for an owner
	m.stmt.selectUser = p.Prep(`SELECT * FROM remote_appspaces WHERE user_id = ?`)

	// insert appspace:
	m.stmt.insert = p.Prep(`INSERT INTO remote_appspaces
		(user_id, domain_name, owner_dropid, dropid, created) VALUES (?, ?, ?, ?, datetime("now"))`)

	// pause
	m.stmt.delete = p.Prep(`DELETE FROM remote_appspaces WHERE user_id = ? AND domain_name = ?`)
}

// Get returns the remote appspace that matches the domain
// it returns sql.ErrNoRows if none found
func (m *RemoteAppspaceModel) Get(userID domain.UserID, domainName string) (domain.RemoteAppspace, error) {
	// normalize domain name
	var remote domain.RemoteAppspace
	err := m.stmt.selectDomain.QueryRowx(userID, domainName).StructScan(&remote)
	if err != nil {
		if err != sql.ErrNoRows {
			m.getLogger("Get(), domain: " + domainName).Error(err)
		}
		return remote, err
	}

	return remote, nil
}

func (m *RemoteAppspaceModel) GetForUser(userID domain.UserID) ([]domain.RemoteAppspace, error) {
	ret := []domain.RemoteAppspace{}

	err := m.stmt.selectUser.Select(&ret, userID)
	if err != nil {
		m.getLogger("GetForUser()").UserID(userID).Error(err)
		return nil, err
	}

	return ret, nil
}

func (m *RemoteAppspaceModel) Create(userID domain.UserID, domainName string, ownerDropID string, dropID string) error {
	_, err := m.stmt.insert.Exec(userID, domainName, ownerDropID, dropID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return errors.New("appspace domain already exists for user")
		}
		m.getLogger("Create").Error(err)
		return err
	}
	return nil
}

func (m *RemoteAppspaceModel) Delete(userID domain.UserID, domainName string) error {
	_, err := m.stmt.delete.Exec(userID, domainName)
	if err != nil {
		m.getLogger("Delete").Error(err)
		return err
	}
	return nil
}

func (m *RemoteAppspaceModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("RemoteAppspaceModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
