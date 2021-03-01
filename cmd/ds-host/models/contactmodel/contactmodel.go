package contactmodel

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/sqlxprepper"
)

// API:
// - createContact
// - updateContact
// - deleteContact
// - getContact
// ...and appspace level stuff
// - setAppspaceContact (appspace_id, proxy_id, contact_id)
// - deleteAppspaceContact (appspace_id, contact_id)
// - getAppspaceContact (appspace_id, contact_id) -> proxy_id //what's the use-case here?
// - getByProxy (appspace_id, proxy_id) => contact_id and maybe other stuff (blocked, metadata like created/updated dt)
// ...combined queries
// - appspaceContacts(appspcae_id) (get all contacts that are in appspce)
// - contactAppspaces(contact_id) (get all appspaces that a contact is part of)

// ContactModel stores user contacts
// and their participation in appspaces
type ContactModel struct {
	DB *domain.DB
	// need config to select db type?

	stmt struct {
		createContact         *sqlx.Stmt
		updateContact         *sqlx.Stmt
		deleteContact         *sqlx.Stmt
		getContact            *sqlx.Stmt
		getUserContacts       *sqlx.Stmt
		insertAppspaceContact *sqlx.Stmt
		deleteAppspaceContact *sqlx.Stmt
		getAppspaceContact    *sqlx.Stmt
		getByProxy            *sqlx.Stmt
		getContactAppspaces   *sqlx.Stmt
		getAppspaceContacts   *sqlx.Stmt
	}
}

// PrepareStatements for appspace model
func (m *ContactModel) PrepareStatements() {
	// Here is the place to get clever with statements if using multiple DBs.
	p := sqlxprepper.NewPrepper(m.DB.Handle)

	// User contacts:
	m.stmt.createContact = p.Prep(`INSERT INTO contacts 
		(user_id, name, display_name, created) 
		VALUES (?, ?, ?, datetime("now"))`)

	m.stmt.updateContact = p.Prep(`UPDATE contacts SET 
		name = ?, display_name = ? 
		WHERE user_id = ? AND contact_id = ?`)

	m.stmt.deleteContact = p.Prep(`DELETE FROM contacts WHERE user_id = ? AND contact_id = ?`)

	m.stmt.getContact = p.Prep(`SELECT * FROM contacts WHERE contact_id = ?`)

	m.stmt.getUserContacts = p.Prep(`SELECT * FROM contacts WHERE user_id = ?`)

	// Appspace contacts
	m.stmt.insertAppspaceContact = p.Prep(`INSERT INTO appspace_contacts (appspace_id, contact_id, proxy_id) VALUES (?, ?, ?)`)

	m.stmt.deleteAppspaceContact = p.Prep(`DELETE FROM appspace_contacts WHERE appspace_id = ? AND contact_id = ?`)

	m.stmt.getAppspaceContact = p.Prep(`SELECT proxy_id FROM appspace_contacts WHERE appspace_id = ? AND contact_id = ?`)

	m.stmt.getByProxy = p.Prep(`SELECT contact_id FROM appspace_contacts WHERE appspace_id = ? AND proxy_id = ?`)

	m.stmt.getContactAppspaces = p.Prep(`SELECT * FROM appspace_contacts WHERE contact_id = ?`)

	m.stmt.getAppspaceContacts = p.Prep(`SELECT * FROM appspace_contacts WHERE appspace_id = ?`)
}

// Create a new contact
func (m *ContactModel) Create(userID domain.UserID, name string, displayName string) (domain.Contact, error) {
	logger := m.getLogger("Create()").UserID(userID)

	r, err := m.stmt.createContact.Exec(userID, name, displayName)
	if err != nil {
		logger.AddNote("insert").Error(err)
		return domain.Contact{}, err
	}

	lastID, err := r.LastInsertId()
	if err != nil {
		logger.AddNote("r.lastInsertId()").Error(err)
		return domain.Contact{}, err
	}
	if lastID >= 0xFFFFFFFF {
		err = errors.New("Last Insert ID from DB greater than uint32")
		logger.Error(err)
		return domain.Contact{}, err
	}

	contactID := domain.ContactID(lastID)

	contact, err := m.Get(contactID)
	if err != nil {
		logger.AddNote("Get()").Error(err)
		return domain.Contact{}, err
	}

	return contact, nil
}

