package appspacemetadb

import (
	"database/sql"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

const schemaKey = "schema"
const vapidPubKey = "vapid-pub-key"
const vapidPrivKey = "vapid-priv-key"

// InfoModel interacts with the info table of appspace meata db
type InfoModel struct {
	AppspaceMetaDB interface {
		GetHandle(domain.AppspaceID) (*sqlx.DB, error)
	} `checkinject:"required"`
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
	schema, err := m.getSchemaWithDB(db)
	if err != nil {
		m.getLogger("GetSchemaWithPath()").AppspaceID(appspaceID).Error(err)
	}
	return schema, err
}

//GetAppspaceMetaInfo returns the schema as stored in the appspace meta db at path
// But this should really return all basic metadata info about the appspace
// app, version, domain, etc...
// Or you could return api version (which should also be stored with appspace presumably)
// .. and then get versioned structs with all info.
func (m *InfoModel) GetAppspaceMetaInfo(dataPath string) (domain.AppspaceMetaInfo, error) {
	db, err := getDb(dataPath)
	defer db.Close()
	if err != nil {
		m.getLogger("GetSchemaWithPath(), getDb()").AddNote(dataPath).Error(err)
		return domain.AppspaceMetaInfo{}, err
	}
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

// We should probably store VAPID keys in INFO,
// But it should probably be versioned
// Returns empty string and nil error if no value
func (m *InfoModel) GetVapidPubKey(appspaceID domain.AppspaceID) (string, error) {
	val, err := m.getStringValue(appspaceID, vapidPubKey)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, nil
}

func (m *InfoModel) GetVapidPrivKey(appspaceID domain.AppspaceID) (string, error) {
	val, err := m.getStringValue(appspaceID, vapidPrivKey)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, nil
}

//
func (m *InfoModel) SetVapidKeys(appspaceID domain.AppspaceID, pubKey, privKey string) error {
	err := m.DeleteVapidKeys(appspaceID)
	if err != nil {
		return err
	}
	db, err := m.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO info (name, value) VALUES (?, ?), (?, ?)`, vapidPrivKey, privKey, vapidPubKey, pubKey)
	if err != nil {
		m.getLogger("SetVapidKeys(), Exec Insert").AppspaceID(appspaceID).Error(err)
		return err
	}
	return err
}

func (m *InfoModel) DeleteVapidKeys(appspaceID domain.AppspaceID) error {
	db, err := m.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM info WHERE name = ? OR name = ?`, vapidPubKey, vapidPrivKey)
	if err != nil {
		m.getLogger("DeleteVapidKeys(), Exec Delete").AppspaceID(appspaceID).Error(err)
	}
	return err
}

func (m *InfoModel) getStringValue(appspaceID domain.AppspaceID, key string) (string, error) {
	db, err := m.AppspaceMetaDB.GetHandle(appspaceID)
	if err != nil {
		return "", err
	}
	var v struct {
		Value string
	}
	err = db.Get(&v, `SELECT value FROM info WHERE name = ?`, key)
	if err != nil {
		return "", err
	}
	return v.Value, err
}

func (m *InfoModel) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("InfoModel")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
