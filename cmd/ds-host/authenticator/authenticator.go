package authenticator

import (
	"net/http"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

const cookieExpMinutes = 30

// Authenticator contains middleware functions for performing authentication
type Authenticator struct {
	CookieModel domain.CookieModel
	Config      *domain.RuntimeConfig
}

// SetForAccount creates a cookie and sends it down
// It is for access to the user account only
func (a *Authenticator) SetForAccount(res http.ResponseWriter, userID domain.UserID) domain.Error {
	cookie := domain.Cookie{
		UserID:      userID,
		UserAccount: true,
		Expires:     time.Now().Add(cookieExpMinutes * time.Minute)} // set expires on cookie And use that on one sent down.
	cookieID, dsErr := a.CookieModel.Create(cookie)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return dsErr
	}

	a.setCookie(res, cookieID, cookie.Expires)

	return nil
}

// AccountAuthorized tells whether a request should be allowed to proceed or not.
// OK, but it's not clear what the "ForAccount" means? What account are we referring to?
func (a *Authenticator) AccountAuthorized(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) domain.Error {
	cookie, dsErr := a.getCookie(req)
	if dsErr != nil {
		return dsErr
	}

	routeData.Cookie = cookie
	// augment routeData to have convenient logged-in user Fields?
	// apparently Cookie holds something useful there?

	a.refreshCookie(res, cookie.CookieID)

	return nil
}

// UnsetForAccount is the opposite of SetForAccount
// deletes cookie, wipes cookie from DB?
func (a *Authenticator) UnsetForAccount(res http.ResponseWriter, req *http.Request) {
	cookie, dsErr := a.getCookie(req)
	if dsErr == nil {
		a.CookieModel.Delete(cookie.CookieID)
		a.setCookie(res, cookie.CookieID, time.Now().Add(-100*time.Second))
	}
}

// I suspect this will be called repeatedly, so maybe separate out the functions:
// - get cookie value
// - get

func (a *Authenticator) getCookie(req *http.Request) (*domain.Cookie, domain.Error) {
	c, err := req.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return unauthorized
			return nil, dserror.New(dserror.Unauthorized, "no cookie sent")
		}
		// For any other type of error, return a bad request status
		return nil, dserror.New(dserror.BadRequest)
	}

	cookie, dsErr := a.CookieModel.Get(c.Value)
	if dsErr != nil {
		return nil, dsErr //this should be internal error?
	}
	if cookie == nil {
		return nil, dserror.New(dserror.Unauthorized)
	}
	if !cookie.UserAccount {
		return nil, dserror.New(dserror.Unauthorized)
	}
	if cookie.Expires.Before(time.Now()) {
		return nil, dserror.New(dserror.Unauthorized)
	}

	return cookie, nil
}

// refreshCookie updates the expires time on both DB and client
func (a *Authenticator) refreshCookie(res http.ResponseWriter, cookieID string) {
	expires := time.Now().Add(cookieExpMinutes * time.Minute)

	dsErr := a.CookieModel.UpdateExpires(cookieID, expires)
	if dsErr != nil {
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
		Domain:   "user." + a.Config.Server.Host,
		SameSite: http.SameSiteStrictMode,
		//secure: true,	// doesn't work on develop
		// domain?
	})

}