// Update updates
func (m *ContactModel) Update(userID domain.UserID, contactID domain.ContactID, name string, displayName string) error {
	logger := m.getLogger("Update()").UserID(userID)

	_, err := m.stmt.updateContact.Exec(name, displayName, userID, contactID)
	if err != nil {
		logger.AddNote("Exec()").Error(err)
		return err
	}
	return nil
}

// Delete deletes
func (m *ContactModel) Delete(userID domain.UserID, contactID domain.ContactID) error {
	logger := m.getLogger("Delete()").UserID(userID)

	_, err := m.stmt.deleteContact.Exec(userID, contactID)
	if err != nil {
		logger.AddNote("Exec()").Error(err)
		return err
	}
	return nil
}

// Get returns a single contact
func (m *ContactModel) Get(contactID domain.ContactID) (domain.Contact, error) {
	var contact domain.Contact

	err := m.stmt.getContact.QueryRowx(contactID).StructScan(&contact)
	if err != nil {
		if err != sql.ErrNoRows {
			m.getLogger("Get()").Error(err)
		}
		return domain.Contact{}, err
	}

	return contact, nil
}

// GetForUser retusn all contacts for teh given user id
func (m *ContactModel) GetForUser(userID domain.UserID) ([]domain.Contact, error) {
	ret := []domain.Contact{}

	err := m.stmt.getUserContacts.Select(&ret, userID)
	if err != nil {
		m.getLogger("GetAll()").UserID(userID).Error(err)
		return nil, err
	}

	return ret, nil
}

// InsertAppspaceContact inserts
func (m *ContactModel) InsertAppspaceContact(appspaceID domain.AppspaceID, contactID domain.ContactID, proxyID domain.ProxyID) error {
	_, err := m.stmt.insertAppspaceContact.Exec(appspaceID, contactID, proxyID)
	if err != nil {
		m.getLogger("InsertAppspaceContact()").AddNote("Exec").Error(err)
		return err
	}
	return nil
}

// DeleteAppspaceContact deletes
func (m *ContactModel) DeleteAppspaceContact(appspaceID domain.AppspaceID, contactID domain.ContactID) error {
	_, err := m.stmt.deleteAppspaceContact.Exec(appspaceID, contactID)
	if err != nil {
		m.getLogger("DeleteAppspaceContact()").AddNote("Exec").Error(err)
		return err
	}
	return nil
}

// GetContactProxy returns the proxy id for the appspace and contact id
func (m *ContactModel) GetContactProxy(appspaceID domain.AppspaceID, contactID domain.ContactID) (domain.ProxyID, error) {
	var p domain.ProxyID
	err := m.stmt.getAppspaceContact.Get(&p, appspaceID, contactID)
	return p, err
}

// GetByProxy returns the contact id given an appspace id and proxy id
func (m *ContactModel) GetByProxy(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.ContactID, error) {
	var c domain.ContactID
	err := m.stmt.getByProxy.Get(&c, appspaceID, proxyID)
	return c, err
}

// GetContactAppspaces returns all the appspaces that the passed contact is in
func (m *ContactModel) GetContactAppspaces(contactID domain.ContactID) ([]domain.AppspaceContact, error) {
	ret := []domain.AppspaceContact{}
	err := m.stmt.getContactAppspaces.Select(&ret, contactID)
	return ret, err
}

// GetAppspaceContacts returns all the contacts that are in a passed appspace
func (m *ContactModel) GetAppspaceContacts(appspaceID domain.AppspaceID) ([]domain.AppspaceContact, error) {
	ret := []domain.AppspaceContact{}
	err := m.stmt.getAppspaceContacts.Select(&ret, appspaceID)
	return ret, err
}

func (m *ContactModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("ContactModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
