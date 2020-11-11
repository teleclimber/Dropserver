package appspacemetadb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/twine"
)

const (
	createCmd = 12
	deleteCmd = 13
)

// V0RouteModel responds to requests about appspace routes for an appspace
// It can cache results (eventually) for rapid reponse times without hitting the DB.
type V0RouteModel struct {
	Validator      domain.Validator
	AppspaceMetaDB interface {
		GetConn(domain.AppspaceID) (domain.DbConn, error)
	}
	RouteEvents interface {
		Send(domain.AppspaceID, domain.AppspaceRouteEvent)
	}
	appspaceID domain.AppspaceID
	//do we need stmts? (I think these should be in the DB obj)
}

type routeRow struct {
	Methods uint16
	Path    string
	Auth    string
	Handler string
}

func (m *V0RouteModel) getDB() (*sqlx.DB, error) {
	dbConn, err := m.AppspaceMetaDB.GetConn(m.appspaceID) // pass location key instaed of appspace id
	if err != nil {
		return nil, err
	}
	return dbConn.GetHandle(), err
}

// HandleMessage processes a command and payload from the reverse listener
func (m *V0RouteModel) HandleMessage(message twine.ReceivedMessageI) {
	switch message.CommandID() {
	case createCmd:
		m.reverseCmdCreate(message)
	case deleteCmd:
		m.reverseCmdDelete(message)
	default:
		message.SendError("Command not recognized")
	}
	// more...

}

// Generally speaking I think errors should be logged
// And errors returned might be generic "something bad happened, logged."

func (m *V0RouteModel) reverseCmdCreate(message twine.ReceivedMessageI) {
	var data struct {
		Methods   []string                    `json:"methods"`
		RoutePath string                      `json:"route-path"`
		Auth      domain.AppspaceRouteAuth    `json:"auth"`
		Handler   domain.AppspaceRouteHandler `json:"handler"`
	}

	payload := message.Payload()

	err := json.Unmarshal(payload, &data)
	if err != nil {
		m.getLogger("reverseCmdCreate, json.Unmarshal").Error(err)
		message.SendError("json unmarshall error")
		return
	}

	err = m.Create(data.Methods, data.RoutePath, data.Auth, data.Handler)
	if err != nil {
		m.getLogger("reverseCmdCreate, m.Create").Error(err)
		if err != nil {
			message.SendError("db error on route create: " + err.Error())
		}
		return
	}

	message.SendOK()
}
func (m *V0RouteModel) reverseCmdDelete(message twine.ReceivedMessageI) {
	var data struct {
		Methods   []string `json:"methods"`
		RoutePath string   `json:"route-path"`
	}

	payload := message.Payload()

	err := json.Unmarshal(payload, &data)
	if err != nil {
		m.getLogger("reverseCmdDelete, json.Unmarshal").Error(err)
		message.SendError("json unmarshall error")
		return
	}

	err = m.Delete(data.Methods, data.RoutePath)
	if err != nil {
		m.getLogger("reverseCmdDelete, m.Delete").Error(err)
		message.SendError("db error on delete")
		return
	}

	message.SendOK()
}

var errRouteExists = errors.New("Appspace route already exists")

