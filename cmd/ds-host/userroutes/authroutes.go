package userroutes

// should this be its own isolated package?
// Handle /login /appspace-login /logout
import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/validator"
)

// AuthRoutes handles all routes related to authentication
type AuthRoutes struct {
	SetupKey interface {
		Has() (bool, error)
		Get() (string, error)
		Delete() error
	} `checkinject:"required"`
	Views interface {
		Login(http.ResponseWriter, domain.LoginViewData)
		Signup(http.ResponseWriter, domain.SignupViewData)
	} `checkinject:"required"`
	SettingsModel interface {
		Get() (domain.Settings, error)
	} `checkinject:"required"`
	UserModel interface {
		CreateWithEmail(email, password string) (domain.User, error)
		GetFromEmailPassword(email, password string) (domain.User, error)
		MakeAdmin(userID domain.UserID) error
	} `checkinject:"required"`
	UserInvitationModel interface {
		Get(email string) (domain.UserInvitation, error)
	} `checkinject:"required"`
	Authenticator interface {
		SetForAccount(http.ResponseWriter, domain.UserID) error
		Unset(http.ResponseWriter, *http.Request)
	} `checkinject:"required"`
}

func (a *AuthRoutes) routeGroup(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(mustNotBeAuthenticated)
		r.Use(a.mustNotHaveSetupKey)    // disables the regular auth routes while setup key exists
		r.Get("/signup", a.getSignup)   // include csrf
		r.Post("/signup", a.postSignup) //check csrf
		r.Get("/login", a.getLogin)     // include csrf
		r.Post("/login", a.postLogin)   //check csrf
	})
	r.Get("/logout", a.handleLogout)

	key, err := a.SetupKey.Get()
	if err != nil || key == "" {
		return
	}
	r.Group(func(r chi.Router) {
		r.Use(a.mustHaveSetupKey)
		r.Use(mustNotBeAuthenticated)
		r.Get("/"+key, a.getSignup)   // include csrf
		r.Post("/"+key, a.postSignup) //check csrf
	})
}

func (a *AuthRoutes) mustHaveSetupKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasKey, err := a.SetupKey.Has()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if !hasKey {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *AuthRoutes) mustNotHaveSetupKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasKey, err := a.SetupKey.Has()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if hasKey {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *AuthRoutes) getLogin(w http.ResponseWriter, r *http.Request) {
	a.Views.Login(w, domain.LoginViewData{})
}

func (a *AuthRoutes) postLogin(w http.ResponseWriter, r *http.Request) {
	// TODO: CSRF!!

	r.ParseForm()

	invalidLoginMessage := domain.LoginViewData{
		Message: "Login incorrect"}

	email := strings.ToLower(r.Form.Get("email"))
	dsErr := validator.Email(email)
	if dsErr != nil {
		// actually re-render page with generic error
		a.Views.Login(w, invalidLoginMessage)
		return
	}

	invalidLoginMessage.Email = email

	password := r.Form.Get("password")
	dsErr = validator.Password(password)
	if dsErr != nil {
		a.Views.Login(w, invalidLoginMessage)
		return
	}

	user, err := a.UserModel.GetFromEmailPassword(email, password)
	if err != nil {
		if err == domain.ErrBadAuth || err == sql.ErrNoRows {
			a.Views.Login(w, invalidLoginMessage)
		} else {
			returnError(w, err)
		}
	} else {
		err := a.Authenticator.SetForAccount(w, user.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (a *AuthRoutes) getSignup(w http.ResponseWriter, r *http.Request) {
	viewData, err := a.getSignupViewData()
	if err != nil {
		returnError(w, err)
		return
	}

	a.Views.Signup(w, viewData)
}

func (a *AuthRoutes) postSignup(w http.ResponseWriter, r *http.Request) {
	// TODO: CSRF!!

	r.ParseForm()

	viewData, err := a.getSignupViewData()
	if err != nil {
		returnError(w, err)
		return
	}

	email := strings.ToLower(r.Form.Get("email"))
	dsErr := validator.Email(email)
	if dsErr != nil {
		viewData.Message = "Please use a valid email"
		a.Views.Signup(w, viewData)
		return
	}
	viewData.Email = email

	// Having a setup key means we allow the registration
	if !viewData.RegistrationOpen && !viewData.HasSetupKey {
		_, err := a.UserInvitationModel.Get(email)
		if err != nil {
			if err == sql.ErrNoRows {
				viewData.Message = "Sorry, this email is not on the invitation list"
				a.Views.Signup(w, viewData)
				return
			}
			returnError(w, err)
			return
		}
	}

	password := r.Form.Get("password")
	err = validator.Password(password)
	if err != nil {
		viewData.Message = "Please use a valid password" // would be really nice to tell people how the passwrod is invalid
		a.Views.Signup(w, viewData)
		return
	}

	password2 := r.Form.Get("password2")
	if password != password2 {
		viewData.Message = "Passwords did not match" // would be really nice to tell people how the passwrod is invalid
		a.Views.Signup(w, viewData)
		return
	}

	user, err := a.UserModel.CreateWithEmail(email, password)
	if err != nil {
		if err == domain.ErrIdentifierExists {
			viewData.Message = "Account already exists with that email"
			a.Views.Signup(w, viewData)
		} else {
			returnError(w, err)
		}
		return
	}

	err = a.Authenticator.SetForAccount(w, user.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if viewData.HasSetupKey {
		err = a.UserModel.MakeAdmin(user.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		err = a.SetupKey.Delete()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func (a *AuthRoutes) getSignupViewData() (domain.SignupViewData, error) {
	d := domain.SignupViewData{FormAction: "/signup"}

	settings, err := a.SettingsModel.Get()
	if err != nil {
		return d, err
	}
	d.RegistrationOpen = settings.RegistrationOpen

	has, err := a.SetupKey.Has()
	if err != nil {
		return d, err
	}
	if has {
		key, err := a.SetupKey.Get()
		if err != nil {
			return d, err
		}
		d.HasSetupKey = true
		d.FormAction = "/" + key
	}

	return d, nil
}

func (a *AuthRoutes) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.Authenticator.Unset(w, r)

	http.Redirect(w, r, "/login", http.StatusFound)
}
