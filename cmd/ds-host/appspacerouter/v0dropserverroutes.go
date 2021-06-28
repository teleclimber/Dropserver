package appspacerouter

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
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
	} `checkinject:"required"`
	Authenticator interface {
		Unset(w http.ResponseWriter, r *http.Request)
	} `checkinject:"required"`
	V0RequestToken interface {
		ReceiveToken(ref, token string)
		ReceiveError(ref string, err error)
	} `checkinject:"required"`
	V0TokenManager interface {
		SendLoginToken(appspaceID domain.AppspaceID, dropID string, ref string) error
	} `checkinject:"required"`
}

func (d *V0DropserverRoutes) subRouter() http.Handler {
	mux := chi.NewRouter()

	mux.Get("/login-token-request", notFound) // we answer "get" with "not found" so that typical browser and bot requests that hit this end point don't see anything useful.
	mux.Post("/login-token-request", d.loginTokenRequest)

	mux.Get("/login-token", notFound)
	mux.Post("/login-token", d.loginTokenResponse)

	mux.Get("/logout", d.logout)

	return mux
}

func (d *V0DropserverRoutes) loginTokenRequest(w http.ResponseWriter, r *http.Request) {
	var data domain.V0LoginTokenRequest
	err := readJSON(r, &data)
	if err != nil {
		http.Error(w, "unable to parse JSON", http.StatusBadRequest)
		return
	}

	err = validator.V0AppspaceLoginRef(data.Ref)
	if err != nil {
		http.Error(w, "invalid ref string", http.StatusBadRequest)
		return
	}

	err = validator.DropIDFull(data.DropID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dropID := validator.NormalizeDropIDFull(data.DropID)

	// TODO: before going any further, do an mTLS check to see if cert matches the drop id domain! Or something

	appspace, ok := domain.CtxAppspaceData(r.Context())
	if !ok {
		panic("loginTokenRequest: epxected appspace in request context")
	}

	// If we made it this far, send "accepted"
	// .. and send the login token independently as a new request.
	w.WriteHeader(http.StatusAccepted)

	d.getLogger("loginTokenRequest").Debug("accepted for " + appspace.DomainName)

	go d.V0TokenManager.SendLoginToken(appspace.AppspaceID, dropID, data.Ref)
}

// loginTokenResponse is the endpoint for appspace login tokens sent by remote
func (d *V0DropserverRoutes) loginTokenResponse(w http.ResponseWriter, r *http.Request) {
	var data domain.V0LoginTokenResponse
	err := readJSON(r, &data)
	if err != nil {
		http.Error(w, "unable to parse JSON", http.StatusBadRequest)
		return
	}

	// validate the ref string first...
	err = validator.V0AppspaceLoginRef(data.Ref)
	if err != nil {
		http.Error(w, "invalid ref string", http.StatusBadRequest)
		return
	}

	// validate appspace and login token
	err = validator.DomainName(data.Appspace)
	if err != nil {
		d.V0RequestToken.ReceiveError(data.Ref, errors.New("failed to validate domain in token payload from remote"))
		http.Error(w, "invalid appspace domain", http.StatusBadRequest)
		return
	}

	// appspaceDomain := validator.NormalizeDomainName(data.Appspace)
	// TODO now do a mTLS check to ensure sender has a valid cert for that appspace domain

	err = validator.V0AppspaceLoginToken(data.Token) // do we really ened to involve validator package here? It's more for things that might get validated in different places, like domains etc...
	if err != nil {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	d.V0RequestToken.ReceiveToken(data.Ref, data.Token)
}

func (d *V0DropserverRoutes) logout(w http.ResponseWriter, r *http.Request) {
	d.Authenticator.Unset(w, r)
	if !strings.Contains(r.Header.Get("accept"), "text/html") {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>Logged out</h1>"))
}

func (d *V0DropserverRoutes) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0DropserverRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

func notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not found", http.StatusNotFound)
}

func readJSON(r *http.Request, data interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}

	return nil
}
