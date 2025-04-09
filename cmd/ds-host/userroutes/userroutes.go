package userroutes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/docgen"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	dshostfrontend "github.com/teleclimber/DropServer/frontend-ds-host"
	"github.com/teleclimber/DropServer/internal/validator"
)

var errNotFound = errors.New("not found")
var errBadRequest = errors.New("bad request") // feels like this should be wrapped or something, so we can also have access to the original error?
var errForbidden = errors.New("forbidden")

func returnError(res http.ResponseWriter, err error) {
	switch err {
	case errNotFound:
		http.Error(res, "not found", http.StatusNotFound)
	case errBadRequest:
		http.Error(res, "bad request", http.StatusBadRequest)
	case errForbidden:
		http.Error(res, "forbidden", http.StatusForbidden)
	default:
		http.Error(res, "internal server error", http.StatusInternalServerError)
	}
}

type subRoutes interface {
	subRouter() http.Handler
}
type routeGroup interface {
	routeGroup(chi.Router)
}

// UserRoutes handles routes for appspaces.
type UserRoutes struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	Authenticator interface {
		AccountUser(http.Handler) http.Handler
	} `checkinject:"required"`
	Views interface {
		GetStaticFS() fs.FS
	} `checkinject:"required"`
	AuthRoutes           routeGroup `checkinject:"required"`
	AppspaceLoginRoutes  routeGroup `checkinject:"required"`
	ApplicationRoutes    subRoutes  `checkinject:"required"`
	AppspaceRoutes       subRoutes  `checkinject:"required"`
	RemoteAppspaceRoutes subRoutes  `checkinject:"required"`
	ContactRoutes        subRoutes  `checkinject:"required"`
	DomainRoutes         subRoutes  `checkinject:"required"`
	DropIDRoutes         subRoutes  `checkinject:"required"`
	MigrationJobRoutes   subRoutes  `checkinject:"required"`
	AdminRoutes          subRoutes  `checkinject:"required"`
	AppspaceStatusEvents interface {
		SubscribeOwner(domain.UserID) <-chan domain.AppspaceStatusEvent
		Unsubscribe(ch <-chan domain.AppspaceStatusEvent)
	} `checkinject:"required"`
	AppspaceTSNetStatusEvents interface {
		SubscribeOwner(domain.UserID) <-chan domain.TSNetAppspaceStatus
		Unsubscribe(ch <-chan domain.TSNetAppspaceStatus)
	} `checkinject:"required"`
	AppspaceTSNetPeersEvents interface {
		SubscribeOwner(domain.UserID) <-chan domain.AppspaceID
		Unsubscribe(ch <-chan domain.AppspaceID)
	} `checkinject:"required"`
	MigrationJobEvents interface {
		SubscribeOwner(domain.UserID) <-chan domain.MigrationJob
		Unsubscribe(ch <-chan domain.MigrationJob)
	} `checkinject:"required"`
	AppGetterEvents interface {
		SubscribeOwner(domain.UserID) <-chan domain.AppGetEvent
		Unsubscribe(ch <-chan domain.AppGetEvent)
	} `checkinject:"required"`
	UserModel interface {
		GetFromID(userID domain.UserID) (domain.User, error)
		UpdateEmail(userID domain.UserID, email string) error
		UpdatePassword(userID domain.UserID, password string) error
		GetFromEmailPassword(email, password string) (domain.User, error)
		IsAdmin(userID domain.UserID) bool
	} `checkinject:"required"`

	mux *chi.Mux
}

