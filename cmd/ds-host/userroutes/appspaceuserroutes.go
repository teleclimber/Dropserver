package userroutes

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type PostAppspaceUser struct {
	AuthType    string   `json:"auth_type"`
	AuthID      string   `json:"auth_id"`
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
}

type PatchAppspaceUserMeta struct {
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
}

// AppspaceUserRoutes handles routes for getting and mainpulating
// appspace users
type AppspaceUserRoutes struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetForAppspace(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
		Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
		UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, permissions []string) error
		Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error
	}
}

func (a *AppspaceUserRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", a.getAllUsers)
	r.Post("/", a.newUser)

	r.Route("/{proxyid}", func(r chi.Router) {
		r.Use(a.userCtx)
		r.Get("/", a.getUser)
		r.Patch("/", a.updateUserMeta)
		r.Delete("/", a.deleteUser)
	})

	return r
}

// using middleware not necessary here. Could simply have a getter function
// since all users of middleware are right here.
func (a *AppspaceUserRoutes) userCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appspace, _ := domain.CtxAppspaceData(r.Context())

		proxyStr := chi.URLParam(r, "proxyid")

		err := validator.UserProxyID(proxyStr)
		if err != nil {
			http.Error(w, "invalid proxy id", http.StatusBadRequest)
			return
		}

		user, err := a.AppspaceUserModel.Get(appspace.AppspaceID, domain.ProxyID(proxyStr))
		if err != nil {
			if err == sql.ErrNoRows {
				returnError(w, errNotFound)
			} else {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}

		r = r.WithContext(domain.CtxWithAppspaceUserData(r.Context(), user))

		next.ServeHTTP(w, r)
	})
}

func (a *AppspaceUserRoutes) getAllUsers(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	users, err := a.AppspaceUserModel.GetForAppspace(appspace.AppspaceID)
	if err != nil {
		returnError(w, err)
		return
	}

	// still unsure whether we just use the struct from domain or a special struct for the JSON.

	writeJSON(w, users)
}

func (a *AppspaceUserRoutes) newUser(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	reqData := PostAppspaceUser{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authID, err := validateAuthStrings(reqData.AuthType, reqData.AuthID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	displayName := ""
	if reqData.DisplayName != "" {
		displayName = validator.NormalizeDisplayName(reqData.DisplayName)
		if err = validator.DisplayName(displayName); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// not yet sure how to deal with permissions....
	if len(reqData.Permissions) != 0 {
		// if( len)

		// hasMeta = true
	}

	proxyID, err := a.AppspaceUserModel.Create(appspace.AppspaceID, reqData.AuthType, authID)
	if err != nil {
		returnError(w, err)
		return
	}

	if displayName != "" {
		err = a.AppspaceUserModel.UpdateMeta(appspace.AppspaceID, proxyID, displayName, []string{})
		if err != nil {
			// This is where it would be nice to roll back....
			// We can ignore the error and return the result of "get"
			// And user will notice that something didn't quite work.
			// error is captured in logger in model
		}
	}

	appspaceUser, err := a.AppspaceUserModel.Get(appspace.AppspaceID, proxyID)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, appspaceUser)
}

func (a *AppspaceUserRoutes) getUser(w http.ResponseWriter, r *http.Request) {
	user, _ := domain.CtxAppspaceUserData(r.Context())
	writeJSON(w, user)
}

func (a *AppspaceUserRoutes) updateUserMeta(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	user, _ := domain.CtxAppspaceUserData(r.Context())

	reqData := PatchAppspaceUserMeta{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	displayName := validator.NormalizeDisplayName(reqData.DisplayName)
	if err = validator.DisplayName(displayName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// not yet sure how to deal with permissions....
	if len(reqData.Permissions) != 0 {
		// if( len)

		// hasMeta = true
	}

	err = a.AppspaceUserModel.UpdateMeta(appspace.AppspaceID, user.ProxyID, displayName, []string{})
	if err != nil {
		returnError(w, err)
		return
	}

	appspaceUser, err := a.AppspaceUserModel.Get(appspace.AppspaceID, user.ProxyID)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, appspaceUser)
}

func (a *AppspaceUserRoutes) deleteUser(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	user, _ := domain.CtxAppspaceUserData(r.Context())

	err := a.AppspaceUserModel.Delete(appspace.AppspaceID, user.ProxyID)
	if err != nil {
		returnError(w, err)
	}
}

func validateAuthStrings(authType, authID string) (string, error) {
	err := validator.AppspaceUserAuthType(authType)
	if err != nil {
		return "", err
	}

	if authType == "email" {
		authID = validator.NormalizeEmail(authID)
		err = validator.Email(authID)
		if err != nil {
			return "", err
		}
	} else if authType == "dropid" {
		authID = validator.NormalizeDropIDFull(authID)
		err = validator.DropIDFull(authID)
		if err != nil {
			return "", err
		}
	} else {
		return "", errors.New("unimplemented auth type " + authType)
	}
	return authID, nil
}
