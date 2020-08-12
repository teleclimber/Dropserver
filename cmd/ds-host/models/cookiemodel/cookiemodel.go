package cookiemodel

// Stores cookies in DB.
// This is so that they can be retrieved by user ID or by appspace ID
// Allows mass logouts of user or appspace.
// A fast in-memory cache will alleviate performance problems.

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// CookieModel stores and retrives cookies for you
type CookieModel struct {
	DB *domain.DB
	// need config to select db type?

	stmt struct {
		selectCookieID *sqlx.Stmt
		create         *sqlx.Stmt
		refresh        *sqlx.Stmt
		delete         *sqlx.Stmt
	}
}

// copy-pasta the prepper helper until I find a good place for it
type prepper struct {
	handle *sqlx.DB
	err    error
}

func (p *prepper) prep(query string) *sqlx.Stmt {
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
func (p *prepper) checkErrors() {
	if p.err != nil {
		panic(p.err)
	}
}

// PrepareStatements pres the statements for this model
func (m *CookieModel) PrepareStatements() {
	p := prepper{handle: m.DB.Handle}

	m.stmt.selectCookieID = p.prep(`SELECT * FROM cookies WHERE cookie_id = ?`)
	m.stmt.create = p.prep(`INSERT INTO cookies VALUES (?, ?, ?, ?, ?)`)
	m.stmt.refresh = p.prep(`UPDATE cookies SET expires = ? WHERE cookie_id = ?`)
	m.stmt.delete = p.prep(`DELETE FROM cookies WHERE cookie_id = ?`)

	p.checkErrors()
}

// Create adds the cookie to the DB and returns the UUID
func (m *CookieModel) Create(cookie domain.Cookie) (string, error) { // maybe we shouldn't pass cookie obj?
	/// genrate cookie_id
	UUID, err := uuid.NewRandom()
	if err != nil {
		m.getLogger("uuid.NewRandom()").Error(err)
		return "", err
	}
	cookieID := UUID.String()

	_, err = m.stmt.create.Exec(cookieID, cookie.UserID, cookie.Expires, cookie.UserAccount, cookie.AppspaceID)
	if err != nil {
		m.getLogger("Create()").Error(err)
		return "", err
	}

	return cookieID, nil
}

// Get returns the locally stored values for a cookie id / uuid
func (m *CookieModel) Get(cookieID string) (*domain.Cookie, error) {
	var cookie domain.Cookie

	err := m.stmt.selectCookieID.QueryRowx(cookieID).StructScan(&cookie)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		m.getLogger("Get()").Error(err)
		return nil, err
	}

	return &cookie, nil
}

// UpdateExpires sets the expiration date on the cooke
func (m *CookieModel) UpdateExpires(cookieID string, expires time.Time) error {
	_, err := m.stmt.refresh.Exec(expires, cookieID)
	if err != nil {
		m.getLogger("UpdateExpires()").Error(err)
		return err
	}

	// I don't want to check that rows affected == 1 because if you call this back-to-back
	// it's possible expires didn't change, so affected rows == 0, but this is a non-error.

	return nil
}

// Delete removes the cookie from the DB
func (m *CookieModel) Delete(cookieID string) error {
	_, err := m.stmt.delete.Exec(cookieID)
	if err != nil {
		m.getLogger("Delete()").Error(err)
		return err
	}

	return nil
}

func (m *CookieModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("CookieModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