// Create adds a new route to the DB
// Wonder if I need an "overwrite" flag?
func (m *V0RouteModel) Create(methods []string, routePath string, auth domain.AppspaceRouteAuth, handler domain.AppspaceRouteHandler) error { // and more stuff...
	rr, err := m.Get(methods, routePath)
	if err != nil {
		return err
	}
	if rr != nil && len(*rr) > 0 {
		return errRouteExists
	}

	var mBitz uint16 = 0
	for _, m := range methods {
		mBit, err := v0normalizeMethod(m)
		if err != nil {
			return err
		}
		mBitz = mBitz | mBit
	}

	routePath, err = v0normalizePath(routePath)
	if err != nil {
		return err
	}

	err = v0validateAuth(auth)
	if err != nil {
		return err
	}

	authStr, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	err = v0validateHandler(handler)
	if err != nil {
		return err
	}

	handlerStr, err := json.Marshal(handler)
	if err != nil {
		return err
	}

	db, err := m.getDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO routes (methods, path, auth, handler) VALUES (?, ?, ?, ?)`, strconv.Itoa(int(mBitz)), routePath, authStr, handlerStr)
	if err != nil {
		// should log this as it's not normal
		return err
	}

	if m.RouteEvents != nil {
		m.RouteEvents.Send(m.appspaceID, domain.AppspaceRouteEvent{
			AppspaceID: m.appspaceID,
			Path:       routePath})
	}

	return nil
}

// Get returns all routes that
// - match one of the methods passed, and
// - matches the routePath exactly (no interpolation is done to match sub-paths)
func (m *V0RouteModel) Get(methods []string, routePath string) (*[]domain.AppspaceRouteConfig, error) {
	var rr []domain.AppspaceRouteConfig //may not work, may need to have interim type to egt from db row, then parse the json columsn

	var mBitz uint16 = 0
	for _, m := range methods {
		mBit, err := v0normalizeMethod(m)
		if err != nil {
			return &rr, err
		}
		mBitz = mBitz | mBit
	}

	routePath, err := v0normalizePath(routePath)
	if err != nil {
		return &rr, err
	}

	db, err := m.getDB()
	if err != nil {
		return nil, err
	}

	var rowz []routeRow

	err = db.Select(&rowz, `SELECT * FROM routes WHERE methods&? != 0 AND path = ?`, mBitz, routePath)
	if err != nil {
		return nil, err
	}

	// if no error expand routeRows into AppspaceRouteConfig
	rr = make([]domain.AppspaceRouteConfig, len(rowz))
	for i, r := range rowz {
		routeConfig, err := v0appspaceRouteFromRow(r)
		if err != nil {
			return nil, err
		}
		rr[i] = routeConfig
	}

	return &rr, nil
}

// Delete each route that matches a method, and the routePath exactly
// If a row has multiple methods, the method is removed from the row.
// If no methods remain, the row is deleted.
func (m *V0RouteModel) Delete(methods []string, routePath string) error {
	// To remove methods from existing route
	// add up all possible method bits
	// then remove the methods from that.
	// Then we can update db rows with & and the existing value
	var mBitz uint16 = 0
	for _, b := range v0methodBits {
		mBitz = mBitz | b
	}
	for _, m := range methods {
		mBit, err := v0normalizeMethod(m)
		if err != nil {
			return err
		}
		mBitz = mBitz ^ mBit
	}

	routePath, err := v0normalizePath(routePath)
	if err != nil {
		return err
	}

	db, err := m.getDB()
	if err != nil {
		return err
	}

	// Do a transaction here to avoid problems?

	_, err = db.Exec(`UPDATE routes SET methods = methods&? WHERE path = ?`, mBitz, routePath)
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM routes WHERE methods = 0 AND path = ?`, routePath)
	if err != nil {
		return err
	}

	if m.RouteEvents != nil {
		m.RouteEvents.Send(m.appspaceID, domain.AppspaceRouteEvent{
			AppspaceID: m.appspaceID,
			Path:       routePath})
	}

	return nil
}

// GetAll returns all routes
func (m *V0RouteModel) GetAll() (*[]domain.AppspaceRouteConfig, error) {
	var rr []domain.AppspaceRouteConfig //may not work, may need to have interim type to egt from db row, then parse the json columsn

	db, err := m.getDB()
	if err != nil {
		return nil, err
	}

	var rowz []routeRow

	err = db.Select(&rowz, `SELECT * FROM routes`)
	if err != nil {
		return nil, err
	}

	// if no error expand routeRows into AppspaceRouteConfig
	rr = make([]domain.AppspaceRouteConfig, len(rowz))
	for i, r := range rowz {
		routeConfig, err := v0appspaceRouteFromRow(r)
		if err != nil {
			return nil, err
		}
		rr[i] = routeConfig
	}

	return &rr, nil
}

// GetPath returns all routes with that exact path
func (m *V0RouteModel) GetPath(routePath string) (*[]domain.AppspaceRouteConfig, error) {
	var rr []domain.AppspaceRouteConfig //may not work, may need to have interim type to egt from db row, then parse the json columsn

	routePath, err := v0normalizePath(routePath)
	if err != nil {
		return &rr, err
	}

	db, err := m.getDB()
	if err != nil {
		return nil, err
	}

	var rowz []routeRow

	err = db.Select(&rowz, `SELECT * FROM routes WHERE path = ?`, routePath)
	if err != nil {
		return nil, err
	}

	// if no error expand routeRows into AppspaceRouteConfig
	rr = make([]domain.AppspaceRouteConfig, len(rowz))
	for i, r := range rowz {
		routeConfig, err := v0appspaceRouteFromRow(r)
		if err != nil {
			return nil, err
		}
		rr[i] = routeConfig
	}

	return &rr, nil
}

