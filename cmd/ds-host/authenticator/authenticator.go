package authenticator

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
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

	a.setCookie(res, cookieID, cookie.Expires, a.Config.Exec.UserRoutesDomain)

	return nil
}

// SetForAppspace creates a cookie and sends it down
// It is for access to the appspace only
func (a *Authenticator) SetForAppspace(res http.ResponseWriter, proxyID domain.ProxyID, appspaceID domain.AppspaceID, dom string) (string, error) {
	if dom == "" {
		return "", errors.New("domain can't be blank")
	}

	cookie := domain.Cookie{
		ProxyID:     proxyID,
		AppspaceID:  appspaceID,
		UserAccount: false,
		Expires:     time.Now().Add(cookieExpMinutes * time.Minute)} // set expires on cookie And use that on one sent down.
	cookieID, err := a.CookieModel.Create(cookie)
	if err != nil {
		return "", err
	}

	a.setCookie(res, cookieID, cookie.Expires, dom)

	return cookieID, nil
}

// Authenticate returns a cookie if a valid and verified cookie was included in the request
// I think we can change the signature: don't pass res, don't return error.
// ue separate method to refresh cookie, called from middleware
// errors are logged, and if any then auth is nil.
func (a *Authenticator) Authenticate(req *http.Request) (auth domain.Authentication) {
	cookie, err := a.getCookie(req)
	if err != nil {
		// TODO log it. Or log it in getCookie.
		return
	}
	if cookie == nil {
		return
	}

	// Do we definitely refresh cookie in all cases?
	//a.refreshCookie(res, cookie.CookieID)
	// ^ commenting out until we can sort this out better.

	auth = domain.Authentication{
		Authenticated: true,
		UserID:        cookie.UserID,
		AppspaceID:    cookie.AppspaceID,
		CookieID:      cookie.CookieID,
		UserAccount:   cookie.UserAccount,
		ProxyID:       cookie.ProxyID}

	return auth
}

// AccountUser middleware sets the user id in context
// if the there is a cookie for user account.
func (a *Authenticator) AccountUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := a.getCookie(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if cookie != nil && cookie.UserAccount {
			ctx := domain.CtxWithAuthUserID(r.Context(), cookie.UserID)
			ctx = domain.CtxWithSessionID(ctx, cookie.CookieID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// AppspaceUserProxyID middleware sets the proxy ID for
// the user authenticated for the requested appspace
func (a *Authenticator) AppspaceUserProxyID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appspace, ok := domain.CtxAppspaceData(r.Context())
		if !ok {
			panic("expected appspace data in request context")
		}
		cookie, err := a.getCookie(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if cookie != nil {
			if cookie.UserAccount || cookie.AppspaceID != appspace.AppspaceID {
				a.getLogger("AppspaceUserProxyID").
					AddNote(fmt.Sprintf("%v", cookie)).
					AppspaceID(appspace.AppspaceID).
					Error(errors.New("got a cookie for wrong appspace or for user account"))
				// also cookie should be deleted
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := domain.CtxWithAppspaceUserProxyID(r.Context(), cookie.ProxyID)
			ctx = domain.CtxWithSessionID(ctx, cookie.CookieID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// Also need to auth via API key eventually

// UnsetForAccount is the opposite of SetForAccount
// deletes cookie, wipes cookie from DB?
func (a *Authenticator) UnsetForAccount(res http.ResponseWriter, req *http.Request) {
	cookie, _ := a.getCookie(req)
	if cookie != nil {
		a.CookieModel.Delete(cookie.CookieID)
		a.setCookie(res, cookie.CookieID, time.Now().Add(-100*time.Second), a.Config.Exec.UserRoutesDomain)
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
		// In current version of Go, the only error is ErrNoCookie
		// If we get here log it
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

	a.setCookie(res, cookieID, expires, "domain") // TODO: not sure how to get domain on cookie refresh?
}

func (a *Authenticator) setCookie(res http.ResponseWriter, cookieID string, expires time.Time, domain string) {
	http.SetCookie(res, &http.Cookie{
		Name:     "session_token",
		Value:    cookieID,
		Expires:  expires, // so here we should have sync between cookie store and cookie sent to client
		MaxAge:   int(expires.Sub(time.Now()).Seconds()),
		Domain:   domain,
		SameSite: http.SameSiteStrictMode,
		Secure:   !a.Config.Server.NoSsl,
		Path:     "/",
		HttpOnly: true,
	})
}

func (a *Authenticator) getLogger(note string) *record.DsLogger {
	return record.NewDsLogger("Authenticator", note)
}
