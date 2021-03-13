package userroutes

import (
	"database/sql"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
	validatorExternal "gopkg.in/validator.v2"
)

type DropIDRoutes struct {
	DomainController interface {
		GetDropIDDomains(userID domain.UserID) ([]domain.DomainData, error)
	}
	DropIDModel interface {
		Create(userID domain.UserID, handle string, dom string, displayName string) (domain.DropID, error)
		Update(userID domain.UserID, handle string, dom string, displayName string) (domain.DropID, error)
		Get(handle string, dom string) (domain.DropID, error)
		GetForUser(userID domain.UserID) ([]domain.DropID, error)
	}
}

func (d *DropIDRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Authentication == nil || !routeData.Authentication.UserAccount {
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we messed up
	}

	switch req.Method {
	case http.MethodGet:
		d.handleGet(res, req, routeData)
	case http.MethodPost:
		d.handlePost(res, req, routeData)
	default:
		http.Error(res, "bad method for /dropid", http.StatusBadRequest)
	}

}

func (d *DropIDRoutes) handleGet(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	query := req.URL.Query()
	domainNames, ok := query["domain"]
	if ok {
		if len(domainNames) != 1 {
			returnError(res, errBadRequest)
			return
		}
		domainName := domainNames[0]

		if err := validator.DomainName(domainName); err != nil {
			returnError(res, errBadRequest)
			return
		}
		domainName = validator.NormalizeDomainName(domainName)

		domains, err := d.DomainController.GetDropIDDomains(routeData.Authentication.UserID)
		if err != nil {
			returnError(res, err)
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
			returnError(res, errForbidden)
			return
		}

		handle := ""
		if len(query["handle"]) == 1 {
			handle = validator.NormalizeDropIDHandle(query["handle"][0])
			if err := validator.DropIDHandle(handle); err != nil {
				returnError(res, errBadRequest)
				return
			}
		}

		dropID, err := d.DropIDModel.Get(handle, domainName)
		if err != nil {
			if err == sql.ErrNoRows {
				returnError(res, errNotFound)
			} else {
				returnError(res, err)
			}
		} else if dropID.UserID == routeData.Authentication.UserID {
			writeJSON(res, dropID)
		} else { // if the drop id requested belogs to a different user, assume this is a request to find out if the id is available. So just return the OK status with no data.
			res.WriteHeader(http.StatusOK)
		}
	} else {
		d.getForUser(res, req, routeData)
	}

}
func (d *DropIDRoutes) getForUser(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	dropIDs, err := d.DropIDModel.GetForUser(routeData.Authentication.UserID)
	if err != nil {
		returnError(res, err)
		return
	}

	writeJSON(res, dropIDs)
}

// handlePost accepts a complete dropID struct,
// And the model either updates or inserts depending on whether
// the domain+handle key exists.
func (d *DropIDRoutes) handlePost(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	reqData := &domain.DropID{}
	err := readJSON(req, reqData)
	if err != nil {
		returnError(res, err)
		return
	}

	if errs := validatorExternal.Validate(reqData); errs != nil {
		http.Error(res, errs.Error(), http.StatusBadRequest)
		return
	}

	insert := false
	dropID, err := d.DropIDModel.Get(reqData.Handle, reqData.Domain)
	if err != nil {
		if err == sql.ErrNoRows {
			insert = true
		} else {
			// actual error
			returnError(res, err)
			return
		}
	}

	if insert {
		dropID, err = d.DropIDModel.Create(routeData.Authentication.UserID, reqData.Handle, reqData.Domain, reqData.DisplayName)
	} else {
		if dropID.UserID != routeData.Authentication.UserID {
			returnError(res, errForbidden)
			return
		}
		dropID, err = d.DropIDModel.Update(dropID.UserID, dropID.Handle, dropID.Domain, reqData.DisplayName)
		if err != nil {
			returnError(res, err)
			return
		}
	}

	writeJSON(res, dropID)
}