// Match finds the route that should handle the request
// The path will be broken into parts to find the subset path that matches.
func (m *V0RouteModel) Match(method string, routePath string) (*domain.AppspaceRouteConfig, error) {
	mBit, err := v0normalizeMethod(method)
	if err != nil {
		return nil, err
	}

	routePath, err = v0normalizePath(routePath)
	if err != nil {
		return nil, err
	}

	db, err := m.getDB()
	if err != nil {
		return nil, err
	}

	// path matching
	// We think routepath will always have a leading /, and no trailing /
	pieces := strings.Split(strings.TrimLeft(routePath, "/"), "/")
	inPaths := make([]string, len(pieces)+1)
	inPath := ""
	inPaths[0] = "/"
	for i, p := range pieces {
		inPath = inPath + "/" + p
		inPaths[i+1] = inPath
	}

	q, args, err := sqlx.In(`SELECT * FROM routes WHERE methods&?=? AND path IN (?) ORDER BY LENGTH(path) DESC`, mBit, mBit, inPaths)
	if err != nil {
		// log because it's an error in this code
		return nil, err
	}

	q = db.Rebind(q)

	var r routeRow
	err = db.Get(&r, q, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // no rows found, no matching route exists
		}
		return nil, err
	}

	routeConfig, err := v0appspaceRouteFromRow(r)
	if err != nil {
		return nil, err
	}

	// need to return something....
	return &routeConfig, nil
}

func (m *V0RouteModel) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("V0RouteModel")
	if note != "" {
		r.AddNote(note)
	}
	return r
}

// func v0selectMethodsOr()

func v0normalizePath(routePath string) (string, error) {
	// to lower case
	// leading /
	// no trailing /
	// can be file name at end
	// remove query params (presence of query params should throw error?)

	u, err := url.Parse(routePath)
	if err != nil {
		return "", err
	}

	ret := strings.ToLower(path.Clean(u.Path))

	if ret == "." {
		return "", errors.New("Failed to normalize path: " + routePath)
	}

	return ret, nil
}

// help: https://www.databasejournal.com/features/mssql/article.php/3359321/Storing-Multiple-Statuses-Using-an-Integer-Column.htm
var v0methodBits = map[string]uint16{
	"get":     1,
	"head":    2,
	"post":    4,
	"put":     8,
	"delete":  16,
	"connect": 32,
	"options": 64,
	"trace":   128,
	"patch":   256}

func v0normalizeMethod(method string) (uint16, error) {
	method = strings.ToLower(method)
	bitz, ok := v0methodBits[method]
	if !ok {
		return 0, errors.New("method not recognized: " + method)
	}
	return bitz, nil
}

func v0GetMethodsFromBits(bitz uint16) []string {
	ret := make([]string, 0)
	for m, b := range v0methodBits {
		if b&bitz == b {
			ret = append(ret, m)
		}
	}
	sort.Strings(ret)
	return ret
}

func v0appspaceRouteFromRow(r routeRow) (domain.AppspaceRouteConfig, error) {
	var auth domain.AppspaceRouteAuth
	err := json.Unmarshal([]byte(r.Auth), &auth)
	if err != nil {
		return domain.AppspaceRouteConfig{}, err
	}

	var handler domain.AppspaceRouteHandler
	err = json.Unmarshal([]byte(r.Handler), &handler)
	if err != nil {
		return domain.AppspaceRouteConfig{}, err
	}

	routeConfig := domain.AppspaceRouteConfig{
		Methods: v0GetMethodsFromBits(r.Methods),
		Path:    r.Path,
		Auth:    auth,
		Handler: handler}

	return routeConfig, nil
}

func v0validateAuth(auth domain.AppspaceRouteAuth) error {
	switch auth.Type {
	case "owner":
		// no need for anything else. all other fields shoudl be zero-value
		return nil
	case "public":
		// no need for anything else
		return nil
	default:
		return errors.New("Unrecognized Auth type: " + auth.Type)
	}
}

var v0validPaths = [3]string{"@dropserver/", "@app/", "@appspace/"}

func v0validateHandler(handler domain.AppspaceRouteHandler) error {
	switch handler.Type {
	case "function":
		if handler.File == "" {
			return errors.New("Route handler of type function has empty file (module) field")
		}
		return nil
		// I don't think you are required to set the function. If it's not set the runner calles the default export.
	case "file":
		if handler.Path == "" {
			return errors.New("Route handler of type file has empty path field")
		}
		pathValid := false
		for _, p := range v0validPaths {
			if strings.HasPrefix(handler.Path, p) {
				pathValid = true
				break
			}
		}
		if !pathValid {
			return errors.New("Route handler of type file has invalid path: " + handler.Path)
		}
		return nil
	default:
		return errors.New("Unrecognized Handler type: " + handler.Type)
	}

}
