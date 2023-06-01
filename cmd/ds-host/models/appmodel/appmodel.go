package appmodel

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/blang/semver/v4"
	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
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
		selectID              *sqlx.Stmt
		selectOwner           *sqlx.Stmt
		insertApp             *sqlx.Stmt
		deleteApp             *sqlx.Stmt
		selectVersion         *sqlx.Stmt
		selectVersionManifest *sqlx.Stmt
		selectAppVerions      *sqlx.Stmt
		insertVersion         *sqlx.Stmt
		deleteVersion         *sqlx.Stmt
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
		("owner_id", "created") VALUES (?, datetime("now"))`)

	// delete app
	m.stmt.deleteApp = p.exec(`DELETE FROM apps WHERE app_id = ?`)

	// get version
	m.stmt.selectVersion = p.exec(`SELECT 
		app_id, version,
		json_extract(manifest, '$.name') AS app_name,
		json_extract(manifest, '$.schema') AS schema,
		created, location_key
		FROM app_versions WHERE app_id = ? AND version = ?`)

	m.stmt.selectVersionManifest = p.exec(`SELECT manifest FROM app_versions WHERE app_id = ? AND version = ?`)

	// get versions for app
	m.stmt.selectAppVerions = p.exec(`SELECT
		app_id, version,
		json_extract(manifest, '$.name') AS app_name,
		json_extract(manifest, '$.schema') AS schema,
		created, location_key
		FROM app_versions WHERE app_id = ?`)

	m.stmt.insertVersion = p.exec(`INSERT INTO app_versions
		(app_id, version, location_key, manifest, created) VALUES (?, ?, ?, json(?), datetime("now"))`)

	m.stmt.deleteVersion = p.exec(`DELETE FROM app_versions WHERE app_id = ? AND version = ?`)

	if p.err != nil {
		panic(p.err)
	}
}

// GetFromID gets the app using its unique ID on the system
// It returns an error if ID is not found
func (m *AppModel) GetFromID(appID domain.AppID) (domain.App, error) {
	var app domain.App
	err := m.stmt.selectID.QueryRowx(appID).StructScan(&app)
	if err != nil {
		log := m.getLogger("GetFromID()").AppID(appID)
		if err == sql.ErrNoRows {
			log.Debug(err.Error())
			return app, domain.ErrNoRowsInResultSet
		} else {
			log.Error(err)
			return app, err
		}
	}
	return app, nil
}

// GetForOwner returns array of application data for a given user
func (m *AppModel) GetForOwner(userID domain.UserID) ([]*domain.App, error) {
	ret := []*domain.App{}

	err := m.stmt.selectOwner.Select(&ret, userID)
	if err != nil {
		m.getLogger("GetForOwner()").UserID(userID).Error(err)
		return nil, err
	}

	return ret, nil
}

// Create adds an app to the database
// Other arguments: source URL, auto-update mode (if that applies to app)
func (m *AppModel) Create(ownerID domain.UserID) (domain.AppID, error) {
	// location key isn't here. It's in a version.
	// do we check name and locationKey for epty string or excess length?
	// -> probably, yes. Or where should that actually happen?

	r, err := m.stmt.insertApp.Exec(ownerID)
	if err != nil {
		m.getLogger("Create(), insertApp.Exec()").UserID(ownerID).Error(err)
		return domain.AppID(0), err
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		m.getLogger("Create(), r.LastInsertId()").UserID(ownerID).Error(err)
		return domain.AppID(0), err
	}
	if lastID >= 0xFFFFFFFF {
		m.getLogger("Create()").Log(fmt.Sprintf("Last insert ID out of bounds: %v", lastID))
		return domain.AppID(0), errors.New("last Insert ID from DB greater than uint32")
	}

	return domain.AppID(lastID), nil
}

// Delete the app from the DB row.
// It fails if there are versions of the app in the DB
func (m *AppModel) Delete(appID domain.AppID) error {
	versions, err := m.GetVersionsForApp(appID)
	if err != nil {
		return err
	}
	if len(versions) != 0 {
		err = errors.New("found app versions in db while trying to delete app")
		m.getLogger("Delete").AppID(appID).Error(err)
		return err
	}

	_, err = m.stmt.deleteApp.Exec(appID)
	if err != nil {
		m.getLogger("Delete(), deleteApp.Exec()").AppID(appID).Error(err)
		return err
	}
	return nil
}

// GetVersion returns the version for the app
func (m *AppModel) GetVersion(appID domain.AppID, version domain.Version) (domain.AppVersion, error) {
	var appVersion domain.AppVersion
	err := m.stmt.selectVersion.QueryRowx(appID, version).StructScan(&appVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return appVersion, domain.ErrNoRowsInResultSet
		}
		m.getLogger("GetVersion()").AppID(appID).AppVersion(version).Error(err)
		return appVersion, err
	}
	return appVersion, nil
}

func (m *AppModel) GetVersionManifestJSON(appID domain.AppID, version domain.Version) (string, error) {
	var manifest string
	err := m.stmt.selectVersionManifest.Get(&manifest, appID, version)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", domain.ErrNoRowsInResultSet
		}
		m.getLogger("GetVersionManifestJSON()").AppID(appID).AppVersion(version).Error(err)
		return "", err
	}
	return manifest, nil
}

func (m *AppModel) GetVersionManifest(appID domain.AppID, version domain.Version) (domain.AppVersionManifest, error) {
	manifestJSON, err := m.GetVersionManifestJSON(appID, version)
	if err != nil {
		return domain.AppVersionManifest{}, err
	}
	var manifest domain.AppVersionManifest
	err = json.Unmarshal([]byte(manifestJSON), &manifest)
	if err != nil {
		m.getLogger("GetVersionManifest() Unmarshal").AppID(appID).AppVersion(version).Error(err)
		return domain.AppVersionManifest{}, err
	}
	return manifest, nil
}

// GetVersionsForApp returns an array of versions of code for that application
func (m *AppModel) GetVersionsForApp(appID domain.AppID) ([]*domain.AppVersion, error) {
	ret := []*domain.AppVersion{}

	err := m.stmt.selectAppVerions.Select(&ret, appID)
	if err != nil {
		m.getLogger("GetVersionsForApp()").AppID(appID).Error(err)
		return nil, err
	}

	sort.Slice(ret, func(i, j int) bool {
		iSemver, err := semver.New(string(ret[i].Version)) // this is not efficient, but ok for now
		if err != nil {
			return false
		}
		jSemver, err := semver.New(string(ret[j].Version))
		if err != nil {
			return false
		}
		return iSemver.Compare(*jSemver) == -1
	})

	return ret, nil
}

// CreateVersion adds a new version for an app in the DB
// has appid, version, location key, create date
// use appid and version as primary keys
// index on appid as well
func (m *AppModel) CreateVersion(appID domain.AppID, locationKey string, manifest domain.AppVersionManifest) (domain.AppVersion, error) {
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		m.getLogger("CreateVersion(), json.Marshal").AppID(appID).AppVersion(manifest.Version).Error(err)
		return domain.AppVersion{}, err
	}
	_, err = m.stmt.insertVersion.Exec(appID, manifest.Version, locationKey, manifestBytes)
	if err != nil {
		m.getLogger("CreateVersion(), insertVersion").AppID(appID).AppVersion(manifest.Version).Error(err)
		return domain.AppVersion{}, err
	}
	appVersion, err := m.GetVersion(appID, manifest.Version)
	if err != nil {
		m.getLogger("CreateVersion(), GetVersion").AppID(appID).AppVersion(manifest.Version).Error(err)
		return domain.AppVersion{}, err
	}
	return appVersion, nil
}

// DeleteVersion removes a version from the DB
func (m *AppModel) DeleteVersion(appID domain.AppID, version domain.Version) error {
	_, err := m.stmt.deleteVersion.Exec(appID, version)
	if err != nil {
		m.getLogger("DeleteVersion()").AppID(appID).AppVersion(version).Error(err)
		return err
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
