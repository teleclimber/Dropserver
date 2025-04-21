package usermodel

import (
	"database/sql"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/nulltypes"
	"github.com/teleclimber/DropServer/internal/validator"
)

// UserModel represents the model for app
type UserModel struct {
	DB *domain.DB
	// need config to select db type?

	stmt struct {
		selectID        *sqlx.Stmt
		selectEmail     *sqlx.Stmt
		selectAll       *sqlx.Stmt
		insertUser      *sqlx.Stmt
		updateEmail     *sqlx.Stmt
		updatePassword  *sqlx.Stmt
		updateTSNet     *sqlx.Stmt
		selectTSNet     *sqlx.Stmt
		getPassword     *sqlx.Stmt
		selectAdmin     *sqlx.Stmt
		selectAllAdmins *sqlx.Stmt
		insertAdmin     *sqlx.Stmt
		deleteAdmin     *sqlx.Stmt
	}
}

// going to try something better for prepare statements:
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
func (m *UserModel) PrepareStatements() {
	p := prepper{handle: m.DB.Handle}

	m.stmt.selectID = p.exec(`SELECT user_id, email, password, tsnet_identifier, tsnet_extra_name FROM users WHERE user_id = ?`)

	m.stmt.selectEmail = p.exec(`SELECT user_id, email, password, tsnet_identifier, tsnet_extra_name FROM users WHERE email = ?`)

	m.stmt.selectAll = p.exec(`SELECT user_id, email, password, tsnet_identifier, tsnet_extra_name FROM users`)

	m.stmt.insertUser = p.exec(`INSERT INTO users 
		("email", "password", "tsnet_identifier", "tsnet_extra_name") VALUES (?, ?, ?, ?)`)

	m.stmt.updateEmail = p.exec(`UPDATE users SET email = ? WHERE user_id = ?`)
	m.stmt.updatePassword = p.exec(`UPDATE users SET password = ? WHERE user_id = ?`)
	m.stmt.getPassword = p.exec(`SELECT password FROM users WHERE user_id = ?`)

	m.stmt.updateTSNet = p.exec(`UPDATE users SET tsnet_identifier = ?, tsnet_extra_name = ? WHERE user_id = ?`)
	m.stmt.selectTSNet = p.exec(`SELECT user_id, email, password, tsnet_identifier, tsnet_extra_name FROM users WHERE tsnet_identifier = ?`)

	m.stmt.selectAdmin = p.exec(`SELECT EXISTS(SELECT 1 FROM admin_users WHERE user_id = ?)`)
	m.stmt.selectAllAdmins = p.exec(`SELECT * FROM admin_users`)
	m.stmt.insertAdmin = p.exec(`INSERT INTO admin_users (user_id) VALUES (?)`)
	m.stmt.deleteAdmin = p.exec(`DELETE FROM admin_users WHERE user_id = ?`)

	if p.err != nil {
		panic(p.err)
	}
}

// Create creates a new user
// rename to CreateWithPassword
func (m *UserModel) CreateWithEmail(email, password string) (domain.User, error) {
	var user domain.User

	// Here we have a minimal check for definitely bad inputs
	// like blank or nearly blank emails and passwords.
	// with the understanding that these should be checked before calling this method
	if len(email) < 4 || len(email) > 200 {
		return user, errors.New("email invalid length")
	}

	if err := validatePassword(password); err != nil {
		return user, err
	}

	hash, err := m.hashPassword(password)
	if err != nil {
		return user, err
	}

	email = validator.NormalizeEmail(email)

	return m.insert(nulltypes.NewString(email, true), hash, nulltypes.NewString("", false), nulltypes.NewString("", false))
}

func (m *UserModel) CreateWithTSNet(tsnetIdentifier string, tsnetExtraName string) (domain.User, error) {
	// TODO validate identifer and extra nmae to some extenst??
	// nb: be careful of normalization: with email we normlalize to lower case. Here, may not be a good idea.

	return m.insert(nulltypes.NewString("", false), nil, nulltypes.NewString(tsnetIdentifier, true), nulltypes.NewString(tsnetExtraName, true))
}

func (m *UserModel) insert(email nulltypes.NullString, pw_hash []byte, tsnet_id, tsnet_extra nulltypes.NullString) (domain.User, error) {
	var user domain.User

	r, err := m.stmt.insertUser.Exec(email, pw_hash, tsnet_id, tsnet_extra)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return user, domain.ErrIdentifierExists
		}
		if err.Error() == "UNIQUE constraint failed: users.tsnet_identifier" {
			return user, domain.ErrIdentifierExists
		}
		m.getLogger("Create(), insertUser").Error(err)
		return user, err
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		m.getLogger("Create() lastID").Error(err)
		return user, err
	}
	if lastID >= 0xFFFFFFFF {
		m.getLogger("Create()").Log("lastID out of bounds")
		return user, err
	}

	userID := domain.UserID(lastID)

	user, err = m.GetFromID(userID) // urgh this should in a transaction
	if err != nil {
		// maybe log that we failed to get the user we just created?
		return user, err
	}

	return user, nil
}

func (m *UserModel) UpdateEmail(userID domain.UserID, email string) error {
	email = validator.NormalizeEmail(email)

	_, err := m.stmt.updateEmail.Exec(email, userID)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return domain.ErrIdentifierExists
		}
		m.getLogger("UpdateEmail() Exec()").Error(err)
		return err
	}
	return nil
}

// UpdatePassword updates the password for the user.
func (m *UserModel) UpdatePassword(userID domain.UserID, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}

	hash, err := m.hashPassword(password)
	if err != nil {
		return err
	}

	_, err = m.stmt.updatePassword.Exec(hash, userID)
	if err != nil {
		m.getLogger("UpdatePassword() Exec()").Error(err)
		return err
	}

	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password less than 8 chars in User Model Create")
	}
	return nil
}

