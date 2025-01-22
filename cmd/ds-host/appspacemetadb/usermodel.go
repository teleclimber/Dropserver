package appspacemetadb

import (
	"database/sql"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/nulltypes"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
	"github.com/teleclimber/DropServer/internal/validator"
)

// stmtPreparer has a Preparex for preparing statements
// This type is implemented by sqlx db handle and transaction.
type stmtPreparer interface {
	Preparex(query string) (*sqlx.Stmt, error)
}

type appspaceUser struct {
	ProxyID     domain.ProxyID     `db:"proxy_id"`
	DisplayName string             `db:"display_name"`
	Avatar      string             `db:"avatar"`
	Permissions string             `db:"permissions"`
	Created     time.Time          `db:"created"`
	LastSeen    nulltypes.NullTime `db:"last_seen"`
}

func validateAuthType(t string) bool {
	return t == "email" || t == "dropid" || t == "tsnetid"
}

// ErrAuthIDExists is returned when the appspace already has a user with that auth identifier string
var ErrAuthIDExists = errors.New("auth ID not unique in this appspace")

// UserModel stores the user's DropIDs
type UserModel struct {
	AppspaceMetaDB interface {
		GetHandle(domain.AppspaceID) (*sqlx.DB, error)
	}
}

// Create an appspace user with provided auth.
func (u *UserModel) Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error) { // acutally should return the User, or at least the proxy id.
	log := u.getLogger("Create()").AppspaceID(appspaceID)

	var proxyID domain.ProxyID
	var err error

	if !validateAuthType(authType) {
		panic("invalid auth type " + authType)
	}

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return proxyID, err
	}

	tx, err := db.Beginx()
	if err != nil {
		log.AddNote("Beginx()").Error(err)
		return domain.ProxyID(""), err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`INSERT INTO users 
		(proxy_id, created) 
		VALUES (?, datetime("now"))`)
	if err != nil {
		log.AddNote("Preparex() users").Error(err)
		return domain.ProxyID(""), err
	}

	for {
		proxyID = randomProxyID()
		_, err = stmt.Exec(proxyID)
		if err == nil {
			break
		}
		if err.Error() != "UNIQUE constraint failed: users.proxy_id" {
			log.AddNote("Exec() proxy_id").Error(err)
			return domain.ProxyID(""), err
		}
	}

	err = u.addAuthSP(tx, proxyID, authType, authID)
	if err != nil {
		log.AddNote("addAuthSP").Error(err)
		return domain.ProxyID(""), err
	}

	err = tx.Commit()
	if err != nil {
		u.getLogger("Create(), Commit()").AppspaceID(appspaceID).Error(err)
		return domain.ProxyID(""), err
	}

	return proxyID, nil
}

func (u *UserModel) AddAuth(appspaceID domain.AppspaceID, proxyID domain.ProxyID, authType string, authID string) error {
	if !validateAuthType(authType) {
		panic("invalid auth type " + authType)
	}
	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}
	return u.addAuthSP(db, proxyID, authType, authID)
}

func (u *UserModel) addAuthSP(sp stmtPreparer, proxyID domain.ProxyID, authType string, authID string) error {
	var stmt *sqlx.Stmt
	stmt, err := sp.Preparex(`INSERT INTO user_auth_ids 
		(proxy_id, type, identifier, created) 
		VALUES (?, ?, ?, datetime("now"))`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(proxyID, authType, authID)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: user_auth_ids.type, user_auth_ids.identifier" {
			return ErrAuthIDExists
		}
		return err
	}
	return nil
}

