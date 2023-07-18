package hosttools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type ChangeDomain struct {
	DB                    *domain.DB
	AppspaceLocation2Path interface {
		Data(locationKey string) string
	}

	oldDomain  string
	newDomain  string
	dropIDsMap map[string]string
}

// Here is the change domain tool
// It changes domain names in the DB and in appspaces and anywhere that's needed.

// Places where we have found domain names:
// In host db:
// - table dropids has domain column
// - cookies has a domain col
// - appspaces has comain name col
// - appspaces has a dropid col
// - remote appspace has a dropid col
// - [remote appspace has a domain but it's inherently not a concern]
// In appspaces:
// - appspace-meta.db users table: dropids have domains
// -

// But let's just entertain the ideas:
// - it should be possible to re-domain an appspace
//   ..including creating redirects to the new domain, wehther it's local or not
// - re-domain dropids, or make dropid transfer a well supported process.
// - change external access domain without changing the appspace domains that use that domain?

// Then go through dropids and make changes, recording the original dropid and new dropids map.
// Then for each table in host DB that has dropids, swap them out.
//  -> appspaces, remote appspaces

// wipe cookes (ideally for any domain that matches the old domain)

// For each table in host db that has a domain column change out the domain as needed
//  -> appspaces

// Then have to iterate over every single appspace:
// -> read all location keys from appspaces DB
// For each loc, open the appsapcemeta db (or can we use existing code for this?)
// swap dropids for users based on the mappings determined above.

// TODO: we really need a "dry-run" mode

func (c *ChangeDomain) ChangeDomain(oldDomain, newDomain string) {
	c.oldDomain = oldDomain
	c.newDomain = newDomain

	c.wipeCookies()
	c.changeDropIDs()
	c.changeDomains()
	c.changeAppspaceMetaDB()
}

func (c *ChangeDomain) wipeCookies() {
	c.DB.Handle.MustExec(`DELETE FROM cookies`) // just wipe everything for now.
}

type DropID struct {
	ID     int    `db:"rowid"`
	Domain string `db:"domain"`
	Handle string `db:"handle"`
}
type Appspace struct {
	AppspaceID  int    `db:"appspace_id"`
	LocationKey string `db:"location_key"`
	DropID      string `db:"dropid"`
	DomainName  string `db:"domain_name"`
}
type RemoteAppspaceRowIDDropID struct {
	RowID       int    `db:"rowid"`
	OwnerDropID string `db:"owner_dropid"`
	DropID      string `db:"dropid"`
}

func (c *ChangeDomain) changeDropIDs() {
	c.dropIDsMap = map[string]string{}

	// get all dropids:
	var dropids []DropID
	err := c.DB.Handle.Select(&dropids, `SELECT rowid, domain, handle FROM dropids`)
	errorIsFatal("Select all dropids", err)

	for _, d := range dropids {
		if newD, replace := c.replaceDomain(d.Domain); replace {
			c.DB.Handle.MustExec(`UPDATE dropids SET domain = ? WHERE rowid = ?`, newD, d.ID)
			c.dropIDsMap[validator.JoinDropID(d.Handle, d.Domain)] = validator.JoinDropID(d.Handle, newD)
		}
	}

	fmt.Println("dropids map", c.dropIDsMap)

	for _, a := range c.getAppspaces() {
		if newDropid, ok := c.dropIDsMap[a.DropID]; ok {
			fmt.Println("replace appspace dropid:", a, newDropid)
			c.DB.Handle.MustExec(`UPDATE appspaces SET dropid = ? WHERE appspace_id = ?`, newDropid, a.AppspaceID)
		}
	}

	var remoteAppspaces []RemoteAppspaceRowIDDropID
	err = c.DB.Handle.Select(&remoteAppspaces, `SELECT rowid, owner_dropid, dropid FROM remote_appspaces`)
	errorIsFatal("Select all remote appspaces", err)
	fmt.Println("remote appsapce", remoteAppspaces)
	for _, r := range remoteAppspaces {
		if newDropid, ok := c.dropIDsMap[r.OwnerDropID]; ok {
			fmt.Println("replace remote appspace owner dropid:", r, newDropid)
			c.DB.Handle.MustExec(`UPDATE remote_appspaces SET owner_dropid = ? WHERE rowid = ?`, newDropid, r.RowID)
		}
		if newDropid, ok := c.dropIDsMap[r.DropID]; ok {
			fmt.Println("replace remote appspace dropid:", r, newDropid)
			c.DB.Handle.MustExec(`UPDATE remote_appspaces SET dropid = ? WHERE rowid = ?`, newDropid, r.RowID)
		}
	}
}

func (c *ChangeDomain) changeDomains() {
	for _, a := range c.getAppspaces() {
		if newDomain, replace := c.replaceDomain(a.DomainName); replace {
			fmt.Println("replace appspace domain:", a, newDomain)
			c.DB.Handle.MustExec(`UPDATE appspaces SET domain_name = ? WHERE appspace_id = ?`, newDomain, a.AppspaceID)
		}
	}
}

type AppspaceUser struct {
	ProxyID domain.ProxyID `db:"proxy_id"`
	AuthID  string         `db:"auth_id"`
}

func (c *ChangeDomain) changeAppspaceMetaDB() {
	for _, a := range c.getAppspaces() {
		fmt.Println("Working on appspace meta db for:", a)

		dbPath := filepath.Join(c.AppspaceLocation2Path.Data(a.LocationKey), "appspace-meta.db")

		handle, err := sqlx.Open("sqlite3", dbPath)
		errorIsFatal("open appspace meta db: "+dbPath, err)

		// HERE we have to change the dropids of users of the appspace
		// get all users first
		var users []AppspaceUser
		err = handle.Select(&users, `SELECT proxy_id, auth_id FROM users WHERE auth_type = "dropid"`)
		errorIsFatal("appspace meta db slect all users", err)

		for _, u := range users {
			if newDropid, ok := c.dropIDsMap[u.AuthID]; ok {
				fmt.Println("replace appspace dropid:", u, newDropid)
				handle.MustExec(`UPDATE users SET auth_id = ? WHERE proxy_id = ?`, newDropid, u.ProxyID)
			}
		}
	}
}

///////////////////// helper functions

func (c *ChangeDomain) getAppspaces() (appspaces []Appspace) {
	err := c.DB.Handle.Select(&appspaces, `SELECT appspace_id, location_key, dropid, domain_name FROM appspaces`)
	errorIsFatal("Select all appspaces", err)
	return
}

func (c *ChangeDomain) replaceDomain(d string) (string, bool) {
	if !strings.HasSuffix(d, c.oldDomain) {
		return "", false
	}
	newP := strings.TrimSuffix(d, c.oldDomain)
	return newP + c.newDomain, true
}

func errorIsFatal(msg string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%v Error: %v", msg, err))
	}
}