func (m *UserModel) hashPassword(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		// log here too because this is not a good error
		m.getLogger("hashPassword()").Error(err)
		return nil, err
	}
	return hash, nil
}

func (m *UserModel) UpdateTSNet(userID domain.UserID, tsnetIdentifier string, tsnetExtraName string) error {
	// validate tsnet
	// they can not be empty. If you want to empty, use delete.

	_, err := m.stmt.updateTSNet.Exec(tsnetIdentifier, tsnetExtraName, userID)
	if err != nil {
		m.getLogger("UpdateTSNet() Exec").Error(err)
	}
	return err
}

func (m *UserModel) DeleteTSNet(userID domain.UserID) error {
	_, err := m.stmt.updateTSNet.Exec(nil, nil, userID)
	if err != nil {
		m.getLogger("DeleteTSNet() Exec").Error(err)
	}
	return err
}

type dbUser struct {
	UserID     uint32               `db:"user_id"`
	Email      nulltypes.NullString `db:"email"`
	Password   nulltypes.NullString `db:"password"`
	TSNetID    nulltypes.NullString `db:"tsnet_identifier"`
	TSNetExtra nulltypes.NullString `db:"tsnet_extra_name"`
}

// GetFromID returns a user
// it retunrs sql.ErrNoRows if not found
func (m *UserModel) GetFromID(userID domain.UserID) (domain.User, error) {
	var dbUser dbUser

	err := m.stmt.selectID.QueryRowx(userID).StructScan(&dbUser) // need a different struct!
	if err != nil {
		if err != sql.ErrNoRows {
			m.getLogger("GetFromID()").Error(err)
		}
		return domain.User{}, err
	}
	// here we should differentiate between no rows returned and every other error?
	// Although if you're querying with an ID and you don't find it, that's pretty internal an error?

	return userFromDB(dbUser), nil
}

// GetFromEmail returns a user
// it retunrs sql.ErrNoRows if not found
func (m *UserModel) GetFromEmail(email string) (domain.User, error) {
	var dbUser dbUser

	err := m.stmt.selectEmail.QueryRowx(strings.ToLower(email)).StructScan(&dbUser)
	if err != nil {
		if err != sql.ErrNoRows {
			m.getLogger("GetFromEmail()").Error(err)
		}
		return domain.User{}, err
	}

	return userFromDB(dbUser), nil
}

// GetFromEmailPassword is the proverbial authentication function
func (m *UserModel) GetFromEmailPassword(email, password string) (domain.User, error) {
	user, err := m.GetFromEmail(email)
	if err != nil {
		return user, err
	}

	var hash []byte
	err = m.stmt.getPassword.Get(&hash, user.UserID)
	if err != nil {
		// this is likely internal error since we know the user exists
		m.getLogger("GetFromEmailPassword()").Error(err)
		return user, err
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		return user, domain.ErrBadAuth
	}

	return user, nil
}

func (m *UserModel) GetFromTSNet(tsnetIdentifier string) (domain.User, error) {
	var dbUser dbUser

	err := m.stmt.selectTSNet.QueryRowx(tsnetIdentifier).StructScan(&dbUser)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, domain.ErrNoRowsInResultSet
		}
		m.getLogger("GetFromTSNet()").Error(err)
		return domain.User{}, err
	}

	return userFromDB(dbUser), nil
}

// GetAll returns all users.
func (m *UserModel) GetAll() ([]domain.User, error) {
	rows := []dbUser{}

	err := m.stmt.selectAll.Select(&rows)
	if err != nil {
		m.getLogger("GetAll()").Error(err)
		return nil, err
	}
	ret := make([]domain.User, len(rows))
	for i, u := range rows {
		ret[i] = userFromDB(u)
	}

	return ret, nil
}

func userFromDB(u dbUser) domain.User {
	return domain.User{
		UserID:          domain.UserID(u.UserID),
		Email:           u.Email.ForceString(),
		HasPassword:     u.Password.Valid,
		TSNetIdentifier: u.TSNetID.ForceString(),
		TSNetExtraName:  u.TSNetExtra.ForceString(),
	}
}

// IsAdmin tells you if the user is an admin on DropServer
func (m *UserModel) IsAdmin(userID domain.UserID) bool {
	var exists int
	err := m.stmt.selectAdmin.Get(&exists, userID)
	if err != nil {
		m.getLogger("IsAdmin()").Error(err)
		return false
	}

	if exists == 1 {
		return true
	}
	return false
}

// GetAllAdmins returns the list of user ids that are admins
func (m *UserModel) GetAllAdmins() ([]domain.UserID, error) {
	ret := []domain.UserID{}

	err := m.stmt.selectAllAdmins.Select(&ret)
	if err != nil {
		m.getLogger("GetAllAdmins()").Error(err)
		return nil, err
	}

	return ret, nil
}

// MakeAdmin adds the user_id to the table of DropServer admin users
func (m *UserModel) MakeAdmin(userID domain.UserID) error {
	_, err := m.stmt.insertAdmin.Exec(userID)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: admin_users.user_id" {
			return nil
		}
		m.getLogger("MakeAdmin()").Error(err)
		return err
	}
	return nil
}

// DeleteAdmin removes the user id from the tableof admin users for this server.
func (m *UserModel) DeleteAdmin(userID domain.UserID) error {
	_, err := m.stmt.deleteAdmin.Exec(userID)
	if err != nil {
		m.getLogger("DeleteAdmin()").Error(err)
		return err
	}

	return nil
}

func (m *UserModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("UserModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
