package appspacemodel

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// AppspaceModel represents the model for app spaces
type AppspaceModel struct {
	DB *domain.DB

	AsPausedEvent interface {
		Send(domain.AppspaceID, bool)
	}

	stmt struct {
		selectID         *sqlx.Stmt
		selectOwner      *sqlx.Stmt
		selectApp        *sqlx.Stmt
		selectAppVersion *sqlx.Stmt
		selectDomain     *sqlx.Stmt
		insert           *sqlx.Stmt
		pause            *sqlx.Stmt
		setVersion       *sqlx.Stmt
	}
}

// PrepareStatements for appspace model
func (m *AppspaceModel) PrepareStatements() {
	// Here is the place to get clever with statements if using multiple DBs.
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	//get from ID
	m.stmt.selectID = p.Prep(`SELECT * FROM appspaces WHERE appspace_id = ?`)

	//get from domain
	m.stmt.selectDomain = p.Prep(`SELECT * FROM appspaces WHERE domain_name = ?`)

	// get all for an owner
	m.stmt.selectOwner = p.Prep(`SELECT * FROM appspaces WHERE owner_id = ?`)

	// Do we have a db select for app / app_version, or do we just look everything up for owner?
	// -> advantage of app_id is that we might one day have non-owner apps
	m.stmt.selectApp = p.Prep(`SELECT * FROM appspaces WHERE app_id = ?`)

	m.stmt.selectAppVersion = p.Prep(`SELECT * FROM appspaces WHERE app_id = ? AND app_version = ?`)

	// insert appspace:
	m.stmt.insert = p.Prep(`INSERT INTO appspaces
		("owner_id", "dropid", "app_id", "app_version", domain_name, created, location_key) VALUES (?, ?, ?, ?, ?, datetime("now"), ?)`)

	// pause
	m.stmt.pause = p.Prep(`UPDATE appspaces SET paused = ? WHERE appspace_id = ?`)

	m.stmt.setVersion = p.Prep(`UPDATE appspaces SET app_version = ? WHERE appspace_id = ?`)
}

// GetFromID gets an AppSpace by its ID
// Q: does this return an error if not found? What kind of error
func (m *AppspaceModel) GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, error) {
	var appspace domain.Appspace

	err := m.stmt.selectID.QueryRowx(appspaceID).StructScan(&appspace)
	if err != nil {
		log := m.getLogger("GetFromID()").AppspaceID(appspaceID)
		if err != sql.ErrNoRows {
			log.Error(err)
		} else {
			log.Debug(err.Error())
		}
		return nil, err
	}

	return &appspace, nil
}

// GetFromDomain gets an AppSpace by looking up the domain
// It returns nil, nil if no matches found
func (m *AppspaceModel) GetFromDomain(dom string) (*domain.Appspace, error) {
	var appspace domain.Appspace

	err := m.stmt.selectDomain.QueryRowx(dom).StructScan(&appspace)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		m.getLogger("GetFromDomain(), domain: " + dom).Error(err)
		return nil, err
	}

	return &appspace, nil
}

// GetForOwner gets all appspaces for an owner
func (m *AppspaceModel) GetForOwner(userID domain.UserID) ([]*domain.Appspace, error) {
	ret := []*domain.Appspace{}

	err := m.stmt.selectOwner.Select(&ret, userID)
	if err != nil {
		m.getLogger("GetForOwner()").UserID(userID).Error(err)
		return nil, err
	}

	return ret, nil
}

// GetForApp gets all appspaces for a given app_id.
func (m *AppspaceModel) GetForApp(appID domain.AppID) ([]*domain.Appspace, error) {
	ret := []*domain.Appspace{}

	err := m.stmt.selectApp.Select(&ret, appID)
	if err != nil {
		m.getLogger("GetForApp()").AppID(appID).Error(err)
		return nil, err
	}

	return ret, nil
}

// GetForAppVersion gets all appspaces for a given app_id.
func (m *AppspaceModel) GetForAppVersion(appID domain.AppID, version domain.Version) ([]*domain.Appspace, error) {
	ret := []*domain.Appspace{}

	err := m.stmt.selectAppVersion.Select(&ret, appID, version)
	if err != nil {
		m.getLogger("GetForAppVersion()").AppID(appID).AppVersion(version).Error(err)
		return nil, err
	}

	return ret, nil // TODO: add test for this function
}

// Create adds an appspace to the database
func (m *AppspaceModel) Create(appspace domain.Appspace) (*domain.Appspace, error) {
	logger := m.getLogger("Create()").UserID(appspace.OwnerID).AppID(appspace.AppID).AppVersion(appspace.AppVersion).AddNote(fmt.Sprintf("domain:%v, locationkey:%v", appspace.DomainName, appspace.LocationKey))

	r, err := m.stmt.insert.Exec(appspace.OwnerID, appspace.DropID, appspace.AppID, appspace.AppVersion, appspace.DomainName, appspace.LocationKey)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: appspaces.domain" {
			return nil, errors.New("Domain not unique")
		}
		logger.AddNote("insert").Error(err)
		return nil, err
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		logger.AddNote("r.lastInsertId()").Error(err)
		return nil, err
	}
	if lastID >= 0xFFFFFFFF {
		logger.Log("LastID greater than uint32")
		return nil, errors.New("Last Insert ID from DB greater than uint32")
	}

	appspaceID := domain.AppspaceID(lastID)

	storedAppspace, err := m.GetFromID(appspaceID)
	if err != nil {
		logger.AddNote("GetFromID()").Error(err)
		return nil, err // this indicates a severe internal problem, not "oh we coudln't find it"
	}

	return storedAppspace, nil
}

// Pause changes the paused status of the appspace
func (m *AppspaceModel) Pause(appspaceID domain.AppspaceID, pause bool) error {
	_, err := m.stmt.pause.Exec(pause, appspaceID)
	if err != nil {
		m.getLogger("Pause").Error(err)
		return err
	}

	m.AsPausedEvent.Send(appspaceID, pause)

	return nil
}

// SetVersion changes the active version of the application for tha tappspace
func (m *AppspaceModel) SetVersion(appspaceID domain.AppspaceID, version domain.Version) error {
	_, err := m.stmt.setVersion.Exec(version, appspaceID)
	if err != nil {
		m.getLogger("SetVersion").Error(err)
		return err
	}

	return nil
}

func (m *AppspaceModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