func (u *UserRoutes) Init() {
	r := chi.NewRouter()

	r.Use(addCSPHeaders)

	// Load auth user ID for all requests.
	r.Use(u.Authenticator.AccountUser)

	// serve frontend assets as a directory
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(u.Views.GetStaticFS()))))

	frontendFS, fserr := fs.Sub(dshostfrontend.FS, "dist")
	if fserr != nil {
		panic(fserr)
	}
	r.Handle("/frontend-assets/*", http.FileServer(http.FS(frontendFS)))

	r.Group(u.AuthRoutes.routeGroup)

	r.Group(func(r chi.Router) {
		r.Use(mustBeAuthenticated)

		// add a "update cookie" as a trailing middleware. It'll only get called if request doesn't get aborted. (I think? Does it really matter?)

		r.Group(u.AppspaceLoginRoutes.routeGroup)

		r.Get("/events/", u.startSSEEvents)

		r.Route("/api", func(r chi.Router) {
			r.Mount("/admin", u.AdminRoutes.subRouter())

			r.Get("/instance/", u.getInstanceData)

			r.Get("/user/", u.getUserData)
			r.Patch("/user/email/", u.changeUserEmail)
			r.Patch("/user/password/", u.changeUserPassword)

			r.Mount("/domainname", u.DomainRoutes.subRouter())
			r.Mount("/dropid", u.DropIDRoutes.subRouter())
			r.Mount("/application", u.ApplicationRoutes.subRouter())
			r.Mount("/appspace", u.AppspaceRoutes.subRouter())
			r.Mount("/remoteappspace", u.RemoteAppspaceRoutes.subRouter())
			r.Mount("/contact", u.ContactRoutes.subRouter())
			r.Mount("/migration-job", u.MigrationJobRoutes.subRouter())
		})
	})

	r.Get("/*", u.serveAppIndex)

	u.mux = r
}

// ServeHTTP handles http traffic to the user routes
func (u *UserRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	u.mux.ServeHTTP(res, req)
}

func (u *UserRoutes) DumpRoutes(dumpRoutes string) {
	if dumpRoutes == "" {
		return
	}
	err := ioutil.WriteFile(dumpRoutes, []byte(docgen.MarkdownRoutesDoc(u.mux, docgen.MarkdownOpts{
		ProjectPath: "github.com/teleclimber/DropServer",
		Intro:       "Welcome to the dropserver routes generated docs.",
	})), 0644)
	if err != nil {
		u.getLogger("Init dump routes").Error(err)
	} else {
		u.getLogger("Init").Log("Dumped routes to file " + dumpRoutes)
	}
}

func addCSPHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// mustBeAuthenticated redirects to login or sends unauthorized response
// if there is no auth user id.
func mustBeAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := domain.CtxAuthUserID(r.Context())
		if !ok {
			// TODO: only do this when request is for an html page.
			if strings.HasPrefix(r.URL.Path, "/api") {
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// mustNotBeAuthenticated middleware redirects to user home page
// if they are logged in
func mustNotBeAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := domain.CtxAuthUserID(r.Context())
		if ok {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (u *UserRoutes) serveAppIndex(w http.ResponseWriter, r *http.Request) {
	htmlBytes, err := dshostfrontend.FS.ReadFile("dist/index.html")
	if err != nil {
		returnError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlBytes)
}

type InstanceData struct {
	DsHostVersion string `json:"ds_host_version"`
	DenoVersion   string `json:"deno_version"`
}

func (u *UserRoutes) getInstanceData(w http.ResponseWriter, r *http.Request) {
	data := InstanceData{
		DsHostVersion: u.Config.Exec.CmdVersion,
		DenoVersion:   u.Config.Exec.DenoVersion,
	}
	writeJSON(w, data)
}

// UserData is single user
type UserData struct {
	Email   string `json:"email"`
	UserID  int    `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
}

// getUserData returns a json with {email: ""...""} I think, so far.
func (u *UserRoutes) getUserData(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		u.getLogger("getUserData").Error(errors.New("no auth user id"))
		httpInternalServerError(w)
		return
	}
	user, err := u.UserModel.GetFromID(userID)
	if err != nil {
		httpInternalServerError(w)
		return
	}

	isAdmin := u.UserModel.IsAdmin(user.UserID)

	userData := UserData{
		UserID:  int(user.UserID),
		Email:   user.Email,
		IsAdmin: isAdmin}

	writeJSON(w, userData)
}

type PatchUserEmailReq struct {
	Email string `json:"email"`
}

func (u *UserRoutes) changeUserEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		u.getLogger("changeUserEmail").Error(errors.New("no auth user id"))
		httpInternalServerError(w)
		return
	}

	var data PatchUserEmailReq
	err := readJSON(r, &data)
	if err != nil {
		u.getLogger("changeUserEmail").AddNote("readJson").Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = validator.Email(data.Email)
	if err != nil {
		w.WriteHeader(http.StatusOK) // Status OK means request was technically correct but action did not take place
		w.Write([]byte("Email not valid"))
		return
	}

	err = u.UserModel.UpdateEmail(userID, data.Email)
	if err != nil {
		if err == domain.ErrIdentifierExists {
			w.WriteHeader(http.StatusOK) // Status OK means request was technically correct but action did not take place
			w.Write([]byte("Email already in use"))
		} else {
			httpInternalServerError(w)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent) // Status no Content means action took place, nothing to say in response.
}

type PatchPasswordReq struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (u *UserRoutes) changeUserPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		u.getLogger("changeUserPassword").Error(errors.New("no auth user id"))
		httpInternalServerError(w)
		return
	}

	var data PatchPasswordReq
	err := readJSON(r, &data)
	if err != nil {
		u.getLogger("changeUserPassword").AddNote("readJson").Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = validator.Password(data.New)
	if err != nil {
		u.getLogger("changeUserPassword").AddNote("validator.Password(data.New)").Error(err)
		w.WriteHeader(http.StatusOK) // Status OK means request was technically correct but action did not take place
		w.Write([]byte("New password invalid"))
		return
	}

	user, err := u.UserModel.GetFromID(userID)
	if err != nil {
		httpInternalServerError(w)
		return
	}

	_, err = u.UserModel.GetFromEmailPassword(user.Email, data.Old)
	if err != nil {
		w.WriteHeader(http.StatusOK) // Status OK means request was technically correct but action did not take place
		w.Write([]byte("Old password incorrect"))
		return
	}

	err = u.UserModel.UpdatePassword(user.UserID, data.New)
	if err != nil {
		returnError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK) // TODO send no content.
}

func (u *UserRoutes) startSSEEvents(w http.ResponseWriter, r *http.Request) {
	authUserID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		// should never happen
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientGone := r.Context().Done()

	asStatCh := u.AppspaceStatusEvents.SubscribeOwner(authUserID)
	defer u.AppspaceStatusEvents.Unsubscribe(asStatCh)

	asTSNetStatCh := u.AppspaceTSNetStatusEvents.SubscribeOwner(authUserID)
	defer u.AppspaceTSNetStatusEvents.Unsubscribe(asTSNetStatCh)

	asTSNetPeersCh := u.AppspaceTSNetPeersEvents.SubscribeOwner(authUserID)
	defer u.AppspaceTSNetPeersEvents.Unsubscribe(asTSNetPeersCh)

	migrationJobCh := u.MigrationJobEvents.SubscribeOwner(authUserID)
	defer u.MigrationJobEvents.Unsubscribe(migrationJobCh)

	appGetterCh := u.AppGetterEvents.SubscribeOwner(authUserID)
	defer u.AppGetterEvents.Unsubscribe(appGetterCh)

	rc := http.NewResponseController(w)
	for {
		select {
		case <-clientGone:
			return
		case stat := <-asStatCh:
			u.sendSSEEvent(w, "AppspaceStatus", stat)
		case stat := <-asTSNetStatCh:
			u.sendSSEEvent(w, "AppspaceTSNetStatus", stat)
		case aid := <-asTSNetPeersCh:
			u.sendSSEEvent(w, "AppspaceTSNetPeers", aid)
		case job := <-migrationJobCh:
			u.sendSSEEvent(w, "MigrationJob", job)
		case get := <-appGetterCh:
			u.sendSSEEvent(w, "AppGetter", get)
		}

		err := rc.Flush()
		if err != nil {
			u.getLogger("startSSEEvents() rc.Flush()").Error(err)
			return
		}
	}
}

func (u *UserRoutes) sendSSEEvent(w http.ResponseWriter, name string, data interface{}) {
	eventJSON, err := json.Marshal(data)
	if err != nil {
		u.getLogger("sendSSEEvent() json.Marshal() event:" + name).Error(err)
		return
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, eventJSON)
	if err != nil {
		u.getLogger("sendSSEEvent() fmt.Fprintf() event:" + name).Error(err)
		return
	}
}

func (u *UserRoutes) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("UserRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
