package userroutes

// should this be its own isolated package?
// Handle /login /appspace-login /logout
import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/usermodel"
	"github.com/teleclimber/DropServer/internal/validator"
)

// AuthRoutes handles all routes related to authentication
type AuthRoutes struct {
	Views interface {
		Login(http.ResponseWriter, domain.LoginViewData)
		Signup(http.ResponseWriter, domain.SignupViewData)
	}
	SettingsModel interface {
		Get() (domain.Settings, error)
	}
	UserModel interface {
		Create(email, password string) (domain.User, error)
		GetFromEmailPassword(email, password string) (domain.User, error)
	}
	UserInvitationModel interface {
		Get(email string) (domain.UserInvitation, error)
	}
	Authenticator interface {
		SetForAccount(http.ResponseWriter, domain.UserID) error
		UnsetForAccount(http.ResponseWriter, *http.Request)
	}
}

func (a *AuthRoutes) routeGroup(r chi.Router) {
	// If authenticated, redirect to user page
	r.Group(func(r chi.Router) {
		r.Use(mustNotBeAuthenticated)
		r.Get("/signup", a.getSignup)   // include csrf
		r.Post("/signup", a.postSignup) //check csrf
		r.Get("/login", a.getLogin)     // include csrf
		r.Post("/login", a.postLogin)   //check csrf
	})

	r.Get("/logout", a.handleLogout)
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
		if err == usermodel.ErrBadAuth || err == sql.ErrNoRows {
			a.Views.Login(w, invalidLoginMessage)
		} else {
			returnError(w, err)
		}
	} else {
		// we're in. What we do now depends on whether we have an asl or not.
		// if asl != "" {
		// 	a.aslLogin(w, r, user.UserID, asl)
		// } else {
		err := a.Authenticator.SetForAccount(w, user.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
		//}
	}
}

func (a *AuthRoutes) getSignup(w http.ResponseWriter, r *http.Request) {
	settings, err := a.SettingsModel.Get()
	if err != nil {
		returnError(w, err)
		return
	}

	viewData := domain.SignupViewData{
		RegistrationOpen: settings.RegistrationOpen}

	a.Views.Signup(w, viewData)
}

func (a *AuthRoutes) postSignup(w http.ResponseWriter, r *http.Request) {
	// TODO: CSRF!!

	r.ParseForm()

	settings, err := a.SettingsModel.Get()
	if err != nil {
		returnError(w, err)
		return
	}

	invalidData := domain.SignupViewData{
		RegistrationOpen: settings.RegistrationOpen}

	email := strings.ToLower(r.Form.Get("email"))
	dsErr := validator.Email(email)
	if dsErr != nil {
		invalidData.Message = "Please use a valid email"
		a.Views.Signup(w, invalidData)
		return
	}
	invalidData.Email = email

	if !settings.RegistrationOpen {
		_, err := a.UserInvitationModel.Get(email)
		if err != nil {
			if err == sql.ErrNoRows {
				invalidData.Message = "Sorry, this email is not on the invitation list"
				a.Views.Signup(w, invalidData)
				return
			}
			returnError(w, err)
			return
		}
	}

	password := r.Form.Get("password")
	err = validator.Password(password)
	if err != nil {
		invalidData.Message = "Please use a valid password" // would be really nice to tell people how the passwrod is invalid
		a.Views.Signup(w, invalidData)
		return
	}

	password2 := r.Form.Get("password2")
	if password != password2 {
		invalidData.Message = "Passwords did not match" // would be really nice to tell people how the passwrod is invalid
		a.Views.Signup(w, invalidData)
		return
	}

	user, err := a.UserModel.Create(email, password)
	if err != nil {
		if err == usermodel.ErrEmailExists {
			invalidData.Message = "Account already exists with that email"
			a.Views.Signup(w, invalidData)
		} else {
			returnError(w, err)
		}
	} else {
		// we're in
		err := a.Authenticator.SetForAccount(w, user.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	}
}

func (a *AuthRoutes) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.Authenticator.UnsetForAccount(w, r)

	http.Redirect(w, r, "/login", http.StatusFound)
}
