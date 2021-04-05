package userroutes

// should this be its own isolated package?
// Handle /login /appspace-login /logout
import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/usermodel"
	"github.com/teleclimber/DropServer/internal/shiftpath"
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

// ServeHTTP handles all /login routes
func (a *AuthRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail
	if head == "signup" {
		a.handleSignup(res, req, routeData)
	} else if head == "login" {
		a.handleLogin(res, req, routeData)
	} else if head == "logout" {
		a.handleLogout(res, req, routeData)
	} else {
		http.Error(res, "Bad path", http.StatusBadRequest)
	}
}

func (a *AuthRoutes) handleSignup(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	switch req.Method {
	case http.MethodGet:
		a.getSignup(res, req, routeData)
	case http.MethodPost:
		a.postSignup(res, req, routeData)
	default:
		http.Error(res, "Bad method", http.StatusBadRequest)
	}
}

func (a *AuthRoutes) handleLogin(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	switch req.Method {
	case http.MethodGet:
		a.loginGet(res, req, routeData)
	case http.MethodPost:
		a.loginPost(res, req, routeData)
	default:
		http.Error(res, "Bad method", http.StatusBadRequest)
	}
}

func (a *AuthRoutes) loginGet(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Authentication != nil && routeData.Authentication.UserAccount {
		http.Redirect(res, req, "/", http.StatusFound)
	} else {
		a.Views.Login(res, domain.LoginViewData{})
	}
}

func (a *AuthRoutes) loginPost(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// TODO: CSRF!!

	req.ParseForm()

	invalidLoginMessage := domain.LoginViewData{
		Message: "Login incorrect"}

	email := strings.ToLower(req.Form.Get("email"))
	dsErr := validator.Email(email)
	if dsErr != nil {
		// actually re-render page with generic error
		a.Views.Login(res, invalidLoginMessage)
		return
	}

	invalidLoginMessage.Email = email

	password := req.Form.Get("password")
	dsErr = validator.Password(password)
	if dsErr != nil {
		a.Views.Login(res, invalidLoginMessage)
		return
	}

	user, err := a.UserModel.GetFromEmailPassword(email, password)
	if err != nil {
		if err == usermodel.ErrBadAuth || err == sql.ErrNoRows {
			a.Views.Login(res, invalidLoginMessage)
		} else {
			returnError(res, err)
		}
	} else {
		// we're in. What we do now depends on whether we have an asl or not.
		// if asl != "" {
		// 	a.aslLogin(res, req, user.UserID, asl)
		// } else {
		err := a.Authenticator.SetForAccount(res, user.UserID)
		if err != nil {
			http.Error(res, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(res, req, "/", http.StatusFound)
		//}
	}
}

func (a *AuthRoutes) getSignup(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	settings, err := a.SettingsModel.Get()
	if err != nil {
		returnError(res, err)
		return
	}

	viewData := domain.SignupViewData{
		RegistrationOpen: settings.RegistrationOpen}

	a.Views.Signup(res, viewData)
}

func (a *AuthRoutes) postSignup(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// TODO: CSRF!!

	req.ParseForm()

	settings, err := a.SettingsModel.Get()
	if err != nil {
		returnError(res, err)
		return
	}

	invalidData := domain.SignupViewData{
		RegistrationOpen: settings.RegistrationOpen}

	email := strings.ToLower(req.Form.Get("email"))
	dsErr := validator.Email(email)
	if dsErr != nil {
		invalidData.Message = "Please use a valid email"
		a.Views.Signup(res, invalidData)
		return
	}
	invalidData.Email = email

	if !settings.RegistrationOpen {
		_, err := a.UserInvitationModel.Get(email)
		if err != nil {
			if err == sql.ErrNoRows {
				invalidData.Message = "Sorry, this email is not on the invitation list"
				a.Views.Signup(res, invalidData)
				return
			}
			returnError(res, err)
			return
		}
	}

	password := req.Form.Get("password")
	err = validator.Password(password)
	if err != nil {
		invalidData.Message = "Please use a valid password" // would be really nice to tell people how the passwrod is invalid
		a.Views.Signup(res, invalidData)
		return
	}

	password2 := req.Form.Get("password2")
	if password != password2 {
		invalidData.Message = "Passwords did not match" // would be really nice to tell people how the passwrod is invalid
		a.Views.Signup(res, invalidData)
		return
	}

	user, err := a.UserModel.Create(email, password)
	if err != nil {
		if err == usermodel.ErrEmailExists {
			invalidData.Message = "Account already exists with that email"
			a.Views.Signup(res, invalidData)
		} else {
			returnError(res, err)
		}
	} else {
		// we're in
		err := a.Authenticator.SetForAccount(res, user.UserID)
		if err != nil {
			http.Error(res, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(res, req, "/", http.StatusMovedPermanently)
	}
}

func (a *AuthRoutes) handleLogout(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	a.Authenticator.UnsetForAccount(res, req)

	http.Redirect(res, req, "/login", http.StatusFound)
}
