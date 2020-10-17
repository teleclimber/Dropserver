package appmodel

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// Note we will have application
// ..and application versions
// So two tables
//

// AppModel represents the model for app
type AppModel struct {
	DB *domain.DB
	// need config to select db type?

	stmt struct {
		selectID         *sqlx.Stmt
		selectOwner      *sqlx.Stmt
		insertApp        *sqlx.Stmt
		selectVersion    *sqlx.Stmt
		selectAppVerions *sqlx.Stmt
		insertVersion    *sqlx.Stmt
		deleteVersion    *sqlx.Stmt
	}
}

type prepper struct {
	handle *sqlx.DB
	err    error
}

func (p *prepper) exec(query string) *sqlx.Stmt {
	if p.err != nil {
		return nil
	}

	stmt, err := p.handle.Preparex(query)
	if err != nil {
		p.err = errors.New("Error preparing statmement " + query + " " + err.Error())
		return nil
	}

	return stmt
}

// PrepareStatements prepares the statements
func (m *AppModel) PrepareStatements() {
	p := prepper{handle: m.DB.Handle}

	//get from ID
	m.stmt.selectID = p.exec(`SELECT * FROM apps WHERE app_id = ?`)

	//get for a given owner user ID
	m.stmt.selectOwner = p.exec(`SELECT * FROM apps WHERE owner_id = ?`)

	// insert app:
	m.stmt.insertApp = p.exec(`INSERT INTO apps 
		("owner_id", "name", "created") VALUES (?, ?, datetime("now"))`)

	// get version
	m.stmt.selectVersion = p.exec(`SELECT * FROM app_versions WHERE app_id = ? AND version = ?`)

	// get versions for app
	m.stmt.selectAppVerions = p.exec(`SELECT * FROM app_versions WHERE app_id = ?`)

	m.stmt.insertVersion = p.exec(`INSERT INTO app_versions
		("app_id", "version", "schema", "api", "location_key", created) VALUES (?, ?, ?, ?, ?, datetime("now"))`)

	m.stmt.deleteVersion = p.exec(`DELETE FROM app_versions WHERE app_id = ? AND version = ?`)

	if p.err != nil {
		panic(p.err)
	}
}

// Additional methods:
// - GetForUser
// - Get versions
// - Delete, DeleteVersion
// Some of the other methods from nodejs impl prob belong in trusted

// GetFromID gets the app using its unique ID on the system
// It returns an error if ID is not found
func (m *AppModel) GetFromID(appID domain.AppID) (*domain.App, domain.Error) {
	var app domain.App
	err := m.stmt.selectID.QueryRowx(appID).StructScan(&app)
	if err != nil {
		m.getLogger("GetFromID()").AppID(appID).Error(err)
		return nil, dserror.FromStandard(err)
	}
	return &app, nil
}

// GetForOwner returns array of application data for a given user
func (m *AppModel) GetForOwner(userID domain.UserID) ([]*domain.App, domain.Error) {
	ret := []*domain.App{}

	err := m.stmt.selectOwner.Select(&ret, userID)
	if err != nil {
		m.getLogger("GetForOwner()").UserID(userID).Error(err)
		return nil, dserror.FromStandard(err)
	}

	return ret, nil
}

// Create adds an app to the database
// This should return an unique ID, right?
// Other arguments: owner, and possibly other things like create date
// Should we have CreateArgs type struct to guarantee proper data passing? -> yes
func (m *AppModel) Create(ownerID domain.UserID, name string) (*domain.App, domain.Error) {
	// location key isn't here. It's in a version.
	// do we check name and locationKey for epty string or excess length?
	// -> probably, yes. Or where should that actually happen?

	r, err := m.stmt.insertApp.Exec(ownerID, name)
	if err != nil {
		m.getLogger("Create(), insertApp.Exec()").UserID(ownerID).Error(err)
		return nil, dserror.FromStandard(err)
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		m.getLogger("Create(), r.LastInsertId()").UserID(ownerID).Error(err)
		return nil, dserror.FromStandard(err)
	}
	if lastID >= 0xFFFFFFFF {
		m.getLogger("Create()").Log(fmt.Sprintf("Last insert ID out of bounds: %v", lastID))
		return nil, dserror.New(dserror.OutOFBounds, "Last Insert ID from DB greater than uint32")
	}

	appID := domain.AppID(lastID)

	app, dsErr := m.GetFromID(appID)
	if dsErr != nil {
		m.getLogger("Create(), GetFromID()").Error(dsErr.ToStandard())
		return nil, dsErr
	}

	return app, nil
}

// GetVersion returns the version for the app
func (m *AppModel) GetVersion(appID domain.AppID, version domain.Version) (*domain.AppVersion, domain.Error) {
	var appVersion domain.AppVersion

	err := m.stmt.selectVersion.QueryRowx(appID, version).StructScan(&appVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dserror.New(dserror.NoRowsInResultSet)
		}
		m.getLogger("GetVersion()").AppID(appID).AppVersion(version).Error(err)
		return nil, dserror.FromStandard(err)
	}

	return &appVersion, nil
}

// GetVersionsForApp returns an array of versions of code for that application
func (m *AppModel) GetVersionsForApp(appID domain.AppID) ([]*domain.AppVersion, domain.Error) {
	ret := []*domain.AppVersion{}

	err := m.stmt.selectAppVerions.Select(&ret, appID)
	if err != nil {
		m.getLogger("GetVersionsForApp()").AppID(appID).Error(err)
		return nil, dserror.FromStandard(err)
	}

	return ret, nil
}

// CreateVersion adds a new version for an app in the DB
// has appid, version, location key, create date
// use appid and version as primary keys
// index on appid as well
func (m *AppModel) CreateVersion(appID domain.AppID, version domain.Version, schema int, api domain.APIVersion, locationKey string) (*domain.AppVersion, domain.Error) {
	// TODO: this should fail if version exists
	// .. but that should be caught by the route first.

	_, err := m.stmt.insertVersion.Exec(appID, version, schema, api, locationKey)
	if err != nil {
		m.getLogger("CreateVersion(), insertVersion").AppID(appID).AppVersion(version).Error(err)
		return nil, dserror.FromStandard(err)
	}

	appVersion, dsErr := m.GetVersion(appID, version)
	if dsErr != nil {
		m.getLogger("CreateVersion(), GetVersion").AppID(appID).AppVersion(version).Error(err)
		return nil, dsErr
	}

	return appVersion, nil
}

// DeleteVersion removes a version from the DB
func (m *AppModel) DeleteVersion(appID domain.AppID, version domain.Version) domain.Error {
	_, err := m.stmt.deleteVersion.Exec(appID, version)
	if err != nil {
		m.getLogger("DeleteVersion()").AppID(appID).AppVersion(version).Error(err)
		return dserror.FromStandard(err)
	}
	return nil
}

func (m *AppModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
