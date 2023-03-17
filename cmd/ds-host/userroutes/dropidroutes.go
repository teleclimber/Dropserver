package userroutes

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type DropIDRoutes struct {
	DomainController interface {
		GetDropIDDomains(userID domain.UserID) ([]domain.DomainData, error)
	} `checkinject:"required"`
	DropIDModel interface {
		Create(userID domain.UserID, handle string, dom string, displayName string) (domain.DropID, error)
		Update(userID domain.UserID, handle string, dom string, displayName string) error
		Get(handle string, dom string) (domain.DropID, error)
		GetForUser(userID domain.UserID) ([]domain.DropID, error)
	} `checkinject:"required"`
}

func (d *DropIDRoutes) subRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/", d.handleGet)
	r.Post("/", d.handlePost)
	r.Patch("/", d.handlePatch)

	return r
}

type DropIDAvailableResp struct {
	Available bool `json:"available"`
}

func (d *DropIDRoutes) handleGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		httpInternalServerError(w)
		return
	}

	query := r.URL.Query()
	if _, ok := query["check"]; ok {
		d.checkAvailable(w, r, userID)

	} else {
		d.getForUser(w, r, userID)
	}
}
func (d *DropIDRoutes) checkAvailable(w http.ResponseWriter, r *http.Request, userID domain.UserID) {
	query := r.URL.Query()
	domainNames, ok := query["domain"]
	if !ok || len(domainNames) != 1 {
		writeBadRequest(w, "domain", "no domain provided")
		return
	}
	domainName := validator.NormalizeDomainName(domainNames[0])
	valErr, err := d.validateDomain(domainName, userID)
	if err != nil {
		httpInternalServerError(w)
		return
	}
	if valErr != nil {
		writeBadRequest(w, "domain", valErr.Error())
		return
	}

	handle := ""
	if len(query["handle"]) == 1 && len(query["handle"][0]) != 0 {
		handle, err = url.QueryUnescape(query["handle"][0])
		if err != nil {
			writeBadRequest(w, "handle", err.Error())
			return
		}
	}
	handle = validator.NormalizeDropIDHandle(handle)

	if err := validator.DropIDHandle(handle); err != nil {
		writeBadRequest(w, "handle", err.Error())
		return
	}

	_, err = d.DropIDModel.Get(handle, domainName)
	if err != nil {
		if err == domain.ErrNoRowsInResultSet {
			writeJSON(w, DropIDAvailableResp{Available: true})
		} else {
			returnError(w, err)
		}
	} else {
		writeJSON(w, DropIDAvailableResp{Available: false})
	}
}
func (d *DropIDRoutes) getForUser(w http.ResponseWriter, r *http.Request, userID domain.UserID) {
	dropIDs, err := d.DropIDModel.GetForUser(userID)
	if err != nil {
		returnError(w, err)
		return
	}
	writeJSON(w, dropIDs)
}

// handlePost gets a complete dropid and creates it
func (d *DropIDRoutes) handlePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		httpInternalServerError(w)
		return
	}

	reqData := &domain.DropID{}
	err := readJSON(r, reqData)
	if err != nil {
		returnError(w, err)
		return
	}

	domainName := validator.NormalizeDomainName(reqData.Domain)
	valErr, err := d.validateDomain(domainName, userID)
	if err != nil {
		httpInternalServerError(w)
		return
	}
	if valErr != nil {
		writeBadRequest(w, "domain", err.Error())
		return
	}

	handle := validator.NormalizeDropIDHandle(reqData.Handle)
	if err := validator.DropIDHandle(handle); err != nil {
		writeBadRequest(w, "handle", err.Error())
		return
	}

	displayName := validator.NormalizeDisplayName(reqData.DisplayName)
	// Here it's OK for the display name to be blank.
	if displayName != "" {
		if err = validator.DisplayName(displayName); err != nil {
			writeBadRequest(w, "display name", err.Error())
			return
		}
	}

	dropID, err := d.DropIDModel.Create(userID, handle, domainName, displayName)
	if err != nil {
		if err == domain.ErrUniqueConstraintViolation {
			writeBadRequest(w, "dropid", "dropid is already in use")
			return
		}
		returnError(w, err)
		return
	}

	writeJSON(w, dropID)
}

type PatchDropID struct {
	DisplayName string `json:"display_name"`
}

// handlePatch accepts the parts of a dropid that can be altered
// Namely the display name only for now.
func (d *DropIDRoutes) handlePatch(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		httpInternalServerError(w)
		return
	}

	// We get the dropid we want to modify via url query parameters
	query := r.URL.Query()
	domainNames, ok := query["domain"]
	if !ok || len(domainNames) != 1 {
		writeNotFound(w)
		return
	}
	domainName := validator.NormalizeDomainName(domainNames[0])
	valErr, err := d.validateDomain(domainName, userID)
	if err != nil {
		httpInternalServerError(w)
		return
	}
	if valErr != nil {
		writeBadRequest(w, "domain", valErr.Error())
		return
	}

	handle := ""
	if len(query["handle"]) == 1 && len(query["handle"][0]) != 0 {
		handle, err = url.QueryUnescape(query["handle"][0])
		if err != nil {
			writeBadRequest(w, "handle", err.Error())
			return
		}
		handle = validator.NormalizeDropIDHandle(handle)
	}
	if err := validator.DropIDHandle(handle); err != nil {
		writeBadRequest(w, "handle", err.Error())
		return
	}

	reqData := &PatchDropID{}
	err = readJSON(r, reqData)
	if err != nil {
		writeBadRequest(w, "JSON", err.Error())
		return
	}

	displayName := validator.NormalizeDisplayName(reqData.DisplayName)
	err = validator.DisplayName(displayName)
	if err != nil {
		writeBadRequest(w, "display_name", err.Error())
		return
	}

	err = d.DropIDModel.Update(userID, handle, domainName, displayName)
	if err != nil {
		if err == domain.ErrNoRowsAffected {
			writeNotFound(w) // assume not found if nothing changed.
			return
		}
		writeServerError(w)
		return
	}

	writeOK(w)
}

func (d *DropIDRoutes) validateDomain(domainName string, userID domain.UserID) (error, error) {
	if err := validator.DomainName(domainName); err != nil {
		return err, nil
	}

	domains, err := d.DomainController.GetDropIDDomains(userID)
	if err != nil {
		return nil, err
	}
	domainOK := false
	for _, d := range domains {
		if d.DomainName == domainName {
			domainOK = true
			break
		}
	}
	if !domainOK {
		return errors.New("domain not found"), nil
	}
	return nil, nil
}
