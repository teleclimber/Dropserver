package appspacerouter

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/getcleanhost"
	"github.com/teleclimber/DropServer/internal/shiftpath"
	"github.com/teleclimber/DropServer/internal/validator"
)

// handles /dropserver/ routes of an app-space
// sub routes are:
// please send token {user's dropid}
//  ..Host should be appspace
// Here is a token {token, user drop id, appspace}
//  ..Host should be dropid

// V0DropserverRoutes represents struct for /dropserver appspace routes
type V0DropserverRoutes struct {
	AppspaceModel interface {
		GetFromDomain(string) (*domain.Appspace, error)
	}
	V0RequestToken interface {
		ReceiveToken(ref, token string)
		ReceiveError(ref string, err error)
	}
	V0TokenManager interface {
		SendLoginToken(appspaceID domain.AppspaceID, dropID string, ref string) error
	}
}

func (r *V0DropserverRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	trail := getURLTail(req.Context())

	head, _ := shiftpath.ShiftPath(trail)

	switch head {
	case "login-token-request": // this route is used by other dropserver hosts to send and receive login tokens
		switch req.Method {
		case http.MethodGet:
			http.Error(res, "not found", http.StatusNotFound) // we answer "get" with not found so that typical browser and bot requests that hit this end point don't see anything useful.
		case http.MethodPost:
			r.loginTokenRequest(res, req)
		default:
			http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
		}
	case "login-token":
		switch req.Method {
		case http.MethodGet:
			http.Error(res, "not found", http.StatusNotFound) // we answer "get" with not found so that typical browser and bot requests that hit this end point don't see anything useful.
		case http.MethodPost:
			r.loginTokenResponse(res, req)
		default:
			http.Error(res, "method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(res, "not found", http.StatusNotFound)
	}
}

func (r *V0DropserverRoutes) loginTokenRequest(res http.ResponseWriter, req *http.Request) {
	var data domain.V0LoginTokenRequest
	err := readJSON(req, &data)
	if err != nil {
		http.Error(res, "unable to parse JSON", http.StatusBadRequest)
		return
	}

	err = validator.V0AppspaceLoginRef(data.Ref)
	if err != nil {
		http.Error(res, "invalid ref string", http.StatusBadRequest)
		return
	}

	err = validator.DropIDFull(data.DropID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	dropID := validator.NormalizeDropIDFull(data.DropID)

	// TODO: before going any further, do an mTLS check to see if cert matches the drop id domain!

	// here Host needs to be the appspace domain
	host, err := getcleanhost.GetCleanHost(req.Host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = validator.DomainName(host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	host = validator.NormalizeDomainName(host)

	// It's debatable whether this should be here or in login controller
	// It's here because if appspace can't be found, it's better to let the caller know
	appspace, err := r.AppspaceModel.GetFromDomain(host)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if appspace == nil {
		http.Error(res, "Appspace does not exist: "+host, http.StatusNotFound)
		return
	}

	// If we made it this far, send "accepted"
	// .. and send the login token independently as a new request.
	res.WriteHeader(http.StatusAccepted)

	r.getLogger("loginTokenRequest").Debug("accepted for " + host)

	go r.V0TokenManager.SendLoginToken(appspace.AppspaceID, dropID, data.Ref)
}

// loginTokenResponse is the endpoint for appspace login tokens sent by remote
func (r *V0DropserverRoutes) loginTokenResponse(res http.ResponseWriter, req *http.Request) {
	var data domain.V0LoginTokenResponse
	err := readJSON(req, &data)
	if err != nil {
		http.Error(res, "unable to parse JSON", http.StatusBadRequest)
		return
	}

	// validate the ref string first...
	err = validator.V0AppspaceLoginRef(data.Ref)
	if err != nil {
		http.Error(res, "invalid ref string", http.StatusBadRequest)
		return
	}

	// validate appspace and login token
	err = validator.DomainName(data.Appspace)
	if err != nil {
		r.V0RequestToken.ReceiveError(data.Ref, errors.New("failed to validate domain in token payload from remote"))
		http.Error(res, "invalid appspace domain", http.StatusBadRequest)
		return
	}

	// appspaceDomain := validator.NormalizeDomainName(data.Appspace)
	// TODO now do a mTLS check to ensure sender has a valid cert for that appspace domain

	err = validator.V0AppspaceLoginToken(data.Token) // do we really ened to involve validator package here? It's more for things that might get validated in different places, like domains etc...
	if err != nil {
		http.Error(res, "invalid token", http.StatusBadRequest)
		return
	}

	r.V0RequestToken.ReceiveToken(data.Ref, data.Token)
}

func (r *V0DropserverRoutes) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0DropserverRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

func readJSON(req *http.Request, data interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}

	return nil
}
