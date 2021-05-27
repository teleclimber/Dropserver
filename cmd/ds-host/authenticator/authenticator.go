package authenticator

import (
	"database/sql"
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
	Config      *domain.RuntimeConfig
	CookieModel interface {
		Get(cookieID string) (domain.Cookie, error)
		Create(domain.Cookie) (string, error)
		UpdateExpires(cookieID string, exp time.Time) error
		Delete(cookieID string) error
	}
}

// SetForAccount creates a cookie and sends it down
// It is for access to the user account only
func (a *Authenticator) SetForAccount(w http.ResponseWriter, userID domain.UserID) error {
	cookie := domain.Cookie{
		UserID:      userID,
		UserAccount: true,
		DomainName:  a.Config.Exec.UserRoutesDomain,
		Expires:     time.Now().Add(cookieExpMinutes * time.Minute)} // set expires on cookie And use that on one sent down.
	cookieID, err := a.CookieModel.Create(cookie)
	if err != nil {
		return err
	}

	a.setCookie(w, cookieID, cookie.Expires, cookie.DomainName)

	return nil
}

// SetForAppspace creates a cookie and sends it down
// It is for access to the appspace only
func (a *Authenticator) SetForAppspace(w http.ResponseWriter, proxyID domain.ProxyID, appspaceID domain.AppspaceID, dom string) (string, error) {
	if dom == "" {
		return "", errors.New("domain can't be blank")
	}

	cookie := domain.Cookie{
		ProxyID:     proxyID,
		AppspaceID:  appspaceID,
		UserAccount: false,
		DomainName:  dom,
		Expires:     time.Now().Add(cookieExpMinutes * time.Minute)} // set expires on cookie And use that on one sent down.
	cookieID, err := a.CookieModel.Create(cookie)
	if err != nil {
		return "", err
	}

	a.setCookie(w, cookieID, cookie.Expires, dom)

	return cookieID, nil
}

// AccountUser middleware sets the user id in context
// if the there is a cookie for user account.
func (a *Authenticator) AccountUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := a.getCookie(r)
		if err != nil && err != errNoCookie {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err == nil {
			if !cookie.UserAccount {
				a.getLogger("AccountUser").
					AddNote(fmt.Sprintf("%v", cookie)).
					Error(errors.New("got a cookie for appspace"))
				a.CookieModel.Delete(cookie.CookieID)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			ctx := domain.CtxWithAuthUserID(r.Context(), cookie.UserID)
			ctx = domain.CtxWithSessionID(ctx, cookie.CookieID)
			r = r.WithContext(ctx)

			a.refreshCookie(w, cookie)
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
		if err != nil && err != errNoCookie {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err == nil {
			if cookie.UserAccount || cookie.AppspaceID != appspace.AppspaceID {
				a.getLogger("AppspaceUserProxyID").
					AddNote(fmt.Sprintf("%v", cookie)).
					AppspaceID(appspace.AppspaceID).
					Error(errors.New("got a cookie for wrong appspace or for user account"))
				a.CookieModel.Delete(cookie.CookieID)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			ctx := domain.CtxWithAppspaceUserProxyID(r.Context(), cookie.ProxyID)
			ctx = domain.CtxWithSessionID(ctx, cookie.CookieID)
			r = r.WithContext(ctx)

			a.refreshCookie(w, cookie)
		}
		next.ServeHTTP(w, r)
	})
}

// Also need to auth via API key eventually

// Unset is the opposite of SetForAccount
// deletes cookie, wipes cookie from DB?
func (a *Authenticator) Unset(w http.ResponseWriter, r *http.Request) {
	cookie, err := a.getCookie(r)
	if err == nil {
		a.CookieModel.Delete(cookie.CookieID)
		a.setCookie(w, cookie.CookieID, time.Now().Add(-100*time.Second), a.Config.Exec.UserRoutesDomain)
	}
}

// I suspect this will be called repeatedly, so maybe separate out the functions:
// - get cookie value
// - get

var errNoCookie = errors.New("no valid cookie")

func (a *Authenticator) getCookie(r *http.Request) (domain.Cookie, error) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return unauthorized
			return domain.Cookie{}, errNoCookie
		}
		// In current version of Go, the only error is ErrNoCookie
		// If we get here log it
		return domain.Cookie{}, err
	}

	cookie, err := a.CookieModel.Get(c.Value)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errNoCookie
		}
		return domain.Cookie{}, err
	}
	if cookie.Expires.Before(time.Now()) {
		return domain.Cookie{}, errNoCookie
	}

	return cookie, nil
}

// refreshCookie updates the expires time on both DB and client
// I don't love this interface. sending the wrong domain is too easy.
func (a *Authenticator) refreshCookie(w http.ResponseWriter, cookie domain.Cookie) {
	expires := time.Now().Add(cookieExpMinutes * time.Minute)

	err := a.CookieModel.UpdateExpires(cookie.CookieID, expires)
	if err != nil {
		// hmmm. If norows just skip it.
		// If something else, log it then ...?
		// This isn't critical.
		// As long as UpdateExpires logs the error, operators can see there is a problem.
		return
	}

	a.setCookie(w, cookie.CookieID, expires, cookie.DomainName)
}

func (a *Authenticator) setCookie(w http.ResponseWriter, cookieID string, expires time.Time, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    cookieID,
		Expires:  expires, // so here we should have sync between cookie store and cookie sent to client
		MaxAge:   int(time.Until(expires).Seconds()),
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