func (u *UserModel) DeleteAuth(appspaceID domain.AppspaceID, proxyID domain.ProxyID, authType string, authID string) error {
	if !validateAuthType(authType) {
		panic("invalid auth type " + authType)
	}
	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM user_auth_ids WHERE proxy_id = ? AND type = ? and identifier = ?`, proxyID, authType, authID)
	if err != nil {
		u.getLogger("DeleteAuth").AppspaceID(appspaceID).Error(err)
		return err
	}
	return nil
}

// UpdateMeta updates the appspace-facing data for the user
func (u *UserModel) UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, permissions []string) error {
	err := validatePermissions(permissions)
	if err != nil {
		return err
	}
	err = validator.UserProxyID(string(proxyID))
	if err != nil {
		return err
	}

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}
	p := sqlxprepper.NewPrepper(db)
	stmt := p.Prep(`UPDATE users SET 
		display_name = ?, avatar = ?, permissions = ?
		WHERE proxy_id = ?`)

	_, err = stmt.Stmt.Exec(displayName, avatar, strings.Join(permissions, ","), proxyID)
	if err != nil {
		u.getLogger("UpdateMeta").AddNote("updateMeta.Stmt.Exec").AppspaceID(appspaceID).Error(err)
		return err
	}
	return nil
}

// Get returns an AppspaceUser
func (u *UserModel) Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error) {
	log := u.getLogger("Get()").AppspaceID(appspaceID)
	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return domain.AppspaceUser{}, err
	}

	tx, err := db.Beginx()
	if err != nil {
		log.AddNote("Beginx()").Error(err)
		return domain.AppspaceUser{}, err
	}
	defer tx.Rollback()

	user, err := getUser(tx, proxyID)
	if err == sql.ErrNoRows {
		return domain.AppspaceUser{}, domain.ErrNoRowsInResultSet
	} else if err != nil {
		log.AddNote("getUser()").Error(err)
		return domain.AppspaceUser{}, err
	}

	auths, err := getUserAuths(tx, proxyID)
	if err != nil {
		log.AddNote("getUserAuths()").Error(err)
		return domain.AppspaceUser{}, err
	}

	err = tx.Commit()
	if err != nil {
		log.AddNote("Commit()").Error(err)
		return domain.AppspaceUser{}, err
	}

	return u.toDomainUser(appspaceID, user, auths), nil
}

func getUser(sp stmtPreparer, proxyID domain.ProxyID) (user appspaceUser, err error) {
	var stmt *sqlx.Stmt
	stmt, err = sp.Preparex(`SELECT * FROM users WHERE proxy_id = ?`)
	if err != nil {
		return
	}
	err = stmt.Get(&user, proxyID)
	return
}

func getUserAuths(sp stmtPreparer, proxyID domain.ProxyID) (auths []domain.AppspaceUserAuth, err error) {
	var stmt *sqlx.Stmt
	stmt, err = sp.Preparex(`SELECT type, identifier, created, last_seen FROM user_auth_ids WHERE proxy_id = ?`)
	if err != nil {
		return
	}
	err = stmt.Select(&auths, proxyID)
	return
}

// GetByAuth returns an AppspaceUser that matches the auth strings
// It returns domain.ErrNoRowsInResultSet if not found
func (u *UserModel) GetByAuth(appspaceID domain.AppspaceID, authType string, identifier string) (domain.AppspaceUser, error) {
	log := u.getLogger("GetByAuth()").AppspaceID(appspaceID)

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return domain.AppspaceUser{}, err
	}

	tx, err := db.Beginx()
	if err != nil {
		log.AddNote("Beginx()").Error(err)
		return domain.AppspaceUser{}, err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`SELECT proxy_id FROM user_auth_ids WHERE type = ? AND identifier = ?`)
	if err != nil {
		log.AddNote("Preparex()").Error(err)
		return domain.AppspaceUser{}, err
	}

	var proxyID domain.ProxyID
	err = stmt.Get(&proxyID, authType, identifier)
	if err == sql.ErrNoRows {
		return domain.AppspaceUser{}, domain.ErrNoRowsInResultSet
	} else if err != nil {
		log.AddNote("Get()").Error(err)
		return domain.AppspaceUser{}, err
	}

	user, err := getUser(tx, proxyID)
	if err == sql.ErrNoRows { // This happens if DB has an auth for a non-existenting Proxy id.
		return domain.AppspaceUser{}, domain.ErrNoRowsInResultSet
	} else if err != nil {
		log.AddNote("getUser()").Error(err)
		return domain.AppspaceUser{}, err
	}

	auths, err := getUserAuths(tx, proxyID)
	if err != nil {
		log.AddNote("getUserAuths()").Error(err)
		return domain.AppspaceUser{}, err
	}

	err = tx.Commit()
	if err != nil {
		log.AddNote("Commit()").Error(err)
		return domain.AppspaceUser{}, err
	}

	return u.toDomainUser(appspaceID, user, auths), nil
}

// GetAll returns an appspace's list of users.
func (u *UserModel) GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error) {
	log := u.getLogger("Create()").AppspaceID(appspaceID)

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return nil, err
	}

	tx, err := db.Beginx()
	if err != nil {
		log.AddNote("Beginx()").Error(err)
		return []domain.AppspaceUser{}, err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`SELECT * FROM users`)
	if err != nil {
		log.AddNote("Preparex()").Error(err)
		return []domain.AppspaceUser{}, err
	}

	users := []appspaceUser{}
	err = stmt.Select(&users)
	if err != nil {
		log.AddNote("Select() users").Error(err)
		return nil, err
	}

	ret := make([]domain.AppspaceUser, len(users))
	for i, user := range users {
		auths, err := getUserAuths(tx, user.ProxyID)
		if err != nil {
			log.AddNote("getUserAuths").Error(err)
			return nil, err
		}
		ret[i] = u.toDomainUser(appspaceID, user, auths)
	}

	err = tx.Commit()
	if err != nil {
		log.AddNote("Commit()").Error(err)
		return []domain.AppspaceUser{}, err
	}

	return ret, nil
}

// Delete the appspace user
// Note: need more thought on what it measn to "delete":
// What happens with the user's data on the appspace?
func (u *UserModel) Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error {
	log := u.getLogger("Delete()").AppspaceID(appspaceID)

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		log.AddNote("Beginx()").Error(err)
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM users WHERE proxy_id = ?`, proxyID)
	if err != nil {
		log.AddNote("Delete from users").Error(err)
		return err
	}

	_, err = tx.Exec(`DELETE FROM user_auth_ids WHERE proxy_id = ?`, proxyID)
	if err != nil {
		log.AddNote("Delete from auth_ids").Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.AddNote("Commit()").Error(err)
		return err
	}

	return nil
}

// here we should probably pass auths array too.
func (u *UserModel) toDomainUser(appspaceID domain.AppspaceID, user appspaceUser, auths []domain.AppspaceUserAuth) domain.AppspaceUser {
	// in Go, splitting an empty string return []string{""}, instead of []string{}
	p := []string{}
	if len(user.Permissions) > 0 {
		p = strings.Split(user.Permissions, ",")
	}
	return domain.AppspaceUser{
		AppspaceID:  appspaceID,
		ProxyID:     user.ProxyID,
		Auths:       auths,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		Permissions: p,
		Created:     user.Created,
		LastSeen:    user.LastSeen,
	}
}

func (u *UserModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("Appspace UserModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

func validatePermissions(permissions []string) error {
	for _, p := range permissions {
		err := validator.AppspacePermission(p)
		if err != nil {
			return err
		}
	}
	return nil
}

// //////////
// random string
const chars36 = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand2 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomProxyID() domain.ProxyID {
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars36[seededRand2.Intn(len(chars36))]
	}
	return domain.ProxyID(string(b))
}
