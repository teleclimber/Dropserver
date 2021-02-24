package userroutes

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"gopkg.in/validator.v2"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// ContactResp is the contact data sent to client as json
type ContactResp struct {
	UserID      domain.UserID    `json:"user_id"`
	ContactID   domain.ContactID `json:"contact_id"`
	Name        string           `json:"name"`
	DisplayName string           `json:"display_name"`
	Created     time.Time        `json:"created_dt"`
	// proabbly include array of authentication methods
}

//ContactRoutes serves routes related to user's contacts
type ContactRoutes struct {
	ContactModel interface {
		Create(userID domain.UserID, name string, displayName string) (domain.Contact, error)
		Update(userID domain.UserID, contactID domain.ContactID, name string, displayName string) error
		Delete(userID domain.UserID, contactID domain.ContactID) error
		Get(contactID domain.ContactID) (domain.Contact, error)
		GetForUser(userID domain.UserID) ([]domain.Contact, error)
		// InsertAppspaceContact(appspaceID domain.AppspaceID, contactID domain.ContactID, proxyID domain.ProxyID) error
		// DeleteAppspaceContact(appspaceID domain.AppspaceID, contactID domain.ContactID) error
		// GetContactProxy(appspaceID domain.AppspaceID, contactID domain.ContactID) (domain.ProxyID, error) // not userful as route?
		// GetByProxy(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.ContactID, error)        /// appspaceroutes? Or actually not useful as a route
		// GetContactAppspaces(contactID domain.ContactID) ([]domain.AppspaceContact, error)
		// GetAppspaceContacts(appspaceID domain.AppspaceID) ([]domain.AppspaceContact, error) // this is probaly in appspaceroutes
	}
}

// ServeHTTP handles http traffic to the application routes
// Namely upload, create new application, delete, ...
func (c *ContactRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Authentication == nil || !routeData.Authentication.UserAccount {
		// maybe log it?
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we dun fogged up
	}

	contact, ok, err := c.getContactFromPath(routeData)
	if err != nil {
		returnError(res, err)
		return
	}
	method := req.Method

	if !ok {
		switch method {
		case http.MethodGet:
			c.getContacts(res, req, routeData)
		case http.MethodPost:
			c.postNewContact(res, req, routeData)
		default:
			http.Error(res, "bad method for /contact", http.StatusBadRequest)
		}
	} else {
		head, tail := shiftpath.ShiftPath(routeData.URLTail)
		routeData.URLTail = tail

		switch head {
		case "":
			c.getContact(res, contact) // assumes get?
		default:
			http.Error(res, "bad request for /contact/:id", http.StatusBadRequest)
		}
	}
}

func (c *ContactRoutes) getContacts(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	contacts, err := c.ContactModel.GetForUser(routeData.Authentication.UserID)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}

	respData := make([]ContactResp, len(contacts))
	for i, c := range contacts {
		respData[i] = makeContactResp(c)
	}

	writeJSON(res, respData)
}

func (c *ContactRoutes) getContact(res http.ResponseWriter, contact domain.Contact) {
	writeJSON(res, makeContactResp(contact))
}

// CreateContactPostReq is incoming json for creating a contact
type CreateContactPostReq struct {
	Name        string `json:"name" validate:"nonzero,max=100"`
	DisplayName string `json:"display_name" validate:"nonzero,max=100"`
}

func (c *ContactRoutes) postNewContact(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	reqData := &CreateContactPostReq{}
	err := readJSON(req, reqData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if errs := validator.Validate(reqData); errs != nil {
		http.Error(res, errs.Error(), http.StatusBadRequest)
		return
	}

	contact, err := c.ContactModel.Create(routeData.Authentication.UserID, reqData.Name, reqData.DisplayName)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(res, makeContactResp(contact))
}

func (c *ContactRoutes) getContactFromPath(routeData *domain.AppspaceRouteData) (domain.Contact, bool, error) {
	contactIDStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if contactIDStr == "" {
		return domain.Contact{}, false, nil
	}

	contactIDInt, err := strconv.Atoi(contactIDStr)
	if err != nil {
		return domain.Contact{}, true, errBadRequest
	}
	contactID := domain.ContactID(contactIDInt)

	contact, err := c.ContactModel.Get(contactID)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Contact{}, true, errNotFound
		}
		return domain.Contact{}, true, err
	}
	if contact.UserID != routeData.Authentication.UserID {
		return domain.Contact{}, true, errForbidden
	}

	return contact, true, nil
}

func makeContactResp(c domain.Contact) ContactResp {
	return ContactResp{
		UserID:      c.UserID,
		ContactID:   c.ContactID,
		Name:        c.Name,
		DisplayName: c.DisplayName,
		Created:     c.Created,
	}
}
