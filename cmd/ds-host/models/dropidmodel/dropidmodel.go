package dropidmodel

import (
	"database/sql"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// DropIDModel stores the user's DropIDs
type DropIDModel struct {
	DB *domain.DB

	stmt struct {
		createDropID   *sqlx.Stmt
		updateDropID   *sqlx.Stmt
		deleteDropID   *sqlx.Stmt
		getDropID      *sqlx.Stmt
		getUserDropIDs *sqlx.Stmt
	}
}

// Regarding case sensitivity
// domain name I'm not too concerned about
// But handle should be stored such taht capitalization is preserved
// yet comparison should be case insensitive.
// Plenty of ways to do that with sqlite.
// https://www.designcise.com/web/tutorial/how-to-do-case-insensitive-comparisons-in-sqlite
// Reason is handle might be used as display name, and case insensitivity makes ugly names
// --> actually sqlite doesn't do well with non-ASCII chars out of the box

// PrepareStatements for appspace model
func (m *DropIDModel) PrepareStatements() {
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	m.stmt.createDropID = p.Prep(`INSERT INTO dropids 
		(user_id, handle, domain, display_name, created) 
		VALUES (?, ?, ?, ?, datetime("now"))`)

	m.stmt.updateDropID = p.Prep(`UPDATE dropids SET 
		display_name = ?
		WHERE user_id = ? AND handle = ? AND domain = ?`)

	m.stmt.deleteDropID = p.Prep(`DELETE FROM dropids WHERE user_id = ? AND handle = ? AND domain = ?`)

	m.stmt.getDropID = p.Prep(`SELECT * FROM dropids WHERE handle = ? AND domain = ?`)

	m.stmt.getUserDropIDs = p.Prep(`SELECT * FROM dropids WHERE user_id = ?`)
}

// Create a DropID
func (m *DropIDModel) Create(userID domain.UserID, handle string, dom string, displayName string) (domain.DropID, error) {
	logger := m.getLogger("Create()").UserID(userID)

	_, err := m.stmt.createDropID.Exec(userID, strings.ToLower(handle), strings.ToLower(dom), displayName)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.DropID{}, domain.ErrUniqueConstraintViolation
		}
		logger.AddNote("insert").Error(err)
		return domain.DropID{}, err
	}

	return m.Get(handle, dom)
}

// Update a DropID
func (m *DropIDModel) Update(userID domain.UserID, handle string, dom string, displayName string) error {
	result, err := m.stmt.updateDropID.Exec(displayName, userID, strings.ToLower(handle), strings.ToLower(dom))
	if err != nil {
		m.getLogger("Update").AddNote("Exec").Error(err)
		return err
	}
	num, err := result.RowsAffected()
	if err != nil {
		m.getLogger("Update").AddNote("RowsAffected").Error(err)
		return err
	}
	if num == 0 {
		return domain.ErrNoRowsAffected
	}
	return nil
}

// Get returns the DropID if found.
// It returns sql.ErrNoRows error if not found.
func (m *DropIDModel) Get(handle string, dom string) (domain.DropID, error) {
	var dropID domain.DropID
	err := m.stmt.getDropID.QueryRowx(strings.ToLower(handle), strings.ToLower(dom)).StructScan(&dropID)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.DropID{}, domain.ErrNoRowsInResultSet
		} else {
			m.getLogger("Get()").Error(err)
			return domain.DropID{}, err
		}
	}

	return dropID, nil
}

// GetForUser returns a user's dropIDs
// Empty array is returned if none are found
func (m *DropIDModel) GetForUser(userID domain.UserID) ([]domain.DropID, error) {
	ret := []domain.DropID{}

	err := m.stmt.getUserDropIDs.Select(&ret, userID)
	if err != nil {
		m.getLogger("GetForUser()").UserID(userID).Error(err)
		return nil, err
	}

	return ret, nil
}

// Delete the drop id.
func (m *DropIDModel) Delete(userID domain.UserID, handle string, dom string) error {
	_, err := m.stmt.deleteDropID.Exec(userID, handle, dom)
	if err != nil {
		m.getLogger("Delete()").AddNote("Exec").Error(err)
		return err
	}
	return nil
}

func (m *DropIDModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("DropIDModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
