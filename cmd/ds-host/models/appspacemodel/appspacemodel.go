package appspacemodel

import (
	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

// AppspaceModel represents the model for app spaces
type AppspaceModel struct {
	DB     *domain.DB
	Logger domain.LogCLientI

	stmt struct {
		selectID        *sqlx.Stmt
		selectOwner     *sqlx.Stmt
		selectApp       *sqlx.Stmt
		selectSubdomain *sqlx.Stmt
		insert          *sqlx.Stmt
		pause           *sqlx.Stmt
	}
}

// PrepareStatements for appspace model
func (m *AppspaceModel) PrepareStatements() {
	// Here is the place to get clever with statements if using multiple DBs.

	var err error

	//get from ID
	m.stmt.selectID, err = m.DB.Handle.Preparex(`SELECT * FROM appspaces WHERE appspace_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement SELECT * FROM appspaces..."+err.Error())
		panic(err)
	}

	//get from subdomain
	m.stmt.selectSubdomain, err = m.DB.Handle.Preparex(`SELECT * FROM appspaces WHERE subdomain = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement SELECT with subdomain..."+err.Error())
		panic(err)
	}

	// get all for an owner
	m.stmt.selectOwner, err = m.DB.Handle.Preparex(`SELECT * FROM appspaces WHERE owner_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement selectOwner "+err.Error())
		panic(err)
	}

	// Do we have a db select for app / app_version, or do we just look everything up for owner?
	// -> advantage of app_id is that we might one day have non-owner apps
	m.stmt.selectApp, err = m.DB.Handle.Preparex(`SELECT * FROM appspaces WHERE app_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement selectApp "+err.Error())
		panic(err)
	}

	// insert appspace:
	m.stmt.insert, err = m.DB.Handle.Preparex(`INSERT INTO appspaces
		("owner_id", "app_id", "app_version", subdomain, created) VALUES (?, ?, ?, ?, datetime("now"))`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement INSERT INTO appspaces..."+err.Error())
		panic(err)
	}

	// pause
	m.stmt.pause, err = m.DB.Handle.Preparex(`UPDATE appspaces SET paused = ? WHERE appspace_id = ?`)
	if err != nil {
		m.Logger.Log(domain.ERROR, nil, "Error preparing statement UPDATE appspaces SET paused = ?... "+err.Error())
		panic(err)
	}

}

// GetFromID gets an AppSpace by its ID
func (m *AppspaceModel) GetFromID(appspaceID domain.AppspaceID) (*domain.Appspace, domain.Error) {
	var appspace domain.Appspace

	err := m.stmt.selectID.QueryRowx(appspaceID).StructScan(&appspace)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}
	// ^^ here we should differentiate between no rows returned and every other error

	return &appspace, nil
}

// GetFromSubdomain gets an AppSpace by looking up the subdomain
func (m *AppspaceModel) GetFromSubdomain(subdomain string) (*domain.Appspace, domain.Error) {
	var appspace domain.Appspace

	err := m.stmt.selectSubdomain.QueryRowx(subdomain).StructScan(&appspace)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, dserror.New(dserror.NoRowsInResultSet)
		}
		return nil, dserror.FromStandard(err)
	}

	return &appspace, nil
}

// GetForOwner gets an AppSpace by looking up the subdomain
func (m *AppspaceModel) GetForOwner(userID domain.UserID) ([]*domain.Appspace, domain.Error) {
	ret := []*domain.Appspace{}

	err := m.stmt.selectOwner.Select(&ret, userID)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	return ret, nil
}

// GetForApp gets all appspaces for a given app_id.
func (m *AppspaceModel) GetForApp(appID domain.AppID) ([]*domain.Appspace, domain.Error) {
	ret := []*domain.Appspace{}

	err := m.stmt.selectApp.Select(&ret, appID)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	return ret, nil
}

// Create adds an appspace to the database
func (m *AppspaceModel) Create(ownerID domain.UserID, appID domain.AppID, version domain.Version, subdomain string) (*domain.Appspace, domain.Error) {
	r, err := m.stmt.insert.Exec(ownerID, appID, version, subdomain)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: appspaces.subdomain" {
			return nil, dserror.New(dserror.DomainNotUnique)
		}
		return nil, dserror.FromStandard(err)
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		return nil, dserror.FromStandard(err)
	}
	if lastID >= 0xFFFFFFFF {
		return nil, dserror.New(dserror.OutOFBounds, "Last Insert ID from DB greater than uint32")
	}

	appspaceID := domain.AppspaceID(lastID)

	appspace, dsErr := m.GetFromID(appspaceID)
	if dsErr != nil {
		return nil, dsErr // this indicates a severe internal problem, not "oh we coudln't find it"
	}

	return appspace, nil
}

// Pause changes the paused status of the appspace
func (m *AppspaceModel) Pause(appspaceID domain.AppspaceID, pause bool) domain.Error {
	_, err := m.stmt.pause.Exec(pause, appspaceID)
	if err != nil {
		return dserror.FromStandard(err)
	}

	return nil
}
