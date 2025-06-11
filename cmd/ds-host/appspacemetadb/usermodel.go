package appspacemetadb

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/validator"
)

// stmtPreparer has a Preparex for preparing statements
// This type is implemented by sqlx db handle and transaction.
type stmtPreparer interface {
	Preparex(query string) (*sqlx.Stmt, error)
}

type appspaceUser struct {
	ProxyID     domain.ProxyID `db:"proxy_id"`
	DisplayName string         `db:"display_name"`
	Avatar      string         `db:"avatar"`
	Permissions string         `db:"permissions"`
	Created     time.Time      `db:"created"`
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
func (u *UserModel) Create(appspaceID domain.AppspaceID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) (domain.ProxyID, error) {
	log := u.getLogger("Create()").AppspaceID(appspaceID).Clone

	var proxyID domain.ProxyID
	var err error

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return proxyID, err
	}

	tx, err := db.Beginx()
	if err != nil {
		log().AddNote("Beginx()").Error(err)
		return domain.ProxyID(""), err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`INSERT INTO users 
		(proxy_id, created) 
		VALUES (?, datetime("now"))`)
	if err != nil {
		log().AddNote("Preparex() users").Error(err)
		return domain.ProxyID(""), err
	}

	for {
		proxyID = randomProxyID()
		_, err = stmt.Exec(proxyID)
		if err == nil {
			break
		}
		if err.Error() != "UNIQUE constraint failed: users.proxy_id" {
			log().AddNote("Exec() proxy_id").Error(err)
			return domain.ProxyID(""), err
		}
	}

	err = updateMetaSP(tx, proxyID, displayName, avatar)
	if err != nil {
		log().AddNote("updateMetaSP()").Error(err)
		return domain.ProxyID(""), err
	}

	// any auth passed to Create is "add"
	for i := range auths {
		auths[i].Operation = domain.EditOperationAdd
	}
	err = updateAuthsSP(tx, proxyID, auths, false)
	if err != nil {
		log().AddNote("updateAuthsSP").Error(err)
		return domain.ProxyID(""), err
	}

	err = tx.Commit()
	if err != nil {
		log().AddNote("Commit").Error(err)
		return domain.ProxyID(""), err
	}

	return proxyID, nil
}

func (u *UserModel) UpdateAuth(appspaceID domain.AppspaceID, proxyID domain.ProxyID, auth domain.EditAppspaceUserAuth) error {
	log := u.getLogger("UpdateAuth()").AppspaceID(appspaceID).Clone

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}

	tx, err := db.Beginx()
	if err != nil {
		log().AddNote("Beginx()").Error(err)
		return err
	}
	defer tx.Rollback()

	err = updateAuthSP(tx, proxyID, auth, true)
	if err != nil {
		log().AddNote("updateAuthSP()").Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log().AddNote("Commit").Error(err)
	}
	return err
}

func updateAuthsSP(sp stmtPreparer, proxyID domain.ProxyID, auths []domain.EditAppspaceUserAuth, allowRemove bool) error {
	var err error
	for _, auth := range auths {
		err = updateAuthSP(sp, proxyID, auth, allowRemove)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateAuthSP(sp stmtPreparer, proxyID domain.ProxyID, auth domain.EditAppspaceUserAuth, allowRemove bool) error {
	var err error
	if auth.Operation == domain.EditOperationAdd {
		err = addAuthSP(sp, proxyID, auth.Type, auth.Identifier, auth.ExtraName)
		if err != nil {
			return err
		}
	} else if auth.Operation == domain.EditOperationRemove {
		if !allowRemove {
			return fmt.Errorf("got a remove op with allowRemove false: %s %s", auth.Type, auth.Identifier)
		}
		err = deleteAuthSP(sp, proxyID, auth.Type, auth.Identifier)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unknown operation (\"%s\") for %s %s", auth.Operation, auth.Type, auth.Identifier)
	}
	return nil
}

func addAuthSP(sp stmtPreparer, proxyID domain.ProxyID, authType string, authID string, extraName string) error {
	err := validator.AppspaceUserAuthType(authType)
	if err != nil {
		return err
	}
	stmt, err := sp.Preparex(`INSERT INTO user_auth_ids 
		(proxy_id, type, identifier, extra_name, created) 
		VALUES (?, ?, ?, ?, datetime("now"))`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(proxyID, authType, authID, extraName)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: user_auth_ids.type, user_auth_ids.identifier" {
			return ErrAuthIDExists
		}
		return err
	}
	return nil
}

func deleteAuthSP(sp stmtPreparer, proxyID domain.ProxyID, authType string, authID string) error {
	stmt, err := sp.Preparex(`DELETE FROM user_auth_ids WHERE proxy_id = ? AND type = ? and identifier = ?`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(proxyID, authType, authID)
	return err
}

// Update the appspace user
// This should be broken up into multiple functions
func (u *UserModel) Update(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) error {
	log := u.getLogger("Update").AppspaceID(appspaceID).Clone

	err := validator.UserProxyID(string(proxyID)) // why validate this here?
	if err != nil {
		log().AddNote("validator.UserProxyID").Error(err)
		return err
	}

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}
	tx, err := db.Beginx()
	if err != nil {
		log().AddNote("Beginx()").Error(err)
		return err
	}
	defer tx.Rollback()

	err = updateMetaSP(tx, proxyID, displayName, avatar)
	if err != nil {
		log().AddNote("updateMetaSP()").Error(err)
		return err
	}

	err = updateAuthsSP(tx, proxyID, auths, true)
	if err != nil {
		log().AddNote("updateAuthsSP()").Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log().AddNote("Commit()").Error(err)
		return err
	}

	return nil
}

func updateMetaSP(sp stmtPreparer, proxyID domain.ProxyID, displayName string, avatar string) error {
	stmt, err := sp.Preparex(`UPDATE users SET 
		display_name = ?, avatar = ?
		WHERE proxy_id = ?`)
	if err != nil {
		return err
	}
	_, err = stmt.Stmt.Exec(displayName, avatar, proxyID)
	return err
}

func (u *UserModel) UpdateAvatar(appspaceID domain.AppspaceID, proxyID domain.ProxyID, avatar string) error {
	log := u.getLogger("Update").AppspaceID(appspaceID).Clone

	db, err := u.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}
	tx, err := db.Beginx()
	if err != nil {
		log().AddNote("Beginx()").Error(err)
		return err
	}
	defer tx.Rollback()

	err = updateAvatarSP(tx, proxyID, avatar)
	if err != nil {
		log().AddNote("updateAvatarSP()").Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log().AddNote("Commit()").Error(err)
		return err
	}
	return nil
}

func updateAvatarSP(sp stmtPreparer, proxyID domain.ProxyID, avatar string) error {
	stmt, err := sp.Preparex(`UPDATE users SET avatar = ? WHERE proxy_id = ?`)
	if err != nil {
		return err
	}
	_, err = stmt.Stmt.Exec(avatar, proxyID)
	return err
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
	stmt, err = sp.Preparex(`SELECT type, identifier, extra_name, created FROM user_auth_ids WHERE proxy_id = ?`)
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
	}
}

func (u *UserModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("Appspace UserModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
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
