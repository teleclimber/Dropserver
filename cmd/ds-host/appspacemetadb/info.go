package appspacemetadb

import (
	"database/sql"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// schemaKey is the feild name that represents that schema of the appspace files
// This schema is set when a migration is run on the appspace data.
const schemaKey = "schema"

// InfoModel interacts with the info table of appspace meata db
type InfoModel struct {
	AppspaceMetaDB interface {
		GetHandle(domain.AppspaceID) (*sqlx.DB, error)
	} `checkinject:"required"`
}

// SetSchema sets the schema of the appspace files in the info db
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

// GetSchema returns the schema of the appspace files or 0 if none exists
func (m *InfoModel) GetSchema(appspaceID domain.AppspaceID) (int, error) {
	// for now just read it from the DB?
	// In future, cache it, and invalidate on SetSchema
	db, err := m.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return 0, err
	}
	schema, err := m.getSchemaWithDB(db)
	if err != nil {
		m.getLogger("GetSchemaWithPath()").AppspaceID(appspaceID).Error(err)
	}
	return schema, err
}

// GetAppspaceMetaInfo returns the schema as stored in the appspace meta db at path
// But this should really return all basic metadata info about the appspace
// app, version, domain, etc...
// Or you could return api version (which should also be stored with appspace presumably)
// .. and then get versioned structs with all info.
func (m *InfoModel) GetAppspaceMetaInfo(dataPath string) (domain.AppspaceMetaInfo, error) {
	db, err := getDb(dataPath)
	if err != nil {
		m.getLogger("GetSchemaWithPath(), getDb()").AddNote(dataPath).Error(err)
		return domain.AppspaceMetaInfo{}, err
	}
	defer db.Close()
	schema, err := m.getSchemaWithDB(db)
	if err != nil {
		m.getLogger("GetSchemaWithPath()").AddNote(dataPath).Error(err)
	}
	return domain.AppspaceMetaInfo{Schema: schema}, err
}
func (m *InfoModel) getSchemaWithDB(db *sqlx.DB) (int, error) {
	var v struct {
		Value string
	}
	err := db.Get(&v, `SELECT value FROM info WHERE name = ?`, schemaKey)
	if err != nil {
		// if no-rows, then return 0
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return strconv.Atoi(v.Value)
}

func (m *InfoModel) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("InfoModel")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
