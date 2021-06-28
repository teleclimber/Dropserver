package appspacemetadb

import (
	"database/sql"
	"strconv"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

const schemaKey = "schema"

// AppspaceInfoModels keeps InfoModels for each appspace
type AppspaceInfoModels struct {
	Config         *domain.RuntimeConfig `checkinject:"required"`
	AppspaceMetaDB domain.AppspaceMetaDB `checkinject:"required"`

	modelsMux sync.Mutex
	models    map[domain.AppspaceID]*InfoModel
}

// Init the data structures as necessary
func (g *AppspaceInfoModels) Init() {
	g.models = make(map[domain.AppspaceID]*InfoModel)
}

// Get returns the route model for the appspace
// There i a single RouteModel per appspaceID so that caching can be implemented in it.
// There will be different route model versions!
func (g *AppspaceInfoModels) Get(appspaceID domain.AppspaceID) domain.AppspaceInfoModel {
	g.modelsMux.Lock()
	defer g.modelsMux.Unlock()

	var m *InfoModel

	m, ok := g.models[appspaceID]
	if !ok {
		// make it and add it
		m = &InfoModel{
			AppspaceMetaDB: g.AppspaceMetaDB,
			appspaceID:     appspaceID,
		}
		g.models[appspaceID] = m
	}

	return m
}

// GetSchema is a convenience function that returns the current schema for the appspaceID
func (g *AppspaceInfoModels) GetSchema(appspaceID domain.AppspaceID) (int, error) {
	m := g.Get(appspaceID)
	return m.GetSchema()
}

// InfoModel interacts with the info table of appspace meata db
type InfoModel struct {
	AppspaceMetaDB domain.AppspaceMetaDB
	appspaceID     domain.AppspaceID
	//do we need stmts? (I think these should be in the DB obj)
}

func (m *InfoModel) getDB() (*sqlx.DB, error) {
	// should probably cache that? Maybe?
	// -> OK, but need to contend with possibility that the conn gets shut down.
	dbConn, err := m.AppspaceMetaDB.GetConn(m.appspaceID) // use location key instead of apspace id
	if err != nil {
		return nil, err
	}
	return dbConn.GetHandle(), err
}

// SetDsAPIVersion sets the ds api version
// But do we need this?
// func (m *InfoModel) SetDsAPIVersion() {

// }

// func (m *InfoModel) DsAPIVersion() {

// }

// SetSchema sets the schema in the info db
func (m *InfoModel) SetSchema(schema int) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM info WHERE name = ?`, schemaKey)
	if err != nil {
		m.getLogger("SetSchema(), Exec Delete").Error(err)
		// does Exec error if no rows area affected?
		return err
	}

	_, err = db.Exec(`INSERT INTO info (name, value) VALUES (?, ?)`, schemaKey, schema)
	if err != nil {
		m.getLogger("SetSchema(), Exec Insert").Error(err)
		return err
	}

	return nil
}

//GetSchema returns the schema or 0 if none exists
func (m *InfoModel) GetSchema() (int, error) {
	// for now just read it from the DB?
	// In future, cache it, and invalidate on SetSchema
	db, err := m.getDB()
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
		m.getLogger("GetSchema()").Error(err)
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
	l := record.NewDsLogger().AddNote("InfoModel").AppspaceID(m.appspaceID)
	if note != "" {
		l.AddNote(note)
	}
	return l
}
