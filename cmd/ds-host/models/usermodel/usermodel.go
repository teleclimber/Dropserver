package usermodel

import (
	"errors"
	"fmt"
	"strings"
	"golang.org/x/crypto/bcrypt"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// UserModel represents the model for app
type UserModel struct {
	DB *domain.DB
	// need config to select db type?
	Logger domain.LogCLientI

	stmt struct {
		selectID	*sqlx.Stmt
		selectEmail *sqlx.Stmt
		insertUser	*sqlx.Stmt
		updatePassword *sqlx.Stmt
		getPassword *sqlx.Stmt
		selectAdmin	*sqlx.Stmt
		insertAdmin	*sqlx.Stmt
		deleteAdmin	*sqlx.Stmt
	}
}

// going to try something better for prepare statements:
type prepper struct {
	handle *sqlx.DB
	err error
}
func  (p *prepper) exec(query string) *sqlx.Stmt {
	if p.err != nil {
		return nil
	}

	stmt, err := p.handle.Preparex(query)
	if err != nil {
		p.err = errors.New("Error preparing statmement "+query+" "+err.Error())
		return nil
	}

	return stmt
}

// PrepareStatements prepares the statements
func (m *UserModel) PrepareStatements() {
	p := prepper{ handle: m.DB.Handle }

	m.stmt.selectID = p.exec(`SELECT user_id, email FROM users WHERE user_id = ?`)

	m.stmt.selectEmail = p.exec(`SELECT user_id, email FROM users WHERE email = ?`)

	m.stmt.insertUser = p.exec(`INSERT INTO users 
		("email", "password") VALUES (?, ?)`)

	m.stmt.updatePassword = p.exec(`UPDATE users SET password = ? WHERE user_id = ?`)
	
	m.stmt.getPassword = p.exec(`SELECT password FROM users WHERE user_id = ?`)

	m.stmt.selectAdmin = p.exec(`SELECT EXISTS(SELECT 1 FROM admin_users WHERE user_id = ?)`)
	m.stmt.insertAdmin = p.exec(`INSERT INTO admin_users (user_id) VALUES (?)`)
	m.stmt.deleteAdmin = p.exec(`DELETE FROM admin_users WHERE user_id = ?`)

	if p.err != nil {
		panic(p.err)
	}
}

// Create creates a new user
func (m *UserModel) Create(email, password string) (*domain.User, domain.Error) { //return User
	// Here we have a minimal check for definitely bad inputs
	// like blank or nearly blank emails and passwords.
	// with the understanding that these should be checked before calling this method
	if len(email) < 4 || len(email) > 200 {
		msg := fmt.Sprintf("User Model: email has unreasonable length: %d chars",len(email))
		m.Logger.Log(domain.WARN, nil, msg)
		return nil, dserror.New(dserror.InternalError, msg)
	}

	if dsErr := m.validatePassword(password); dsErr != nil {
		m.Logger.Log(domain.WARN, nil, dsErr.ExtraMessage())
		return nil, dsErr
	}
	
	hash, dsErr := m.hashPassword(password)
	if dsErr != nil {
		return nil, dsErr
	}

	email = strings.ToLower(email)

	r, err := m.stmt.insertUser.Exec(email, hash)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return nil, dserror.New(dserror.EmailExists)
		}
		m.Logger.Log(domain.ERROR, nil, "User Model Insert User error: "+err.Error())
		return nil, dserror.FromStandard(err)
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "User Model lastID error:"+err.Error())
		return nil, dserror.FromStandard(err)
	}
	if lastID >= 0xFFFFFFFF {
		m.Logger.Log(domain.ERROR, nil, "User Model lastID out of bounds error")
		return nil, dserror.New(dserror.OutOFBounds, "Last Insert ID from DB greater than uint32")
	}

	userID := domain.UserID(lastID)

	user, dsErr := m.GetFromID(userID)
	if dsErr != nil {
		return nil, dsErr
	}

	return user, nil
}

