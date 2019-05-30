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
	UserModel     domain.UserModel
	Authenticator domain.Authenticator
	Validator     domain.Validator
}

// ServeHTTP handles all /login routes
func (a *AuthRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	head, _ := shiftpath.ShiftPath(routeData.URLTail)
	switch head {
	case "":
		switch req.Method {
		case http.MethodGet:
			// send page down
			http.Error(res, head+" not implemented", http.StatusNotImplemented)
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

	email := strings.ToLower(req.Form.Get("email"))
	dsErr := a.Validator.Email(email)
	if dsErr != nil {
		// actually re-render page with generic error
		return
	}

	password := strings.ToLower(req.Form.Get("password"))
	dsErr = a.Validator.Password(password)
	if dsErr != nil {
		// re-render page with generic error
		return
	}

	user, dsErr := a.UserModel.GetFromEmailPassword(email, password)
	if dsErr != nil {
		code := dsErr.Code()
		if code == dserror.AuthenticationIncorrect || code == dserror.NoRowsInResultSet {
			// re-render page with email and "email or pw incorrect"
		} else {
			dsErr.HTTPError(res)
		}
	} else {
		// we're in
		a.Authenticator.SetForAccount(res, user.UserID)

		http.Redirect(res, req, "/", http.StatusMovedPermanently)
	}
}
