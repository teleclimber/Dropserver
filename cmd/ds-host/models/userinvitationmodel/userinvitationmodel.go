package userinvitationmodel

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// UserInvitationModel represents the model for settings
type UserInvitationModel struct {
	DB *domain.DB

	stmt struct {
		getAll *sqlx.Stmt
		get    *sqlx.Stmt
		create *sqlx.Stmt
		delete *sqlx.Stmt
	}
}

// PrepareStatements prepares the statements
func (m *UserInvitationModel) PrepareStatements() {
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	// We are using the params table, which is also used by the db migration system to stash the current schema

	m.stmt.getAll = p.Prep(`SELECT * FROM user_invitations`)
	m.stmt.get = p.Prep(`SELECT * FROM user_invitations WHERE email = ?`)
	m.stmt.create = p.Prep(`INSERT INTO user_invitations (email) VALUES ( ? )`)
	m.stmt.delete = p.Prep(`DELETE FROM user_invitations WHERE email = ?`)

}

// GetAll returns all invitations
func (m *UserInvitationModel) GetAll() ([]domain.UserInvitation, error) {
	invites := make([]domain.UserInvitation, 0)

	err := m.stmt.getAll.Select(&invites)
	if err != nil {
		m.getLogger("GetAll()").Error(err)
		return nil, err
	}

	return invites, nil
}

// Get is used to know if an email is invited
func (m *UserInvitationModel) Get(email string) (domain.UserInvitation, error) {
	email = normalizeEmail(email)

	var invite domain.UserInvitation
	err := m.stmt.get.Get(&invite, email)
	if err != nil {
		if err != sql.ErrNoRows {
			m.getLogger("Get()").Error(err)
		}
		return invite, err
	}

	return invite, nil
}

// Create adds an invitiation to the table
func (m *UserInvitationModel) Create(email string) error {
	email = normalizeEmail(email)

	if len(email) < 3 || len(email) > 200 {
		msg := fmt.Sprintf("UserInvitationModel: email has unreasonable length: %d chars", len(email))
		m.getLogger("Create()").Log(msg)
		return errors.New("email has unreasonable length")
	}

	_, err := m.stmt.create.Exec(email)
	if err != nil {
		m.getLogger("Create()").Error(err)
		return err
	}

	return nil
}

// Delete removes an invitiation from the table
func (m *UserInvitationModel) Delete(email string) error {
	email = normalizeEmail(email)

	_, err := m.stmt.delete.Exec(email)
	if err != nil {
		m.getLogger("Delete").Error(err)
		return err
	}

	return nil
}

func (m *UserInvitationModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("UserInvitationModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

func normalizeEmail(email string) string {
	// may ned to trim whitespace too?
	return strings.ToLower(email)
}
