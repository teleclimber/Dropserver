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

type appspaceUser struct {
	ProxyID     domain.ProxyID     `db:"proxy_id"`
	AuthType    string             `db:"auth_type"`
	AuthID      string             `db:"auth_id"`
	DisplayName string             `db:"display_name"`
	Avatar      string             `db:"avatar"`
	Permissions string             `db:"permissions"`
	Created     time.Time          `db:"created"`
	LastSeen    nulltypes.NullTime `db:"last_seen"`
}

// ErrAuthIDExists is returned when the appspace already has a user with that auth_id string
var ErrAuthIDExists = errors.New("auth ID (email or dropid) not unique in this appspace")

// UserModel stores the user's DropIDs
type UserModel struct {
	AppspaceMetaDB interface {
		GetHandle(domain.AppspaceID) (*sqlx.DB, error)
	}
}

// Create an appspace user with provided auth.
func (u *UserModel) Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error) { // acutally should return the User, or at least the proxy id.
	var proxyID domain.ProxyID
	var err error

	if authType != "email" && authType != "dropid" { // We could maybe have a type for auth types if we use this a bunch.
		panic("invalid auth type " + authType)
	}

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return proxyID, err
	}
	p := sqlxprepper.NewPrepper(db)
	stmt := p.Prep(`INSERT INTO users 
		(proxy_id, auth_type, auth_id, created) 
		VALUES (?, ?, ?, datetime("now"))`)

	for {
		proxyID = randomProxyID()
		_, err = stmt.Exec(proxyID, authType, authID)
		if err == nil {
			break
		}
		if err != nil {
			if err.Error() == "UNIQUE constraint failed: users.auth_type, users.auth_id" {
				return domain.ProxyID(""), ErrAuthIDExists
			}
			if err.Error() != "UNIQUE constraint failed: users.proxy_id" {
				// probably need to log it.
				return domain.ProxyID(""), err
			}
			// if we get here it means we had a dupe proxy_id, and therefore generate it again and try again
		}
	}

	return proxyID, nil
}

func (u *UserModel) UpdateAuth(appspaceID domain.AppspaceID, proxyID domain.ProxyID, authType string, authID string) error {
	if authType != "email" && authType != "dropid" { // We could maybe have a type for auth types if we use this a bunch.
		panic("invalid auth type " + authType)
	}
	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}

	p := sqlxprepper.NewPrepper(db)
	stmt := p.Prep(`UPDATE users SET
		auth_type = ?, auth_id = ?
		WHERE proxy_id = ?`)
	_, err = stmt.Stmt.Exec(authType, authID, proxyID)
	if err != nil && err.Error() == "UNIQUE constraint failed: users.auth_type, users.auth_id" {
		return ErrAuthIDExists
	}
	if err != nil {
		u.getLogger("UpdateAuth").AddNote("Stmt.Exec").AppspaceID(appspaceID).Error(err)
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
	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return domain.AppspaceUser{}, err
	}
	p := sqlxprepper.NewPrepper(db)
	stmt := p.Prep(`SELECT * FROM users WHERE proxy_id = ?`)

	var user appspaceUser
	err = stmt.QueryRowx(proxyID).StructScan(&user)
	if err != nil {
		if err != sql.ErrNoRows {
			u.getLogger("Get()").Error(err)
		}
		return domain.AppspaceUser{}, err
	}

	return u.toDomainUser(appspaceID, user), nil
}

// GetByDropID returns an appspace that matches the dropid string
// It returns sql.ErrNoRows if not found
func (u *UserModel) GetByDropID(appspaceID domain.AppspaceID, dropID string) (domain.AppspaceUser, error) {
	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return domain.AppspaceUser{}, err
	}
	p := sqlxprepper.NewPrepper(db)
	stmt := p.Prep(`SELECT * FROM users WHERE auth_type = "dropid" AND auth_id = ?`)

	var user appspaceUser
	err = stmt.QueryRowx(dropID).StructScan(&user)
	if err != nil {
		if err != sql.ErrNoRows {
			u.getLogger("GetByDropID()").Error(err)
		}
		return domain.AppspaceUser{}, err
	}

	return u.toDomainUser(appspaceID, user), nil
}

// GetAll returns an appspace's list of users.
func (u *UserModel) GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error) {
	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return nil, err
	}
	p := sqlxprepper.NewPrepper(db)
	stmt := p.Prep(`SELECT * FROM users`)

	users := []appspaceUser{}
	err = stmt.Select(&users)
	if err != nil {
		u.getLogger("GetAll()").AppspaceID(appspaceID).Error(err)
		return nil, err
	}
	ret := make([]domain.AppspaceUser, len(users))
	for i, user := range users {
		ret[i] = u.toDomainUser(appspaceID, user)
	}
	return ret, nil
}

// Delete the appspace user
// Note: need more thought on what it measn to "delete":
// What happens with the user's data on the appspace?
func (u *UserModel) Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error {
	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}
	p := sqlxprepper.NewPrepper(db)
	stmt := p.Prep(`DELETE FROM users WHERE proxy_id = ?`)

	_, err = stmt.Exec(proxyID)
	if err != nil {
		u.getLogger("Delete()").AppspaceID(appspaceID).Error(err)
		return err
	}
	return nil
}

func (u *UserModel) toDomainUser(appspaceID domain.AppspaceID, user appspaceUser) domain.AppspaceUser {
	// in Go, splitting an empty string return []string{""}, instead of []string{}
	p := []string{}
	if len(user.Permissions) > 0 {
		p = strings.Split(user.Permissions, ",")
	}
	return domain.AppspaceUser{
		AppspaceID:  appspaceID,
		ProxyID:     user.ProxyID,
		AuthType:    user.AuthType,
		AuthID:      user.AuthID,
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
