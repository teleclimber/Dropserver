package userroutes

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"gopkg.in/validator.v2"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
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
	} `checkinject:"required"`
}

func (c *ContactRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", c.getContacts)
	r.Post("/", c.postNewContact)

	r.Route("/{contact}", func(r chi.Router) {
		r.Get("/", c.getContact)
	})

	return r
}

func (c *ContactRoutes) getContacts(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	contacts, err := c.ContactModel.GetForUser(userID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	respData := make([]ContactResp, len(contacts))
	for i, c := range contacts {
		respData[i] = makeContactResp(c)
	}

	writeJSON(w, respData)
}

func (c *ContactRoutes) getContact(w http.ResponseWriter, r *http.Request) {
	contact, err := c.getContactFromRequest(r)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, makeContactResp(contact))
}

// CreateContactPostReq is incoming json for creating a contact
type CreateContactPostReq struct {
	Name        string `json:"name" validate:"nonzero,max=100"`
	DisplayName string `json:"display_name" validate:"nonzero,max=100"`
}

func (c *ContactRoutes) postNewContact(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	reqData := &CreateContactPostReq{}
	err := readJSON(r, reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if errs := validator.Validate(reqData); errs != nil {
		http.Error(w, errs.Error(), http.StatusBadRequest)
		return
	}

	contact, err := c.ContactModel.Create(userID, reqData.Name, reqData.DisplayName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, makeContactResp(contact))
}

func (c *ContactRoutes) getContactFromRequest(r *http.Request) (domain.Contact, error) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	contactIDStr := chi.URLParam(r, "contact")

	contactIDInt, err := strconv.Atoi(contactIDStr)
	if err != nil {
		return domain.Contact{}, errBadRequest
	}
	contactID := domain.ContactID(contactIDInt)

	contact, err := c.ContactModel.Get(contactID)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Contact{}, errNotFound
		}
		return domain.Contact{}, err
	}
	if contact.UserID != userID {
		return domain.Contact{}, errForbidden
	}

	return contact, nil
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
