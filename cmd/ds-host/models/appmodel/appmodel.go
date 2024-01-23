package appmodel

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/blang/semver/v4"
	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

type stmtPreparer interface {
	Preparex(query string) (*sqlx.Stmt, error)
}

// AppModel represents the model for app
type AppModel struct {
	DB *domain.DB

	AppUrlDataEvents interface {
		Send(ownerID domain.UserID, data domain.AppURLData)
	}

	stmt struct {
		selectID               *sqlx.Stmt
		selectOwner            *sqlx.Stmt
		selectAutoURlData      *sqlx.Stmt
		selectVersion          *sqlx.Stmt
		selectVersionForUI     *sqlx.Stmt
		selectVersionManifest  *sqlx.Stmt
		getAllVersions         *sqlx.Stmt
		selectAppVersions      *sqlx.Stmt
		selectAppVersionsForUI *sqlx.Stmt
		insertVersion          *sqlx.Stmt
		deleteVersion          *sqlx.Stmt
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

	m.stmt.selectAutoURlData = p.exec(`SELECT app_id FROM app_urls WHERE automatic = ? AND last_dt <= ?`)

	// get version
	// This one is intended for internal use (like running a sandbox)
	// should include entrypoint. app_name, created may not be necessary.
	m.stmt.selectVersion = p.exec(`SELECT 
		app_id, version,
		json_extract(manifest, '$.schema') AS schema,
		json_extract(manifest, '$.entrypoint') AS entrypoint,
		created, location_key
		FROM app_versions WHERE app_id = ? AND version = ?`)

	m.stmt.selectVersionForUI = p.exec(`SELECT
		app_id, version,
		ifnull(json_extract(manifest, '$.name'), "") AS name,
		ifnull(json_extract(manifest, '$.short-description'), "") AS short_desc,
		json_extract(manifest, '$.schema') AS schema,
		ifnull(json_extract(manifest, '$.accent-color'), "") AS color,
		ifnull(json_extract(manifest, '$.authors'), "") AS authors,
		ifnull(json_extract(manifest, '$.website'), "") AS website,
		ifnull(json_extract(manifest, '$.code'), "") AS code,
		ifnull(json_extract(manifest, '$.funding'), "") AS funding,
		ifnull(json_extract(manifest, '$.release-date'), "") AS release_date,
		ifnull(json_extract(manifest, '$.license'), "") AS license,
		created
		FROM app_versions WHERE app_id = ? AND version = ?`)

	m.stmt.selectAppVersionsForUI = p.exec(`SELECT
		app_id, version,
		ifnull(json_extract(manifest, '$.name'), "") AS name,
		ifnull(json_extract(manifest, '$.short-description'), "") AS short_desc,
		json_extract(manifest, '$.schema') AS schema,
		ifnull(json_extract(manifest, '$.accent-color'), "") AS color,
		ifnull(json_extract(manifest, '$.authors'), "") AS authors,
		ifnull(json_extract(manifest, '$.website'), "") AS website,
		ifnull(json_extract(manifest, '$.code'), "") AS code,
		ifnull(json_extract(manifest, '$.funding'), "") AS funding,
		ifnull(json_extract(manifest, '$.release-date'), "") AS release_date,
		ifnull(json_extract(manifest, '$.license'), "") AS license,
		created
		FROM app_versions WHERE app_id = ?`)

	m.stmt.selectVersionManifest = p.exec(`SELECT manifest FROM app_versions WHERE app_id = ? AND version = ?`)

	m.stmt.getAllVersions = p.exec(`SELECT version FROM app_versions WHERE app_id = ?`)

	// get versions for app
	m.stmt.selectAppVersions = p.exec(`SELECT
		app_id, version,
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

// Create adds an app to the database with no URL data
// For use with manually uploaded apps.
func (m *AppModel) Create(ownerID domain.UserID) (domain.AppID, error) {
	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		m.getLogger("Create(), Begin()").Error(err)
		return domain.AppID(0), err
	}
	defer tx.Rollback()

	appID, err := m.create(ownerID, tx)
	if err != nil {
		return domain.AppID(0), err
	}

	err = tx.Commit()
	if err != nil {
		m.getLogger("Create(), tx.Commit()").Error(err)
		return domain.AppID(0), err
	}

	return appID, nil
}

// CreateFromURL creates the app and stores app url data
func (m *AppModel) CreateFromURL(ownerID domain.UserID, url string, auto bool, listingFetch domain.AppListingFetch) (domain.AppID, error) {
	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		m.getLogger("CreateFromURL(), Begin()").Error(err)
		return domain.AppID(0), err
	}
	defer tx.Rollback()

	appID, err := m.create(ownerID, tx)
	if err != nil {
		return domain.AppID(0), err
	}

	err = createAppUrlData(appID, url, auto, tx)
	if err != nil {
		m.getLogger("CreateFromURL(), createAppUrlData()").Error(err)
		return domain.AppID(0), err
	}

	err = setListing(appID, listingFetch, tx)
	if err != nil {
		m.getLogger("CreateFromURL(), setListing()").Error(err)
		return domain.AppID(0), err
	}

	err = setLast(appID, "ok", listingFetch.FetchDatetime, tx)
	if err != nil {
		m.getLogger("CreateFromURL(), setLast()").Error(err)
		return domain.AppID(0), err
	}

	err = tx.Commit()
	if err != nil {
		m.getLogger("Create(), tx.Commit()").Error(err)
		return domain.AppID(0), err
	}

	m.sendAppURLDataEvent(appID)

	return appID, nil
}

func (m *AppModel) create(ownerID domain.UserID, sp stmtPreparer) (domain.AppID, error) {
	stmt, err := sp.Preparex(`INSERT INTO apps 
		("owner_id", "created") VALUES (?, datetime("now"))`)
	if err != nil {
		m.getLogger("create(), tx.Preparex()").Error(err)
		return domain.AppID(0), err
	}
	defer stmt.Close()

	r, err := stmt.Exec(ownerID)
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

func createAppUrlData(appID domain.AppID, url string, auto bool, sp stmtPreparer) error {
	stmt, err := sp.Preparex(`INSERT INTO app_urls 
		("app_id", "url", "automatic", "new_url") VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(appID, url, auto, "")

	return err
}

// GetAppUrlData returns the app url data.
// If there is no app url data it returns domain.ErrNoRowsInResultSet
func (m *AppModel) GetAppUrlData(appID domain.AppID) (domain.AppURLData, error) {
	urlData, err := getAppUrlData(appID, m.DB.Handle)
	if err == domain.ErrNoRowsInResultSet {
		return domain.AppURLData{}, err
	} else if err != nil {
		m.getLogger("GetAppUrlData").AppID(appID).Error(err)
		return domain.AppURLData{}, err
	}
	return urlData, nil
}

func getAppUrlData(appID domain.AppID, sp stmtPreparer) (domain.AppURLData, error) {
	stmt, err := sp.Preparex(`SELECT app_id, url, automatic, 
		last_dt, last_result, 
		new_url, new_url_dt, 
		listing_dt, etag, latest_version 
		FROM app_urls WHERE app_id = ?`)
	if err != nil {
		return domain.AppURLData{}, err
	}
	defer stmt.Close()

	var urlData domain.AppURLData
	err = stmt.QueryRowx(appID).StructScan(&urlData)
	if err == sql.ErrNoRows {
		return domain.AppURLData{}, domain.ErrNoRowsInResultSet
	} else if err != nil {
		return domain.AppURLData{}, err
	}
	return urlData, nil
}

func getListing(appID domain.AppID, sp stmtPreparer) (domain.AppListing, error) {
	stmt, err := sp.Preparex(`SELECT listing FROM app_urls WHERE app_id = ?`)
	if err != nil {
		return domain.AppListing{}, err
	}
	defer stmt.Close()

	var listingStr string
	err = stmt.Get(&listingStr, appID)
	if err == sql.ErrNoRows {
		return domain.AppListing{}, domain.ErrNoRowsInResultSet
	} else if err != nil {
		return domain.AppListing{}, err
	}

	var listing domain.AppListing
	err = json.Unmarshal([]byte(listingStr), &listing)
	if err != nil {
		return domain.AppListing{}, err
	}
	return listing, nil
}

// GetAppUrlListing returns the listing along with the URL data
// If app is not from a URL it returns the error domain.ErrNoRowsInResultSet
func (m *AppModel) GetAppUrlListing(appID domain.AppID) (domain.AppListing, domain.AppURLData, error) {
	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		m.getLogger("GetAppUrlListing(), Beginx()").Error(err)
		return domain.AppListing{}, domain.AppURLData{}, err
	}
	defer tx.Commit()

	urlData, err := getAppUrlData(appID, tx)
	if err == domain.ErrNoRowsInResultSet {
		return domain.AppListing{}, domain.AppURLData{}, err
	} else if err != nil {
		m.getLogger("GetAppUrlListing(), getAppUrlData()").AppID(appID).Error(err)
		return domain.AppListing{}, domain.AppURLData{}, err
	}

	listing, err := getListing(appID, tx)
	if err != nil {
		m.getLogger("GetAppUrlListing(), getListing()").AppID(appID).Error(err)
		return domain.AppListing{}, domain.AppURLData{}, err
	}

	return listing, urlData, nil
}

// GetAutoUrlDataByLastDt returns the app IDs that have automatic
// refresh enabled and haven't been refreshed since last
func (m *AppModel) GetAutoUrlDataByLastDt(last time.Time) ([]domain.AppID, error) {
	ret := []domain.AppID{}
	err := m.stmt.selectAutoURlData.Select(&ret, true, last)
	if err != nil {
		m.getLogger("GetAutoUrlDataByLastDt()").Error(err)
		return nil, err
	}
	return ret, nil
}

// UpdateAutomatic to set the value of the automatic column in app url data
func (m *AppModel) UpdateAutomatic(appID domain.AppID, auto bool) error {
	stmt, err := m.DB.Handle.Preparex(`UPDATE app_urls SET automatic = ? WHERE app_id = ?`)
	if err != nil {
		m.getLogger("UpdateAutomatic(), Preparex()").AppID(appID).Error(err)
		return err
	}
	_, err = stmt.Exec(auto, appID)
	if err != nil {
		m.getLogger("UpdateAutomatic(), Preparex()").AppID(appID).Error(err)
		return err
	}
	m.sendAppURLDataEvent(appID)
	return nil
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

	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		m.getLogger("Delete(), Beginx()").AppID(appID).Error(err)
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`DELETE FROM app_urls WHERE app_id = ?`)
	if err != nil {
		m.getLogger("Delete(), Preparex() for app_urls").AppID(appID).Error(err)
		return err
	}
	_, err = stmt.Exec(appID)
	if err != nil {
		m.getLogger("Delete(), Exec() for app_urls").AppID(appID).Error(err)
		return err
	}

	stmt, err = tx.Preparex(`DELETE FROM apps WHERE app_id = ?`)
	if err != nil {
		m.getLogger("Delete(), Preparex() for apps").AppID(appID).Error(err)
		return err
	}
	_, err = stmt.Exec(appID)
	if err != nil {
		m.getLogger("Delete(), Exec() for apps").AppID(appID).Error(err)
		return err
	}

	tx.Commit()
	return nil
}

