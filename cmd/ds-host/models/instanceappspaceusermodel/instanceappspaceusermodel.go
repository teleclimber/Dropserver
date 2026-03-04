package instanceappspaceusermodel

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// InstanceAppspaceModel handles instance_appspace_users table access
type InstanceAppspaceModel struct {
	DB *domain.DB

	stmt struct {
		selectBoth       *sqlx.Stmt
		selectByUser     *sqlx.Stmt
		selectByAppspace *sqlx.Stmt
		insert           *sqlx.Stmt
		delete           *sqlx.Stmt
	}
}

func (m *InstanceAppspaceModel) PrepareStatements() {
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	m.stmt.selectBoth = p.Prep(`SELECT * FROM instance_appspace_users WHERE user_id = ? AND appspace_id = ?`)
	m.stmt.selectByUser = p.Prep(`SELECT * FROM instance_appspace_users WHERE user_id = ?`)
	m.stmt.selectByAppspace = p.Prep(`SELECT * FROM instance_appspace_users WHERE appspace_id = ?`)

	m.stmt.insert = p.Prep(`INSERT INTO instance_appspace_users (user_id, appspace_id, proxy_id) VALUES (?, ?, ?)`)
	m.stmt.delete = p.Prep(`DELETE FROM instance_appspace_users WHERE user_id = ? AND appspace_id = ?`)
}

// Get returns the single mapping for (user_id, appspace_id).
// Returns sql.ErrNoRows if not found.
func (m *InstanceAppspaceModel) Get(userID domain.UserID, appspaceID domain.AppspaceID) (domain.InstanceAppspaceUser, error) {
	var ia domain.InstanceAppspaceUser
	err := m.stmt.selectBoth.QueryRowx(userID, appspaceID).StructScan(&ia)
	if err != nil {
		if err != sql.ErrNoRows {
			m.getLogger("Get").UserID(userID).Error(err)
			return ia, err
		}
		return ia, domain.ErrNoRowsInResultSet
	}
	return ia, nil
}

func (m *InstanceAppspaceModel) GetForUser(userID domain.UserID) ([]domain.InstanceAppspaceUser, error) {
	ret := []domain.InstanceAppspaceUser{}
	err := m.stmt.selectByUser.Select(&ret, userID)
	if err != nil {
		m.getLogger("GetForUser").UserID(userID).Error(err)
		return nil, err
	}
	return ret, nil
}

func (m *InstanceAppspaceModel) GetForAppspace(appspaceID domain.AppspaceID) ([]domain.InstanceAppspaceUser, error) {
	ret := []domain.InstanceAppspaceUser{}
	err := m.stmt.selectByAppspace.Select(&ret, appspaceID)
	if err != nil {
		m.getLogger("GetForAppspace").AppspaceID(appspaceID).Error(err)
		return nil, err
	}
	return ret, nil
}

func (m *InstanceAppspaceModel) Create(userID domain.UserID, appspaceID domain.AppspaceID, proxyID domain.ProxyID) error {
	_, err := m.stmt.insert.Exec(userID, appspaceID, proxyID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return errors.New("mapping already exists")
		}
		m.getLogger("Create").Error(err)
		return err
	}
	return nil
}

func (m *InstanceAppspaceModel) Delete(userID domain.UserID, appspaceID domain.AppspaceID) error {
	_, err := m.stmt.delete.Exec(userID, appspaceID)
	if err != nil {
		m.getLogger("Delete").Error(err)
		return err
	}
	return nil
}

// func (m *InstanceAppspaceModel) SetUsersForAppspace(appspaceID domain.AppspaceID, userIDs []domain.UserID) error {
// 	tx, err := m.DB.Handle.Beginx()
// 	if err != nil {
// 		m.getLogger("SetUsersForAppspace(), Beginx()").AppspaceID(appspaceID).Error(err)
// 		return err
// 	}
// 	defer tx.Rollback()

// 	// remove all existing mappings for this appspace
// 	delStmt, err := tx.Preparex(`DELETE FROM instance_appspace_users WHERE appspace_id = ?`)
// 	if err != nil {
// 		m.getLogger("SetUsersForAppspace(), Preparex DELETE").AppspaceID(appspaceID).Error(err)
// 		return err
// 	}
// 	_, err = delStmt.Exec(appspaceID)
// 	delStmt.Close()
// 	if err != nil {
// 		m.getLogger("SetUsersForAppspace(), Exec DELETE").AppspaceID(appspaceID).Error(err)
// 		return err
// 	}

// 	// insert new mappings
// 	if len(userIDs) > 0 {
// 		insStmt, err := tx.Preparex(`INSERT INTO instance_appspace_users (user_id, appspace_id) VALUES (?, ?)`)
// 		if err != nil {
// 			m.getLogger("SetUsersForAppspace(), Preparex INSERT").AppspaceID(appspaceID).Error(err)
// 			return err
// 		}
// 		for _, uid := range userIDs {
// 			_, err := insStmt.Exec(uid, appspaceID)
// 			if err != nil {
// 				insStmt.Close()
// 				m.getLogger("SetUsersForAppspace(), Exec INSERT").AppspaceID(appspaceID).UserID(uid).Error(err)
// 				return err
// 			}
// 		}
// 		insStmt.Close()
// 	}

// 	if err := tx.Commit(); err != nil {
// 		m.getLogger("SetUsersForAppspace(), Commit").AppspaceID(appspaceID).Error(err)
// 		return err
// 	}

// 	return nil
// }

func (m *InstanceAppspaceModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("InstanceAppspaceModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
