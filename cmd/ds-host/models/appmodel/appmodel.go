package appmodel

import (
	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
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
	Logger domain.LogCLientI

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

// PrepareStatements prepares the statements
func (m *AppModel) PrepareStatements() {
	// Here is the place to get clever with statements if using multiple DBs.

	var err error

	//get from ID
	m.stmt.selectID, err = m.DB.Handle.Preparex(`SELECT * FROM apps WHERE app_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement SELECT * FROM apps..."+err.Error())
		panic(err)
	}

	//get for a given owner user ID
	m.stmt.selectOwner, err = m.DB.Handle.Preparex(`SELECT * FROM apps WHERE owner_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement SELECT * FROM apps WHERE owner_id = ..."+err.Error())
		panic(err)
	}

	// insert app:
	m.stmt.insertApp, err = m.DB.Handle.Preparex(`INSERT INTO apps 
		("owner_id", "name", "created") VALUES (?, ?, datetime("now"))`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement INSERT INTO apps..."+err.Error())
		panic(err)
	}

	// get version
	m.stmt.selectVersion, err = m.DB.Handle.Preparex(`SELECT * FROM app_versions WHERE app_id = ? AND version = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement SELECT * FROM app_versions..."+err.Error())
		panic(err)
	}

	// get versions for app
	m.stmt.selectAppVerions, err = m.DB.Handle.Preparex(`SELECT * FROM app_versions WHERE app_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement selectAppVerions"+err.Error())
		panic(err)
	}

	m.stmt.insertVersion, err = m.DB.Handle.Preparex(`INSERT INTO app_versions
		("app_id", "version", "schema", "location_key", created) VALUES (?, ?, ?, ?, datetime("now"))`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement insertVersion"+err.Error())
		panic(err)
	}

	m.stmt.deleteVersion, err = m.DB.Handle.Preparex(`DELETE FROM app_versions WHERE app_id = ? AND version = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement deleteVersion "+err.Error())
		panic(err)
	}
}

// Additional methods:
// - GetForUser
// - Get versions
// - Delete, DeleteVersion
// Some of the other methods from nodejs impl prob belong in trusted

// GetFromID gets the app using its unique ID on the system
func (m *AppModel) GetFromID(appID domain.AppID) (*domain.App, domain.Error) {
	var app domain.App

	err := m.stmt.selectID.QueryRowx(appID).StructScan(&app)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}
	// ^^ here we should differentiate between no rows returned and every other error

	return &app, nil
}

// GetForOwner returns array of application data for a given user
func (m *AppModel) GetForOwner(userID domain.UserID) ([]*domain.App, domain.Error) {
	ret := []*domain.App{}

	err := m.stmt.selectOwner.Select(&ret, userID)
	if err != nil {
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
		return nil, dserror.FromStandard(err)
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		return nil, dserror.FromStandard(err)
	}
	if lastID >= 0xFFFFFFFF {
		return nil, dserror.New(dserror.OutOFBounds, "Last Insert ID from DB greater than uint32")
	}

	appID := domain.AppID(lastID)

	app, dsErr := m.GetFromID(appID)
	if dsErr != nil {
		return nil, dsErr
	}

	return app, nil
}

// GetVersion returns the version for the app
func (m *AppModel) GetVersion(appID domain.AppID, version domain.Version) (*domain.AppVersion, domain.Error) {
	var appVersion domain.AppVersion

	err := m.stmt.selectVersion.QueryRowx(appID, version).StructScan(&appVersion)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, dserror.New(dserror.NoRowsInResultSet)
		}
		return nil, dserror.FromStandard(err)
	}

	return &appVersion, nil
}

// GetVersionsForApp returns an array of versions of code for that application
func (m *AppModel) GetVersionsForApp(appID domain.AppID) ([]*domain.AppVersion, domain.Error) {
	ret := []*domain.AppVersion{}

	err := m.stmt.selectAppVerions.Select(&ret, appID)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	return ret, nil
}

// CreateVersion adds a new version for an app in the DB
// has appid, version, location key, create date
// use appid and version as primary keys
// index on appid as well
func (m *AppModel) CreateVersion(appID domain.AppID, version domain.Version, schema int, locationKey string) (*domain.AppVersion, domain.Error) {
	// TODO: this should fail if version exists
	// .. but that should be caught by the route first.

	_, err := m.stmt.insertVersion.Exec(appID, version, schema, locationKey)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	appVersion, dsErr := m.GetVersion(appID, version)
	if dsErr != nil {
		return nil, dsErr
	}

	return appVersion, nil
}

// DeleteVersion removes a version from the DB
func (m *AppModel) DeleteVersion(appID domain.AppID, version domain.Version) domain.Error {
	_, err := m.stmt.deleteVersion.Exec(appID, version)
	if err != nil {
		return dserror.FromStandard(err)
	}
	return nil
}
