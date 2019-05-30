package authenticator

import (
	"net/http"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// Authenticator contains middleware functions for performing authentication
type Authenticator struct {
	UserModel   domain.UserModel
	CookieModel domain.CookieModel
}

// SetForAccount creates a cookie and sends it down
// It is for access to the user account only
func (a *Authenticator) SetForAccount(res http.ResponseWriter, userID domain.UserID) domain.Error {
	cookie := domain.Cookie{
		UserID:      userID,
		UserAccount: true,
		Expires:     time.Now().Add(120 * time.Second)} // set expires on cookie And use that on one sent down.
	cookieID, dsErr := a.CookieModel.Create(cookie)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return dsErr
	}

	a.setCookie(res, cookieID, cookie.Expires)

	return nil
}

// GetForAccount tells whether a request should be allowed to proceed or not.
func (a *Authenticator) GetForAccount(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) bool {
	c, err := req.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return an unauthorized status
			res.WriteHeader(http.StatusUnauthorized)
			return false
		}
		// For any other type of error, return a bad request status
		res.WriteHeader(http.StatusBadRequest)
		return false
	}

	cookie, dsErr := a.CookieModel.Get(c.Value)
	if dsErr != nil {
		dsErr.HTTPError(res)
		return false
	}
	if cookie == nil {
		res.WriteHeader(http.StatusUnauthorized)
		return false
	}
	if !cookie.UserAccount {
		res.WriteHeader(http.StatusUnauthorized)
		return false
	}
	if cookie.Expires.Before(time.Now()) {
		res.WriteHeader(http.StatusUnauthorized)
		return false
	}

	routeData.Cookie = cookie

	a.refreshCookie(res, cookie.CookieID)

	return true
}

// I suspect this will be called repeatedly, so maybe separate out the functions:
// - get cookie value
// - get

// refreshCookie updates the expires time on both DB and client
func (a *Authenticator) refreshCookie(res http.ResponseWriter, cookieID string) {
	expires := time.Now().Add(120 * time.Second)

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
		MaxAge:   int(time.Now().Sub(expires).Seconds()),
		SameSite: http.SameSiteStrictMode,
		//secure: true,	// doesn't work on develop
		// domain?
	})

}