// SetLastFetch time of the last listing fetch
func (m *AppModel) SetLastFetch(appID domain.AppID, lastDt time.Time, lastResult string) error {
	err := setLast(appID, lastResult, lastDt, m.DB.Handle)
	if err != nil {
		m.getLogger("SetLastFetch(), setLast()").AppID(appID).Error(err)
		return err
	}
	m.sendAppURLDataEvent(appID)
	return nil
}

// SetListing and the last fetch data and reset the new url data.
func (m *AppModel) SetListing(appID domain.AppID, listingFetch domain.AppListingFetch) error {
	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		m.getLogger("SetListing(), Beginx()").Error(err)
		return err
	}
	defer tx.Rollback()

	// set last fetch data
	err = setLast(appID, "ok", listingFetch.FetchDatetime, tx)
	if err != nil {
		m.getLogger("SetListing(), setLast()").AppID(appID).Error(err)
		return err
	}

	// reset new URL data since we got a listing
	err = setNewUrl(appID, "", nulltypes.NewTime(time.Time{}, false), tx)
	if err != nil {
		m.getLogger("SetListing(), setNewUrl()").AppID(appID).Error(err)
		return err
	}

	// set listing
	err = setListing(appID, listingFetch, tx)
	if err != nil {
		m.getLogger("SetListing(), setListing()").AppID(appID).Error(err)
		return err
	}

	tx.Commit()
	m.sendAppURLDataEvent(appID)
	return nil
}

