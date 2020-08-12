package userroutes

// should this be its own isolated package?
// Handle /login /appspace-login /logout
import (
	"net/http"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// AuthRoutes handles all routes related to authentication
type AuthRoutes struct {
	Views               domain.Views
	SettingsModel       domain.SettingsModel
	UserModel           domain.UserModel
	UserInvitationModel domain.UserInvitationModel
	Authenticator       domain.Authenticator
	Validator           domain.Validator
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
	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	switch head {
	case "":
		switch req.Method {
		case http.MethodGet:
			a.Views.Login(res, domain.LoginViewData{})
		case http.MethodPost:
			a.loginPost(res, req, routeData)
		default:
			http.Error(res, "Bad method", http.StatusBadRequest)
		}
	case "appspace":
		// handle appsace login
		http.Error(res, "not implemented", http.StatusNotImplemented)
	default:
		http.Error(res, "Bad path", http.StatusBadRequest)
	}
}

func (a *AuthRoutes) loginPost(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// TODO: CSRF!!

	req.ParseForm()

	invalidLoginMessage := domain.LoginViewData{
		Message: "Login incorrect"}

	email := strings.ToLower(req.Form.Get("email"))
	dsErr := a.Validator.Email(email)
	if dsErr != nil {
		// actually re-render page with generic error
		a.Views.Login(res, invalidLoginMessage)
		return
	}

	invalidLoginMessage.Email = email

	password := req.Form.Get("password")
	dsErr = a.Validator.Password(password)
	if dsErr != nil {
		a.Views.Login(res, invalidLoginMessage)
		return
	}

	user, dsErr := a.UserModel.GetFromEmailPassword(email, password)
	if dsErr != nil {
		code := dsErr.Code()
		if code == dserror.AuthenticationIncorrect || code == dserror.NoRowsInResultSet {
			a.Views.Login(res, invalidLoginMessage)
		} else {
			dsErr.HTTPError(res)
		}
	} else {
		// we're in
		err := a.Authenticator.SetForAccount(res, user.UserID)
		if err != nil {
			http.Error(res, "internal error", http.StatusInternalServerError)
			return
		}

		http.Redirect(res, req, "/", http.StatusFound)
	}
}

func (a *AuthRoutes) getSignup(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	settings, dsErr := a.SettingsModel.Get()
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	viewData := domain.SignupViewData{
		RegistrationOpen: settings.RegistrationOpen}

	a.Views.Signup(res, viewData)
}

func (a *AuthRoutes) postSignup(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// TODO: CSRF!!

	req.ParseForm()

	settings, dsErr := a.SettingsModel.Get()
	if dsErr != nil {
		dsErr.HTTPError(res)
		return
	}

	invalidData := domain.SignupViewData{
		RegistrationOpen: settings.RegistrationOpen}

	email := strings.ToLower(req.Form.Get("email"))
	dsErr = a.Validator.Email(email)
	if dsErr != nil {
		invalidData.Message = "Please use a valid email"
		a.Views.Signup(res, invalidData)
		return
	}
	invalidData.Email = email

	if !settings.RegistrationOpen {
		_, dsErr := a.UserInvitationModel.Get(email)
		if dsErr != nil {
			if dsErr.Code() == dserror.NoRowsInResultSet {
				invalidData.Message = "Sorry, this email is not on the invitation list"
				a.Views.Signup(res, invalidData)
				return
			}
			dsErr.HTTPError(res)
			return
		}
	}

	password := req.Form.Get("password")
	dsErr = a.Validator.Password(password)
	if dsErr != nil {
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

	user, dsErr := a.UserModel.Create(email, password)
	if dsErr != nil {
		code := dsErr.Code()
		if code == dserror.EmailExists {
			invalidData.Message = "Account already exists with that email"
			a.Views.Signup(res, invalidData)
		} else {
			dsErr.HTTPError(res)
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
