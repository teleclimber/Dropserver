package userroutes

// should this be its own isolated package?
// Handle /login /appspace-login /logout
import (
	"errors"
	"net/http"
	"net/url"
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
	Authenticator       interface {
		SetForAccount(http.ResponseWriter, domain.UserID) error
		UnsetForAccount(http.ResponseWriter, *http.Request)
	}
	Validator     domain.Validator
	AppspaceLogin interface {
		LogIn(string, domain.UserID) (domain.AppspaceLoginToken, error)
	}
}

// ServeHTTP handles all /login routes
func (a *AuthRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail
	if head == "signup" {
		a.handleSignup(res, req, routeData)
	} else if head == "appspacelogin" {
		a.getAppspaceLogin(res, req, routeData)
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

func (a *AuthRoutes) getAppspaceLogin(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if req.Method != http.MethodGet {
		http.Error(res, "Unsupported method", http.StatusBadRequest)
		return
	}

	asl, err := getAsl(req.URL)
	if err != nil || asl == "" {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	a.Views.AppspaceLogin(res, domain.AppspaceLoginViewData{
		AppspaceLoginToken: asl,
	})
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
	// could be a regular login, or one intended for an appspace
	// if get request has asl= url parameter, then it's appspace login

	asl, err := getAsl(req.URL)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
	}

	if asl != "" {
		if routeData.Authentication != nil {
			a.aslLogin(res, req, routeData.Authentication.UserID, asl)
		} else {
			a.Views.Login(res, domain.LoginViewData{
				AppspaceLoginToken: asl,
			})
		}
	} else {
		if routeData.Authentication != nil && routeData.Authentication.UserAccount {
			http.Redirect(res, req, "/", http.StatusFound)
		} else {
			a.Views.Login(res, domain.LoginViewData{})
		}
	}
}

func (a *AuthRoutes) aslLogin(res http.ResponseWriter, req *http.Request, userID domain.UserID, asl string) {
	// User is currently logged in, so complete login to appspace
	token, err := a.AppspaceLogin.LogIn(asl, userID)
	if err != nil {
		// bad asl token. Basically need to redirect to appspace so a new token is issued and then start over.
		// ah, but with a bad token, you don't know where to redirect to.
		// For now just dump the error
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	url := token.AppspaceURL
	q := url.Query()
	q.Del("dropserver-login-token") // in case there is one already
	q.Add("dropserver-login-token", token.RedirectToken.Token)
	url.RawQuery = q.Encode()
	http.Redirect(res, req, url.String(), http.StatusFound)
}

func (a *AuthRoutes) loginPost(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// TODO: CSRF!!

	req.ParseForm()

	asl := req.Form.Get("asl")

	invalidLoginMessage := domain.LoginViewData{
		AppspaceLoginToken: asl,
		Message:            "Login incorrect"}

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
		// we're in. What we do now depends on whether we have an asl or not.
		if asl != "" {
			a.aslLogin(res, req, user.UserID, asl)
		} else {
			err := a.Authenticator.SetForAccount(res, user.UserID)
			if err != nil {
				http.Error(res, "internal error", http.StatusInternalServerError)
				return
			}
			http.Redirect(res, req, "/", http.StatusFound)
		}
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

func getAsl(u *url.URL) (string, error) {
	aslValues := u.Query()["asl"]
	if len(aslValues) == 0 {
		return "", nil
	}
	if len(aslValues) > 1 {
		return "", errors.New("multiple asl")
	}

	return aslValues[0], nil
}
