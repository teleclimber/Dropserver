package appspacemodel

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// AppspaceModel represents the model for app spaces
type AppspaceModel struct {
	DB *domain.DB

	stmt struct {
		selectID        *sqlx.Stmt
		selectOwner     *sqlx.Stmt
		selectApp       *sqlx.Stmt
		selectSubdomain *sqlx.Stmt
		insert          *sqlx.Stmt
		pause           *sqlx.Stmt
		setVersion      *sqlx.Stmt
	}
}

// PrepareStatements for appspace model
func (m *AppspaceModel) PrepareStatements() {
	// Here is the place to get clever with statements if using multiple DBs.
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	//get from ID
	m.stmt.selectID = p.Prep(`SELECT * FROM appspaces WHERE appspace_id = ?`)

	//get from subdomain
	m.stmt.selectSubdomain = p.Prep(`SELECT * FROM appspaces WHERE subdomain = ?`)

	// get all for an owner
	m.stmt.selectOwner = p.Prep(`SELECT * FROM appspaces WHERE owner_id = ?`)

	// Do we have a db select for app / app_version, or do we just look everything up for owner?
	// -> advantage of app_id is that we might one day have non-owner apps
	m.stmt.selectApp = p.Prep(`SELECT * FROM appspaces WHERE app_id = ?`)

	// insert appspace:
	m.stmt.insert = p.Prep(`INSERT INTO appspaces
		("owner_id", "app_id", "app_version", subdomain, created, location_key) VALUES (?, ?, ?, ?, datetime("now"), ?)`)

	// pause
	m.stmt.pause = p.Prep(`UPDATE appspaces SET paused = ? WHERE appspace_id = ?`)

	m.stmt.setVersion = p.Prep(`UPDATE appspaces SET app_version = ? WHERE appspace_id = ?`)
}

// GetFromID gets an AppSpace by its ID
func (m *AppspaceModel) GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, domain.Error) {
	var appspace domain.Appspace

	err := m.stmt.selectID.QueryRowx(appspaceID).StructScan(&appspace)
	if err != nil {
		m.getLogger("GetFromID()").AppspaceID(appspaceID).Error(err)
		return nil, dserror.FromStandard(err)
	}

	return &appspace, nil
}

// GetFromSubdomain gets an AppSpace by looking up the subdomain
func (m *AppspaceModel) GetFromSubdomain(subdomain string) (*domain.Appspace, domain.Error) {
	var appspace domain.Appspace

	err := m.stmt.selectSubdomain.QueryRowx(subdomain).StructScan(&appspace)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dserror.New(dserror.NoRowsInResultSet)
		}
		m.getLogger("GetFromSubdomain(), subdomain: " + subdomain).Error(err)
		return nil, dserror.FromStandard(err)
	}

	return &appspace, nil
}

// GetForOwner gets all appspaces for an owner
func (m *AppspaceModel) GetForOwner(userID domain.UserID) ([]*domain.Appspace, domain.Error) {
	ret := []*domain.Appspace{}

	err := m.stmt.selectOwner.Select(&ret, userID)
	if err != nil {
		m.getLogger("GetForOwner()").UserID(userID).Error(err)
		return nil, dserror.FromStandard(err)
	}

	return ret, nil
}

// GetForApp gets all appspaces for a given app_id.
func (m *AppspaceModel) GetForApp(appID domain.AppID) ([]*domain.Appspace, domain.Error) {
	ret := []*domain.Appspace{}

	err := m.stmt.selectApp.Select(&ret, appID)
	if err != nil {
		m.getLogger("GetForOwner()").AppID(appID).Error(err)
		return nil, dserror.FromStandard(err)
	}

	return ret, nil
}

// Create adds an appspace to the database
func (m *AppspaceModel) Create(ownerID domain.UserID, appID domain.AppID, version domain.Version, subdomain string, locationKey string) (*domain.Appspace, domain.Error) {
	logger := m.getLogger("GetForOwner()").UserID(ownerID).AppID(appID).AppVersion(version).AddNote(fmt.Sprintf("subdomain:%v, locationkey:%v", subdomain, locationKey))

	r, err := m.stmt.insert.Exec(ownerID, appID, version, subdomain, locationKey)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: appspaces.subdomain" {
			return nil, dserror.New(dserror.DomainNotUnique)
		}
		logger.AddNote("insert").Error(err)
		return nil, dserror.FromStandard(err)
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		logger.AddNote("r.lastInsertId()").Error(err)
		return nil, dserror.FromStandard(err)
	}
	if lastID >= 0xFFFFFFFF {
		logger.Log("LastID greater than uint32")
		return nil, dserror.New(dserror.OutOFBounds, "Last Insert ID from DB greater than uint32")
	}

	appspaceID := domain.AppspaceID(lastID)

	appspace, dsErr := m.GetFromID(appspaceID)
	if dsErr != nil {
		logger.AddNote("GetFromID()").Error(dsErr.ToStandard())
		return nil, dsErr // this indicates a severe internal problem, not "oh we coudln't find it"
	}

	return appspace, nil
}

// Pause changes the paused status of the appspace
func (m *AppspaceModel) Pause(appspaceID domain.AppspaceID, pause bool) domain.Error {
	_, err := m.stmt.pause.Exec(pause, appspaceID)
	if err != nil {
		m.getLogger("Pause").Error(err)
		return dserror.FromStandard(err)
	}

	return nil
}

// SetVersion changes the active version of the application for tha tappspace
func (m *AppspaceModel) SetVersion(appspaceID domain.AppspaceID, version domain.Version) domain.Error {
	_, err := m.stmt.setVersion.Exec(version, appspaceID)
	if err != nil {
		m.getLogger("SetVersion").Error(err)
		return dserror.FromStandard(err)
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
