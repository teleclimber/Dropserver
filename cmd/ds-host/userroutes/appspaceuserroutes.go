package userroutes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type PostAppspaceUser struct {
	AuthType    string   `json:"auth_type"`
	AuthID      string   `json:"auth_id"`
	DisplayName string   `json:"display_name"`
	Avatar      string   `json:"avatar"` //  "replace", any other value means no avatar is loaded
	Permissions []string `json:"permissions"`
}

// All this needs to be versioned?

// AppspaceUserRoutes handles routes for getting and mainpulating
// appspace users
type AppspaceUserRoutes struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
		Create(appspaceID domain.AppspaceID, authType string, authID string) (domain.ProxyID, error)
		UpdateAuth(appspaceID domain.AppspaceID, proxyID domain.ProxyID, authType string, authID string) error
		UpdateMeta(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, permissions []string) error
		Delete(appspaceID domain.AppspaceID, proxyID domain.ProxyID) error
	} `checkinject:"required"`
	Avatars interface {
		Save(locationKey string, proxyID domain.ProxyID, img io.Reader) (string, error)
		Remove(locationKey string, fn string) error
	} `checkinject:"required"`
	Config                *domain.RuntimeConfig `checkinject:"required"`
	AppspaceLocation2Path interface {
		Avatar(string, string) string
	} `checkinject:"required"`
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
		r.Get("/avatar/{filename}", a.getAvatar)
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
	users, err := a.AppspaceUserModel.GetAll(appspace.AppspaceID)
	if err != nil {
		returnError(w, err)
		return
	}

	// still unsure whether we just use the struct from domain or a special struct for the JSON.

	writeJSON(w, users)
}

func (a *AppspaceUserRoutes) newUser(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())

	// multi part
	err := r.ParseMultipartForm(16 << 20) // maxMemory 16MB
	if err != nil {
		http.Error(w, "failed to parse multipart message: "+err.Error(), http.StatusBadRequest)
		return
	}

	f, _, err := r.FormFile("metadata")
	if err != nil {
		http.Error(w, "failed to get metadata: "+err.Error(), http.StatusBadRequest)
		return
	}
	metadata, err := ioutil.ReadAll(f)
	if err != nil {
		http.Error(w, "failed to get metadata: "+err.Error(), http.StatusBadRequest)
		return
	}

	reqData := PostAppspaceUser{}
	err = json.Unmarshal(metadata, &reqData)
	if err != nil {
		http.Error(w, "failed to get metadata: "+err.Error(), http.StatusBadRequest)
		return
	}

	authID, err := validateAuthStrings(reqData.AuthType, reqData.AuthID)
	if err != nil {
		http.Error(w, fmt.Errorf("failed to validate auth: %w", err).Error(), http.StatusBadRequest)
		return
	}

	displayName := validator.NormalizeDisplayName(reqData.DisplayName)
	// For now display name can not be blank.
	// Revisit when display name can be inherited from contact info or dropid
	if displayName == "" {
		http.Error(w, "display name can not be blank", http.StatusBadRequest)
		return
	}
	if err = validator.DisplayName(displayName); err != nil {
		http.Error(w, fmt.Errorf("failed to normalize display name: %w", err).Error(), http.StatusBadRequest)
		return
	}

	proxyID, err := a.AppspaceUserModel.Create(appspace.AppspaceID, reqData.AuthType, authID)
	if err != nil {
		returnError(w, err)
		return
	}

	// handle avatar...
	avatar := ""
	if reqData.Avatar == "replace" {
		f, _, err = r.FormFile("avatar")
		if err != nil {
			// this should trigger a rollback
			http.Error(w, "unable to get avatar from multipart: "+err.Error(), http.StatusBadRequest)
			return
		}
		avatar, err = a.Avatars.Save(appspace.LocationKey, proxyID, f)
		if err != nil {
			// trigger rollback
			http.Error(w, "unable to save avatar: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// not yet sure how to deal with permissions....
	if len(reqData.Permissions) != 0 {
		// if( len)

		// hasMeta = true
	}

	err = a.AppspaceUserModel.UpdateMeta(appspace.AppspaceID, proxyID, displayName, avatar, []string{})
	if err != nil {
		// This is where it would be nice to roll back....
		// We can ignore the error and return the result of "get"
		// And user will notice that something didn't quite work.
		// error is captured in logger in model
		returnError(w, err)
		return
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

	// multi part
	err := r.ParseMultipartForm(16 << 20) // maxMemory 16MB
	if err != nil {
		http.Error(w, "failed to parse multipart message: "+err.Error(), http.StatusBadRequest)
		return
	}

	// now get the metadata from parts:
	f, _, err := r.FormFile("metadata")
	if err != nil {
		http.Error(w, "failed to get metadata: "+err.Error(), http.StatusBadRequest)
		return
	}
	metadata, err := ioutil.ReadAll(f)
	if err != nil {
		http.Error(w, "failed to get metadata: "+err.Error(), http.StatusBadRequest)
		return
	}

	reqData := PostAppspaceUser{}
	err = json.Unmarshal(metadata, &reqData)
	if err != nil {
		http.Error(w, "failed to get metadata: "+err.Error(), http.StatusBadRequest)
		return
	}

	// handle potential auth changes...
	if reqData.AuthID != "" {
		authID, err := validateAuthStrings(reqData.AuthType, reqData.AuthID)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to validate auth: %w", err).Error(), http.StatusBadRequest)
			return
		}
		err = a.AppspaceUserModel.UpdateAuth(appspace.AppspaceID, user.ProxyID, reqData.AuthType, authID)
		if err != nil {
			returnError(w, err)
			return
		}
	}

	displayName := validator.NormalizeDisplayName(reqData.DisplayName)
	if err = validator.DisplayName(displayName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	avatar := ""
	switch reqData.Avatar {
	case "preserve":
		avatar = user.Avatar
	case "delete":
		if user.Avatar != "" {
			a.Avatars.Remove(appspace.LocationKey, user.Avatar)
		}
		avatar = ""
	case "replace":
		// load image data from request, then process
		f, _, err = r.FormFile("avatar")
		if err != nil {
			http.Error(w, "unable to get avatar from multipart: "+err.Error(), http.StatusBadRequest)
			return
		}
		avatar, err = a.Avatars.Save(appspace.LocationKey, user.ProxyID, f)
		if err != nil {
			http.Error(w, "unable to save avatar: "+err.Error(), http.StatusBadRequest)
			return
		}
		// now delete the old avatar...
		if user.Avatar != "" {
			a.Avatars.Remove(appspace.LocationKey, user.Avatar)
		}
	default:
		http.Error(w, "avatar metadata not recognized: "+reqData.Avatar, http.StatusBadRequest)
	}

	// not yet sure how to deal with permissions....
	if len(reqData.Permissions) != 0 {
		// if( len)

		// hasMeta = true
	}

	err = a.AppspaceUserModel.UpdateMeta(appspace.AppspaceID, user.ProxyID, displayName, avatar, []string{})
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

func (a *AppspaceUserRoutes) getAvatar(w http.ResponseWriter, r *http.Request) {
	appspace, _ := domain.CtxAppspaceData(r.Context())
	avatarFilename := chi.URLParam(r, "filename")

	err := validator.AppspaceAvatarFilename(avatarFilename)
	if err != nil {
		returnError(w, err)
		return
	}

	http.ServeFile(w, r, a.AppspaceLocation2Path.Avatar(appspace.LocationKey, avatarFilename))
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
