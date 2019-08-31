package userinvitationmodel

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// UserInvitationModel represents the model for settings
type UserInvitationModel struct {
	DB *domain.DB

	Logger domain.LogCLientI

	stmt struct {
		getAll *sqlx.Stmt
		get    *sqlx.Stmt
		create *sqlx.Stmt
		delete *sqlx.Stmt
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
func (m *UserInvitationModel) PrepareStatements() {
	p := prepper{handle: m.DB.Handle}

	// We are using the params table, which is also used by the db migration system to stash the current schema

	m.stmt.getAll = p.exec(`SELECT * FROM user_invitations`)
	m.stmt.get = p.exec(`SELECT * FROM user_invitations WHERE email = ?`)
	m.stmt.create = p.exec(`INSERT INTO user_invitations (email) VALUES ( ? )`)
	m.stmt.delete = p.exec(`DELETE FROM user_invitations WHERE email = ?`)

	if p.err != nil {
		panic(p.err)
	}
}

// GetAll returns all invitations
func (m *UserInvitationModel) GetAll() ([]*domain.UserInvitation, domain.Error) {
	var invites []*domain.UserInvitation

	err := m.stmt.getAll.Select(&invites)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Settings Model, db error, Get: "+err.Error())
		return nil, dserror.FromStandard(err)
	}

	return invites, nil
}

// Get is used to know if an email is invited
func (m *UserInvitationModel) Get(email string) (*domain.UserInvitation, domain.Error) {
	email = normalizeEmail(email)
	
	var invite domain.UserInvitation
	err := m.stmt.get.Get(&invite, email)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, dserror.New(dserror.NoRowsInResultSet)
		}
		m.Logger.Log(domain.ERROR, nil, "Settings Model, db error, Get: "+err.Error())
		return nil, dserror.FromStandard(err)
	}

	return &invite, nil
}

// Create adds an invitiation to the table
func (m *UserInvitationModel) Create(email string) domain.Error {
	email = normalizeEmail(email)

	if len(email) < 4 || len(email) > 200 {
		msg := fmt.Sprintf("UserInvitationModel: email has unreasonable length: %d chars",len(email))
		m.Logger.Log(domain.WARN, nil, msg)
		return dserror.New(dserror.InternalError, msg)
	}

	// should we not normalize emails?
	// I think we do this in 

	_, err := m.stmt.create.Exec(email)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "UserInvitationModel Create error: "+err.Error())
		return dserror.FromStandard(err)
	}

	return nil
}

// Delete removes an invitiation from the table
func (m *UserInvitationModel) Delete(email string) domain.Error {
	email = normalizeEmail(email)

	_, err := m.stmt.delete.Exec(email)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "UserInvitationModel Delete error: "+err.Error())
		return dserror.FromStandard(err)
	}

	return nil
}

func normalizeEmail(email string) string {
	// may ned to trim whitespace too?
	return strings.ToLower(email)
}


