package userroutes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

type PostAuth struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
	Op         string `json:"op"`
}

type PostAppspaceUser struct {
	DisplayName string     `json:"display_name"`
	Avatar      string     `json:"avatar"` //  "replace", any other value means no avatar is loaded
	Permissions []string   `json:"permissions"`
	Auths       []PostAuth `json:"auths"`
}

// All this needs to be versioned?

// AppspaceUserRoutes handles routes for getting and mainpulating
// appspace users
type AppspaceUserRoutes struct {
	AppspaceUserModel interface {
		Get(appspaceID domain.AppspaceID, proxyID domain.ProxyID) (domain.AppspaceUser, error)
		GetAll(appspaceID domain.AppspaceID) ([]domain.AppspaceUser, error)
		Create(appspaceID domain.AppspaceID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) (domain.ProxyID, error)
		Update(appspaceID domain.AppspaceID, proxyID domain.ProxyID, displayName string, avatar string, auths []domain.EditAppspaceUserAuth) error
		UpdateAvatar(appspaceID domain.AppspaceID, proxyID domain.ProxyID, avatar string) error
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
			if err == domain.ErrNoRowsInResultSet {
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
	metadata, err := io.ReadAll(f)
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

	auths, err := getEditAuths(reqData.Auths, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	proxyID, err := a.AppspaceUserModel.Create(appspace.AppspaceID, displayName, "", auths)
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

	err = a.AppspaceUserModel.UpdateAvatar(appspace.AppspaceID, proxyID, avatar)
	if err != nil {
		returnError(w, err) // maybe softwen the error if the avatar was not updated correctly.
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
	metadata, err := io.ReadAll(f)
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

	auths, err := getEditAuths(reqData.Auths, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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

	err = a.AppspaceUserModel.Update(appspace.AppspaceID, user.ProxyID, displayName, avatar, auths)
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

var errNoOp = errors.New("edit is no-op")

func getEditAuths(postAuths []PostAuth, allowRemove bool) ([]domain.EditAppspaceUserAuth, error) {
	auths := make([]domain.EditAppspaceUserAuth, 0)
	for _, auth := range postAuths {
		editAuth, err := getEditAuth(auth, allowRemove)
		if err == errNoOp {
			// skip it
		} else if err != nil {
			return nil, err
		} else {
			auths = append(auths, editAuth)
		}
	}
	return auths, nil
}
func getEditAuth(auth PostAuth, allowRemove bool) (domain.EditAppspaceUserAuth, error) {
	op := domain.EditOperation(auth.Op)

	if op == domain.EditOperationNoOp {
		return domain.EditAppspaceUserAuth{}, errNoOp
	}
	if !allowRemove && op == domain.EditOperationRemove {
		return domain.EditAppspaceUserAuth{}, fmt.Errorf("Operation 'remove' on %s: %s is not allowed on this route", auth.Type, auth.Identifier)
	}
	if op != domain.EditOperationAdd && op != domain.EditOperationRemove {
		return domain.EditAppspaceUserAuth{}, fmt.Errorf("unknown operation %s", auth.Op)
	}
	authID, err := validateAuthStrings(auth.Type, auth.Identifier)
	if err != nil {
		return domain.EditAppspaceUserAuth{}, fmt.Errorf("failed to validate auth: %w", err)
	}
	return domain.EditAppspaceUserAuth{
		Type:       auth.Type,
		Identifier: authID,
		Operation:  op,
	}, nil
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
	} else if authType == "tsnetid" {
		authID = validator.NormalizeTSNetIDFull(authID)
		err = validator.TSNetIDFull(authID)
		if err != nil {
			return "", err
		}
	} else {
		return "", errors.New("unknown auth type " + authType)
	}
	return authID, nil
}
