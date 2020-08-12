package authenticator

import (
	"net/http"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

const cookieExpMinutes = 30

// Authenticator contains middleware functions for performing authentication
type Authenticator struct {
	CookieModel domain.CookieModel
	Config      *domain.RuntimeConfig
}

// SetForAccount creates a cookie and sends it down
// It is for access to the user account only
func (a *Authenticator) SetForAccount(res http.ResponseWriter, userID domain.UserID) error {
	cookie := domain.Cookie{
		UserID:      userID,
		UserAccount: true,
		Expires:     time.Now().Add(cookieExpMinutes * time.Minute)} // set expires on cookie And use that on one sent down.
	cookieID, err := a.CookieModel.Create(cookie)
	if err != nil {
		return err
	}

	a.setCookie(res, cookieID, cookie.Expires) // need to set domain (and path)

	return nil
}

// Authenticate returns a cookie if a valid and verified cookie was included in the request
func (a *Authenticator) Authenticate(res http.ResponseWriter, req *http.Request) (*domain.Cookie, error) {
	cookie, err := a.getCookie(req)
	if err != nil {
		return nil, err
	}
	if cookie == nil {
		return nil, nil
	}

	// Do we definitely refresh cookie in all cases?
	a.refreshCookie(res, cookie.CookieID)

	return cookie, nil
}

// UnsetForAccount is the opposite of SetForAccount
// deletes cookie, wipes cookie from DB?
func (a *Authenticator) UnsetForAccount(res http.ResponseWriter, req *http.Request) {
	cookie, err := a.getCookie(req)
	if err == nil {
		a.CookieModel.Delete(cookie.CookieID)
		a.setCookie(res, cookie.CookieID, time.Now().Add(-100*time.Second))
	}
}

// I suspect this will be called repeatedly, so maybe separate out the functions:
// - get cookie value
// - get

func (a *Authenticator) getCookie(req *http.Request) (*domain.Cookie, error) {
	c, err := req.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return unauthorized
			return nil, nil
		}
		// For any other type of error, return a bad request status
		return nil, err
	}

	cookie, err := a.CookieModel.Get(c.Value)
	if err != nil {
		return nil, err //this should be internal error?
	}
	if cookie == nil {
		return nil, nil
	}
	if cookie.Expires.Before(time.Now()) {
		return nil, nil
	}

	return cookie, nil
}

// refreshCookie updates the expires time on both DB and client
func (a *Authenticator) refreshCookie(res http.ResponseWriter, cookieID string) {
	expires := time.Now().Add(cookieExpMinutes * time.Minute)

	err := a.CookieModel.UpdateExpires(cookieID, expires)
	if err != nil {
		// hmmm. If norows just skip it.
		// If something else, log it then ...?
		// This isn't critical.
		// As long as UpdateExpires logs the error, operators can see there is a problem.
		return
	}

	a.setCookie(res, cookieID, expires)
}

func (a *Authenticator) setCookie(res http.ResponseWriter, cookieID string, expires time.Time) {
	http.SetCookie(res, &http.Cookie{
		Name:     "session_token",
		Value:    cookieID,
		Expires:  expires, // so here we should have sync between cookie store and cookie sent to client
		MaxAge:   int(expires.Sub(time.Now()).Seconds()),
		Domain:   "user." + a.Config.Server.Host, // TODO: this is obviously not right for appspaces
		SameSite: http.SameSiteStrictMode,
		Secure:   !a.Config.Server.NoSsl,
		Path:     "/",
		HttpOnly: true,
	})
}