// SetNewUrl sets the new url that the remote site says future requests should go
func (m *AppModel) SetNewUrl(appID domain.AppID, url string, dt time.Time) error {
	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		m.getLogger("SetNewUrl(), Beginx()").Error(err)
		return err
	}
	defer tx.Rollback()

	err = setNewUrl(appID, url, nulltypes.NewTime(dt, true), tx)
	if err != nil {
		m.getLogger("SetNewUrl(), setNewUrl()").AppID(appID).Error(err)
		return err
	}

	err = setLast(appID, "ok", dt, tx)
	if err != nil {
		m.getLogger("SetNewUrl(), setLast()").AppID(appID).Error(err)
		return err
	}

	tx.Commit()
	m.sendAppURLDataEvent(appID)
	return nil
}

// UpdateURL of app listing and set the listing.
func (m *AppModel) UpdateURL(appID domain.AppID, url string, listingFetch domain.AppListingFetch) error {
	tx, err := m.DB.Handle.Beginx()
	if err != nil {
		m.getLogger("SetListing(), Beginx()").Error(err)
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`UPDATE app_urls SET url = ? WHERE app_id = ?`)
	if err != nil {
		m.getLogger("UpdateURL(), Preparex()").AppID(appID).Error(err)
		return err
	}
	_, err = stmt.Exec(url, appID)
	if err != nil {
		m.getLogger("UpdateURL(), Exec()").AppID(appID).Error(err)
		return err
	}

	// set last fetch data
	err = setLast(appID, "ok", listingFetch.FetchDatetime, tx)
	if err != nil {
		m.getLogger("UpdateURL(), setLast()").AppID(appID).Error(err)
		return err
	}

	// reset new URL data since we got a listing
	err = setNewUrl(appID, "", nulltypes.NewTime(time.Time{}, false), tx)
	if err != nil {
		m.getLogger("UpdateURL(), setNewUrl()").AppID(appID).Error(err)
		return err
	}

	// set listing
	err = setListing(appID, listingFetch, tx)
	if err != nil {
		m.getLogger("UpdateURL(), setListing()").AppID(appID).Error(err)
		return err
	}

	tx.Commit()
	m.sendAppURLDataEvent(appID)
	return nil
}

