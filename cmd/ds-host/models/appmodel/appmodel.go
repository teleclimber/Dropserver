package appmodel

import (
	"fmt"

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
		insertApp *sqlx.Stmt
		selectID *sqlx.Stmt
	}
}

// PrepareStatements prepares the statements 
func (m *AppModel) PrepareStatements() {
	// Here is the place to get clever with statements if using multiple DBs.

	var err error

	// insert app:
	m.stmt.insertApp, err = m.DB.Handle.Preparex(`INSERT INTO apps ("name", "location_key") VALUES (?, ?)`)
	if err != nil {
		fmt.Println("Error preparing statement INSERT INTO apps...", err)
		panic(err)
	}

	//get from ID
	m.stmt.selectID, err = m.DB.Handle.Preparex(`SELECT rowid, * FROM apps WHERE rowid = ?`)
	if err != nil {
		fmt.Println("Error preparing statement SELECT * FROM apps...", err)
		panic(err)
	}
}

// GetForUser

// GetFromID gets the app using its unique ID on the system
func (m *AppModel) GetFromID(appID int64) (*domain.App, domain.Error) {
	var app domain.App

	err := m.stmt.selectID.QueryRowx(appID).StructScan(&app)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	return &app, nil
}


// Create adds an app to the database
// This should return an unique ID, right?
// Other arguments: locationKey, owner
// Should we have CreateArgs type struct to guarantee proper data passing?
func (m *AppModel) Create(name string, locationKey string) (*domain.App, domain.Error) {
	// do we check name and locationKey for epty string or excess length?
	// -> probably, yes. Or where should that actually happen?

	r, err := m.stmt.insertApp.Exec(name, locationKey)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		return nil, dserror.FromStandard(err)
	}

	app, dsErr := m.GetFromID(lastID)
	if dsErr != nil {
		return nil, dsErr
	}
	
	return app, nil
	
}

