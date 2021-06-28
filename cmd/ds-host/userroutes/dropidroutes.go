package userroutes

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
	validatorExternal "gopkg.in/validator.v2"
)

type DropIDRoutes struct {
	DomainController interface {
		GetDropIDDomains(userID domain.UserID) ([]domain.DomainData, error)
	} `checkinject:"required"`
	DropIDModel interface {
		Create(userID domain.UserID, handle string, dom string, displayName string) (domain.DropID, error)
		Update(userID domain.UserID, handle string, dom string, displayName string) (domain.DropID, error)
		Get(handle string, dom string) (domain.DropID, error)
		GetForUser(userID domain.UserID) ([]domain.DropID, error)
	} `checkinject:"required"`
}

func (d *DropIDRoutes) subRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/", d.handleGet)
	r.Post("/", d.handlePost)

	return r
}

func (d *DropIDRoutes) handleGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		//d.getLogger("getUserData").Error(errors.New("no auth user id"))
		httpInternalServerError(w)
		return
	}

	query := r.URL.Query()
	domainNames, ok := query["domain"]
	if ok {
		if len(domainNames) != 1 {
			returnError(w, errBadRequest)
			return
		}
		domainName := domainNames[0]

		if err := validator.DomainName(domainName); err != nil {
			returnError(w, errBadRequest)
			return
		}
		domainName = validator.NormalizeDomainName(domainName)

		domains, err := d.DomainController.GetDropIDDomains(userID)
		if err != nil {
			returnError(w, err)
			return
		}
		domainOK := false
		for _, d := range domains {
			if d.DomainName == domainName {
				domainOK = true
				break
			}
		}
		if !domainOK {
			returnError(w, errForbidden)
			return
		}

		handle := ""
		if len(query["handle"]) == 1 {
			// don't we need to unescape query?
			handle = validator.NormalizeDropIDHandle(query["handle"][0])
			if err := validator.DropIDHandle(handle); err != nil {
				returnError(w, errBadRequest)
				return
			}
		}

		dropID, err := d.DropIDModel.Get(handle, domainName)
		if err != nil {
			if err == sql.ErrNoRows {
				returnError(w, errNotFound)
			} else {
				returnError(w, err)
			}
		} else if dropID.UserID == userID {
			writeJSON(w, dropID)
		} else { // if the drop id requested belogs to a different user, assume this is a request to find out if the id is available. So just return the OK status with no data.
			w.WriteHeader(http.StatusOK)
		}
	} else {
		d.getForUser(w, r, userID)
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

// handlePost accepts a complete dropID struct,
// And the model either updates or inserts depending on whether
// the domain+handle key exists.
func (d *DropIDRoutes) handlePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		//d.getLogger("getUserData").Error(errors.New("no auth user id"))
		httpInternalServerError(w)
		return
	}

	reqData := &domain.DropID{}
	err := readJSON(r, reqData)
	if err != nil {
		returnError(w, err)
		return
	}

	if errs := validatorExternal.Validate(reqData); errs != nil {
		http.Error(w, errs.Error(), http.StatusBadRequest)
		return
	}

	insert := false
	dropID, err := d.DropIDModel.Get(reqData.Handle, reqData.Domain)
	if err != nil {
		if err == sql.ErrNoRows {
			insert = true
		} else {
			// actual error
			returnError(w, err)
			return
		}
	}

	if insert {
		dropID, err = d.DropIDModel.Create(userID, reqData.Handle, reqData.Domain, reqData.DisplayName)
		if err != nil {
			returnError(w, err)
			return
		}
	} else {
		if dropID.UserID != userID {
			returnError(w, errForbidden)
			return
		}
		dropID, err = d.DropIDModel.Update(dropID.UserID, dropID.Handle, dropID.Domain, reqData.DisplayName)
		if err != nil {
			returnError(w, err)
			return
		}
	}

	writeJSON(w, dropID)
}