func setLast(appID domain.AppID, result string, dt time.Time, sp stmtPreparer) error {
	stmt, err := sp.Preparex(`UPDATE app_urls SET
		last_dt = ?, last_result = ?
		WHERE app_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(dt, result, appID)

	return err
}

func setNewUrl(appID domain.AppID, new_url string, dt nulltypes.NullTime, sp stmtPreparer) error {
	stmt, err := sp.Preparex(`UPDATE app_urls SET
		new_url = ?, new_url_dt = ?
		WHERE app_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(new_url, dt, appID)

	return err
}

func setListing(appID domain.AppID, l domain.AppListingFetch, sp stmtPreparer) error {
	listingBytes, err := json.Marshal(l.Listing)
	if err != nil {
		return err
	}

	stmt, err := sp.Preparex(`UPDATE app_urls SET
		listing =  json(?), listing_dt = ?, etag = ?, latest_version = ?
		WHERE app_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(listingBytes, l.ListingDatetime, l.Etag, l.LatestVersion, appID)

	return err
}

func (m *AppModel) sendAppURLDataEvent(appID domain.AppID) {
	if m.AppUrlDataEvents == nil {
		return
	}
	app, err := m.GetFromID(appID)
	if err != nil {
		return
	}
	urlData, err := m.GetAppUrlData(appID)
	if err != nil {
		return
	}
	m.AppUrlDataEvents.Send(app.OwnerID, urlData)
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

type AppVersionUIDB struct {
	domain.AppVersionUI
	AuthorsDB string `db:"authors"`
}

func (m *AppModel) GetVersionForUI(appID domain.AppID, version domain.Version) (domain.AppVersionUI, error) {
	var appVersion AppVersionUIDB
	err := m.stmt.selectVersionForUI.QueryRowx(appID, version).StructScan(&appVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.AppVersionUI{}, domain.ErrNoRowsInResultSet
		}
		m.getLogger("GetVersionForUI()").AppID(appID).AppVersion(version).Error(err)
		return domain.AppVersionUI{}, err
	}
	ret, err := makeAppVersionUI(appVersion)
	if err != nil {
		m.getLogger("GetVersionForUI() makeAppVersionUI()").AppID(appID).AppVersion(version).Error(err)
		return domain.AppVersionUI{}, err
	}
	return ret, nil
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

	err := m.stmt.selectAppVersions.Select(&ret, appID)
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

// GetUIOVersionsForApp returns an array of versions of code for that application
func (m *AppModel) GetVersionsForUIForApp(appID domain.AppID) ([]domain.AppVersionUI, error) {
	rows := []AppVersionUIDB{}
	err := m.stmt.selectAppVersionsForUI.Select(&rows, appID)
	if err != nil {
		m.getLogger("GetVersionsForUIForApp()").AppID(appID).Error(err)
		return nil, err
	}

	sort.Slice(rows, func(i, j int) bool {
		iSemver, err := semver.New(string(rows[i].Version)) // this is not efficient, but ok for now
		if err != nil {
			return false
		}
		jSemver, err := semver.New(string(rows[j].Version))
		if err != nil {
			return false
		}
		return iSemver.Compare(*jSemver) == -1
	})

	ret := make([]domain.AppVersionUI, len(rows))
	for i, r := range rows {
		ui, err := makeAppVersionUI(r)
		if err != nil {
			m.getLogger("GetVersionForUI() makeAppVersionUI()").AppID(appID).AppVersion(r.Version).Error(err)
			return nil, err
		}
		ret[i] = ui
	}

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

type parsedVersion struct {
	dom    domain.Version
	parsed semver.Version
}

// GetCurrentVersion returns the current version of the app.
// If there are no versions it returns domain.ErrNotRowsInResultSet
func (m *AppModel) GetCurrentVersion(appID domain.AppID) (domain.Version, error) {
	pVersions, err := m.getParsedVersions(appID)
	if err != nil {
		return domain.Version(""), err
	}
	if len(pVersions) == 0 {
		return domain.Version(""), domain.ErrNoRowsInResultSet
	}

	// here we should iterate over the array backwards and ignore pre-release versions

	return pVersions[len(pVersions)-1].dom, nil
}

// getParsedVersions returns a sorted array of versions in both parsed
// and original form for a given app id.
func (m *AppModel) getParsedVersions(appID domain.AppID) ([]parsedVersion, error) {
	versions := make([]domain.Version, 0)
	err := m.stmt.getAllVersions.Select(&versions, appID)
	if err != nil {
		m.getLogger("getAllVersions() Select").AppID(appID).Error(err)
		return nil, err
	}
	pVersions := make([]parsedVersion, len(versions))
	for i, v := range versions {
		p, err := semver.Parse(string(v))
		if err != nil {
			m.getLogger("getAllVersions() Parse").AppID(appID).Error(err)
			return nil, err
		}
		pVersions[i] = parsedVersion{v, p}
	}
	sortVersions(pVersions)
	return pVersions, nil
}

// sortVersions sorts the array such that the earliest version is at index 0
// and latest / highest version number is at the end of the array
func sortVersions(pVersions []parsedVersion) {
	sort.Slice(pVersions, func(i, j int) bool {
		return pVersions[i].parsed.Compare(pVersions[j].parsed) == -1
	})
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

func makeAppVersionUI(row AppVersionUIDB) (domain.AppVersionUI, error) {
	authors := make([]domain.ManifestAuthor, 0)
	if row.AuthorsDB != "" {
		err := json.Unmarshal([]byte(row.AuthorsDB), &authors)
		if err != nil {
			return domain.AppVersionUI{}, err
		}
	}
	return domain.AppVersionUI{
		AppID:            row.AppID,
		Name:             row.Name,
		Version:          row.Version,
		Schema:           row.Schema,
		Created:          row.Created,
		ShortDescription: row.ShortDescription,
		AccentColor:      row.AccentColor,
		Authors:          authors,
		Website:          row.Website,
		Code:             row.Code,
		Funding:          row.Funding,
		ReleaseDate:      row.ReleaseDate,
		License:          row.License,
	}, nil
}

func (m *AppModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