// UpdatePassword updates the password for the user.
func (m *UserModel) UpdatePassword(userID domain.UserID, password string) domain.Error {
	hash, dsErr := m.hashPassword(password)
	if dsErr != nil {
		return dsErr
	}

	_, err := m.stmt.updatePassword.Exec(hash, userID)
	if err != nil {
		return dserror.FromStandard(err)
	}

	return nil
}

func (m *UserModel) validatePassword(password string) domain.Error {
	if len(password) < 8 {
		return dserror.New(dserror.InternalError, "password less than 8 chars in User Model Create")
		//internal error because this shouldn't have made it this far, correct?
	}
	return nil
}

func (m *UserModel) hashPassword(password string) ([]byte, domain.Error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		// log here too because this is not a good error
		m.Logger.Log(domain.ERROR, nil, "User Model: error generating bcrypt: "+err.Error())
		return nil, dserror.FromStandard(err)
	}
	return hash, nil
}

// GetFromID returns a user
func (m *UserModel) GetFromID(userID domain.UserID) (*domain.User, domain.Error) {
	var user domain.User

	err := m.stmt.selectID.QueryRowx(userID).StructScan(&user)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, dserror.New(dserror.NoRowsInResultSet)
		}
		m.Logger.Log(domain.ERROR, nil, "User Model, db error, GetFromID: "+err.Error())
		return nil, dserror.FromStandard(err)
	}
	// here we should differentiate between no rows returned and every other error?
	// Although if you're querying with an ID and you don't find it, that's pretty internal an error?

	return &user, nil
}

// GetFromEmail returns a user
func (m *UserModel) GetFromEmail(email string) (*domain.User, domain.Error) {
	var user domain.User

	err := m.stmt.selectEmail.QueryRowx(strings.ToLower(email)).StructScan(&user)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, dserror.New(dserror.NoRowsInResultSet)
		}
		m.Logger.Log(domain.ERROR, nil, "User Model, db error, GetFromEmail: "+err.Error())
		return nil, dserror.FromStandard(err)
	}

	return &user, nil
}

// GetFromEmailPassword is the proverbial authentication function
func (m *UserModel) GetFromEmailPassword(email, password string) (*domain.User, domain.Error) {

	user, dsErr := m.GetFromEmail(email)
	if dsErr != nil {
		return nil, dsErr
	}
	
	var hash []byte
	err := m.stmt.getPassword.Get(&hash, user.UserID)
	if err != nil {
		// this is likely internal error since we know the user exists
		m.Logger.Log(domain.ERROR, nil, "User Model, db error, GetPassword: "+err.Error())
		return nil, dserror.FromStandard(err)
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		return nil, dserror.New(dserror.AuthenticationIncorrect)
	}

	return user, nil
}

// IsAdmin tells you if the user is an admin on DropServer
func (m *UserModel) IsAdmin(userID domain.UserID) bool {
	var exists int
	err := m.stmt.selectAdmin.Get(&exists, userID)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "User Model, db error, admin exists: "+err.Error())
		return false
	}

	if exists == 1 {
		return true
	}
	return false
}

// MakeAdmin adds the user_id to the table of DropServer admin users
func (m *UserModel) MakeAdmin(userID domain.UserID) domain.Error {
	_, err := m.stmt.insertAdmin.Exec(userID)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: admin_users.user_id" {
			return nil
		}
		msg := "User Model, make admin: db error "+err.Error()
		m.Logger.Log(domain.ERROR, nil, msg)
		return dserror.New(dserror.InternalError, msg)
	}
	return nil
}

// DeleteAdmin removes the user id from the tableof admin users for this server.
func (m *UserModel) DeleteAdmin(userID domain.UserID) domain.Error {
	_, err := m.stmt.deleteAdmin.Exec(userID)
	if err != nil {
		msg := "User Model, delete admin: db error "+err.Error()
		m.Logger.Log(domain.ERROR, nil, msg)
		return dserror.New(dserror.InternalError, msg)
	}

	return nil
}

