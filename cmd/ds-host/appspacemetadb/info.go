package appspacemetadb

import (
	"database/sql"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

const schemaKey = "schema"

// InfoModel interacts with the info table of appspace meata db
type InfoModel struct {
	AppspaceMetaDB interface {
		GetHandle(domain.AppspaceID) (*sqlx.DB, error)
	}
}

// SetDsAPIVersion sets the ds api version
// But do we need this?
// func (m *InfoModel) SetDsAPIVersion() {

// }

// func (m *InfoModel) DsAPIVersion() {

// }

// SetSchema sets the schema in the info db
func (m *InfoModel) SetSchema(appspaceID domain.AppspaceID, schema int) error {
	db, err := m.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM info WHERE name = ?`, schemaKey)
	if err != nil {
		m.getLogger("SetSchema(), Exec Delete").AppspaceID(appspaceID).Error(err)
		// does Exec error if no rows area affected?
		return err
	}

	_, err = db.Exec(`INSERT INTO info (name, value) VALUES (?, ?)`, schemaKey, schema)
	if err != nil {
		m.getLogger("SetSchema(), Exec Insert").AppspaceID(appspaceID).Error(err)
		return err
	}

	return nil
}

//GetSchema returns the schema or 0 if none exists
func (m *InfoModel) GetSchema(appspaceID domain.AppspaceID) (int, error) {
	// for now just read it from the DB?
	// In future, cache it, and invalidate on SetSchema
	db, err := m.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return 0, err
	}

	var v struct {
		Value string
	}

	err = db.Get(&v, `SELECT value FROM info WHERE name = ?`, schemaKey)
	if err != nil {
		// if no-rows, then return 0
		if err == sql.ErrNoRows {
			return 0, nil
		}
		m.getLogger("GetSchema()").AppspaceID(appspaceID).Error(err)
		return 0, err
	}

	schema, err := strconv.Atoi(v.Value)
	if err != nil {
		// log it.
		return 0, err
	}

	return schema, nil
}

func (m *InfoModel) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("InfoModel")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
