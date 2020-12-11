package appspacemetadb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/twine"
)

const (
	getUserCmd     = 12
	userIsOwnerCmd = 13
)

// V0UserModel responds to requests about appspace routes for an appspace
// It can cache results (eventually) for rapid reponse times without hitting the DB.
type V0UserModel struct {
	Validator      domain.Validator
	AppspaceMetaDB interface {
		GetConn(domain.AppspaceID) (domain.DbConn, error)
	}
	AppspaceContactModel interface {
		GetByProxy(domain.AppspaceID, domain.ProxyID) (domain.AppspaceContact, error)
	}

	//going to need a way to know if a user is the owner or not.
	// get appspace user on host side <appspace id, user_id, proxy_id>

	appspaceID domain.AppspaceID
	//do we need stmts? (I think these should be in the DB obj)
}

type userRow struct {
	ProxyID     domain.ProxyID `db:"proxy_id"`
	DisplayName string         `db:"display_name"`
	Permissions string         `db:"permissions"`
}

func (m *V0UserModel) getDB() (*sqlx.DB, error) {
	dbConn, err := m.AppspaceMetaDB.GetConn(m.appspaceID) // pass location key instaed of appspace id
	if err != nil {
		return nil, err
	}
	return dbConn.GetHandle(), err
}

// service HandleMessage is for sandboxed code.
// Not frontend or anything. ...I think?
// host frontend and other systems will use controllers that will call regular model methods.

// HandleMessage processes a command and payload from the reverse listener
func (m *V0UserModel) HandleMessage(message twine.ReceivedMessageI) {
	switch message.CommandID() {
	case getUserCmd:
		// from proxy id fetch user's name and permissions
		// and figure out if they are owner or not.
		m.handleGetUserCommand(message)
	case userIsOwnerCmd:
		m.handleIsOwnerCommand(message)
	default:
		message.SendError("Command not recognized")
	}
}

func (m *V0UserModel) handleGetUserCommand(message twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(message.Payload()))
	user, err := m.Get(proxyID)
	if err != nil {
		message.SendError(err.Error())
		return
	}
	if user.ProxyID == "" {
		message.Reply(13, nil)
	} else {
		bytes, err := json.Marshal(user)
		if err != nil {
			m.getLogger("handleGetUserCommand(), json Marshal error").Error(err)
			message.SendError("Error on host")
		}
		message.Reply(14, bytes)
	}
}

func (m *V0UserModel) handleIsOwnerCommand(message twine.ReceivedMessageI) {
	proxyID := domain.ProxyID(string(message.Payload()))
	contact, err := m.AppspaceContactModel.GetByProxy(m.appspaceID, proxyID)
	if err != nil {
		message.SendError(err.Error())
		return
	}
	if contact.ProxyID == "" {
		// Here we could really use Twine's error-code facility, if it existed
		message.Reply(13, nil) // use command 13 to signify "not found"?
		return
	} else if contact.IsOwner {
		message.Reply(14, nil)
	} else {
		message.Reply(15, nil)
	}
}

// Create adds a new route to the DB
func (m *V0UserModel) Create(proxyID domain.ProxyID, displayName string, permissions []string) error {
	err := m.validatePermissions(permissions)
	if err != nil {
		return err
	}
	err = m.validateProxyID(proxyID)
	if err != nil {
		return err
	}

	db, err := m.getDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO users (proxy_id, display_name, permissions) VALUES (?, ?, ?)`, proxyID, displayName, strings.Join(permissions, ","))
	if err != nil {
		m.getLogger("Create(), db Exec").Error(err)
		return err
	}

	return nil
}

// Update the proxy id's display name and permissions
func (m *V0UserModel) Update(proxyID domain.ProxyID, displayName string, permissions []string) error {
	err := m.validatePermissions(permissions)
	if err != nil {
		return err
	}
	err = m.validateProxyID(proxyID)
	if err != nil {
		return err
	}

	db, err := m.getDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(`UPDATE users SET display_name = ?, permissions= ? WHERE proxy_id = ?`, displayName, strings.Join(permissions, ","), proxyID)
	if err != nil {
		m.getLogger("Update(), db Exec").Error(err)
		return err
	}

	// verify one row was changed.

	return nil
}

// Delete a user from appspace
func (m *V0UserModel) Delete(proxyID domain.ProxyID) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	err = m.validateProxyID(proxyID)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM users WHERE proxy_id = ?`, proxyID)
	if err != nil {
		m.getLogger("Delete(), db Exec").Error(err)
		return err
	}

	return nil
}

// Get a appspace user by proxy id
// If proxy id does not exist it returns zero-value V0User and nil error
func (m *V0UserModel) Get(proxyID domain.ProxyID) (domain.V0User, error) {
	err := m.validateProxyID(proxyID)
	if err != nil {
		return domain.V0User{}, err
	}

	db, err := m.getDB()
	if err != nil {
		return domain.V0User{}, err
	}

	var u userRow
	err = db.Get(&u, `SELECT * FROM users WHERE proxy_id = ?`, proxyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.V0User{}, nil
		}
		m.getLogger("Get(), db Exec").Error(err)
		return domain.V0User{}, err
	}

	return domain.V0User{
		ProxyID:     proxyID,
		DisplayName: u.DisplayName,
		Permissions: strings.Split(u.Permissions, ","),
	}, nil
}

// GetAll appspace users
func (m *V0UserModel) GetAll() ([]domain.V0User, error) {
	db, err := m.getDB()
	if err != nil {
		return []domain.V0User{}, err
	}

	var uRows []userRow
	err = db.Select(&uRows, `SELECT * FROM users`)
	if err != nil {
		if err == sql.ErrNoRows {
			return []domain.V0User{}, nil
		}
		m.getLogger("GetAll(), db Select").Error(err)
		return []domain.V0User{}, err
	}

	users := make([]domain.V0User, len(uRows))
	for i, u := range uRows {
		users[i] = domain.V0User{
			ProxyID:     u.ProxyID,
			DisplayName: u.DisplayName,
			Permissions: strings.Split(u.Permissions, ",")}
	}

	return users, nil
}

func (m *V0UserModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("V0UserModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

func (m *V0UserModel) validateProxyID(proxyID domain.ProxyID) error {
	if proxyID == "" {
		return errors.New("proxy id is empty")
	}
	if len(string(proxyID)) > 10 {
		return errors.New("proxy id longer than 10 chars")
	}
	// TODO check the makeup of proxy id chars. Should be a strict set
	return nil
}

func (m *V0UserModel) validatePermissions(permissions []string) error {
	for _, p := range permissions {
		if len(p) == 0 || len(p) > 20 {
			return errors.New("invalid permission: " + p)
		}
	}
	return nil
}
