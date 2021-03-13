package appspaceusermodel

import (
	"database/sql"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
	"github.com/teleclimber/DropServer/internal/validator"
)

// ErrAuthIDExists is returned when the appspace already has a user with that auth_id string
var ErrAuthIDExists = errors.New("Auth ID (email or dropid) not unique in this appspace")

// AppspaceUserModel stores the user's DropIDs
type AppspaceUserModel struct {
	DB *domain.DB

	stmt struct {
		insert         *sqlx.Stmt
		updateAuth     *sqlx.Stmt
		updateMeta     *sqlx.Stmt
		updateLastSeen *sqlx.Stmt
		delete         *sqlx.Stmt
		get            *sqlx.Stmt
		getForAppspace *sqlx.Stmt
	}
}

// PrepareStatements for appspace model
func (m *AppspaceUserModel) PrepareStatements() {
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	m.stmt.insert = p.Prep(`INSERT INTO appspace_users 
		(appspace_id, proxy_id, auth_type, auth_id, created) 
		VALUES (?, ?, ?, ?, datetime("now"))`)

	m.stmt.updateAuth = p.Prep(`UPDATE appspace_users SET 
		auth_type = ?, auth_id = ?
		WHERE appspace_id = ? AND proxy_id = ?`)

	m.stmt.updateMeta = p.Prep(`UPDATE appspace_users SET 
		display_name = ?, permissions = ?
		WHERE appspace_id = ? AND proxy_id = ?`)

	m.stmt.updateLastSeen = p.Prep(`UPDATE appspace_users SET 
		last_seen = datetime("now")
		WHERE appspace_id = ? AND proxy_id = ?`)

	m.stmt.delete = p.Prep(`DELETE FROM appspace_users WHERE appspace_id = ? AND proxy_id = ?`)

	m.stmt.get = p.Prep(`SELECT * FROM appspace_users WHERE appspace_id = ? AND proxy_id = ?`)

	m.stmt.getForAppspace = p.Prep(`SELECT * FROM appspace_users WHERE appspace_id = ?`)
}

// Create an appspace user with provided auth.
func (m *AppspaceUserModel) Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error) { // acutally should return the User, or at least the proxy id.

	if authType != "email" && authType != "dropid" { // We could maybe have a type for auth types if we use this a bunch.
		panic("invalid auth type " + authType)
	}

	//var result sql.Result
	var proxyID domain.ProxyID
	var err error
	for {
		proxyID = randomProxyID()
		_, err = m.stmt.insert.Exec(appspaceID, proxyID, authType, authID)
		if err == nil {
			break
		}
		if err != nil {
			if err.Error() == "UNIQUE constraint failed: appspace_users.appspace_id, appspace_users.auth_type, appspace_users.auth_id" {
				return domain.ProxyID(""), ErrAuthIDExists
			}
			if err.Error() != "UNIQUE constraint failed: appspace_users.appspace_id, appspace_users.proxy_id" {
				// probably need to log it.
				return domain.ProxyID(""), err
			}
			// if we get here it means we had a dupe proxy_id, and therefore generate it again and try again
		}
	}

	return proxyID, nil
}

// You probably will have an update auth function, but I'm not sure exactly what form that will take.

// UpdateMeta updates the appspace-facing data for the user
func (m *AppspaceUserModel) UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, permissions []string) error {
	err := validatePermissions(permissions)
	if err != nil {
		return err
	}
	err = validator.UserProxyID(string(proxyID))
	if err != nil {
		return err
	}

	_, err = m.stmt.updateMeta.Stmt.Exec(displayName, strings.Join(permissions, ","), appspaceID, proxyID)
	if err != nil {
		m.getLogger("UpdateMeta").AddNote("updateMeta.Stmt.Exec").AppspaceID(appspaceID).Error(err)
		return err
	}
	return nil
}

// Get returns an AppspaceUser
func (m *AppspaceUserModel) Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error) {
	var appspaceUser domain.AppspaceUser
	err := m.stmt.get.QueryRowx(appspaceID, proxyID).StructScan(&appspaceUser)
	if err != nil {
		if err != sql.ErrNoRows {
			m.getLogger("Get()").Error(err)
		}
		return domain.AppspaceUser{}, err
	}

	return appspaceUser, nil
}

// GetForAppspace returns an appspace's list of users.
func (m *AppspaceUserModel) GetForAppspace(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error) {
	appspaceUsers := []domain.AppspaceUser{}
	err := m.stmt.getForAppspace.Select(&appspaceUsers, appspaceID)
	if err != nil {
		m.getLogger("GetForAppspace()").AppspaceID(appspaceID).Error(err)
		return nil, err
	}
	return appspaceUsers, nil
}

// Delete the appspace user
// Note: need more thought on what it measn to "delete":
// What happens with the user's data on the appspace?
func (m *AppspaceUserModel) Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error {
	_, err := m.stmt.delete.Exec(appspaceID, proxyID)
	if err != nil {
		m.getLogger("Delete()").AppspaceID(appspaceID).Error(err)
		return err
	}
	return nil
}

func (m *AppspaceUserModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceUserModel")
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

////////////
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
